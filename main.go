// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/config/settings"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/types"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos"
	"github.com/elastos/Elastos.ELA/dpos/account"
	dlog "github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet"
	"github.com/elastos/Elastos.ELA/elanet/routes"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/pow"
	"github.com/elastos/Elastos.ELA/servers"
	"github.com/elastos/Elastos.ELA/servers/httpjsonrpc"
	"github.com/elastos/Elastos.ELA/servers/httpnodeinfo"
	"github.com/elastos/Elastos.ELA/servers/httprestful"
	"github.com/elastos/Elastos.ELA/servers/httpwebsocket"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/utils/elalog"
	"github.com/elastos/Elastos.ELA/utils/signal"

	"github.com/urfave/cli"
)

const (
	// dataPath indicates the path storing the chain data.
	dataPath = "data"

	// logPath indicates the path storing the node log.
	nodeLogPath = "logs/node"

	// checkpointPath indicates the path storing the checkpoint data.
	checkpointPath = "checkpoints"

	// nodePrefix indicates the prefix of node version.
	nodePrefix = "ela-"
)

var (
	// Build version generated when build program.
	Version string

	// The go source code version at build.
	GoVersion string

	// printStateInterval is the interval to print out peer-to-peer network
	// state.
	printStateInterval = time.Minute
)

func main() {
	if err := setupNode().Run(os.Args); err != nil {
		cmdcom.PrintErrorMsg(err.Error())
		os.Exit(1)
	}
}

func setupNode() *cli.App {
	appSettings := settings.NewSettings()

	app := cli.NewApp()
	app.Name = "ela"
	app.Version = Version
	app.HelpName = "ela"
	app.Usage = "ela node for elastos blockchain"
	app.UsageText = "ela [options] [args]"
	app.Flags = []cli.Flag{
		cmdcom.ConfigFileFlag,
		cmdcom.DataDirFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TestNetFlag,
		cmdcom.RegTestFlag,
		cmdcom.InfoPortFlag,
		cmdcom.RestPortFlag,
		cmdcom.WsPortFlag,
		cmdcom.InstantBlockFlag,
		cmdcom.RPCPortFlag,
		cmdcom.RPCIpFlag,
	}
	app.Flags = append(app.Flags, appSettings.Flags()...)
	app.Action = func(c *cli.Context) {
		setupLog(c, appSettings)
		startNode(c, appSettings)
	}
	app.Before = func(c *cli.Context) error {
		appSettings.SetContext(c)
		appSettings.SetupConfig()
		appSettings.InitParamsValue()

		// Use all processor cores.
		runtime.GOMAXPROCS(runtime.NumCPU())

		// Block and transaction processing can cause bursty allocations.  This
		// limits the garbage collector from excessively overallocating during
		// bursts.  This value was arrived at with the help of profiling live
		// usage.
		if appSettings.Params().NodeProfileStrategy ==
			config.MemoryFirst.String() {
			debug.SetGCPercent(10)
		}

		return nil
	}

	return app
}

func startNode(c *cli.Context, st *settings.Settings) {

	// Initialize functions
	functions.GetTransactionByTxType = getTransaction
	functions.GetTransactionByBytes = getTransactionByBytes
	functions.CreateTransaction = createTransaction
	functions.GetTransactionParameters = getTransactionContextParameters

	// Initialize default parameters
	config.DefaultParams = config.GetDefaultParams()

	// Enable http profiling server if requested.
	if st.Config().ProfilePort != 0 {
		go utils.StartPProf(st.Config().ProfilePort)
	}

	flagDataDir := c.String("datadir")
	dataDir := filepath.Join(flagDataDir, dataPath)
	st.Params().CkpManager.SetDataPath(
		filepath.Join(dataDir, checkpointPath))

	var act account.Account
	if st.Config().DPoSConfiguration.EnableArbiter {
		password, err := cmdcom.GetFlagPassword(c)
		if err != nil {
			printErrorAndExit(err)
		}
		act, err = account.Open(password, st.Params().WalletPath)
		if err != nil {
			printErrorAndExit(err)
		}
	}

	log.Infof("Node version: %s", Version)
	log.Info(GoVersion)

	var interrupt = signal.NewInterrupt()

	// fixme remove singleton Ledger
	ledger := blockchain.Ledger{}

	// Initializes the foundation address
	blockchain.FoundationAddress = st.Params().Foundation
	chainStore, err := blockchain.NewChainStore(dataDir, st.Params())
	if err != nil {
		printErrorAndExit(err)
	}
	defer chainStore.Close()
	ledger.Store = chainStore // fixme

	txMemPool := mempool.NewTxPool(st.Params())
	blockMemPool := mempool.NewBlockPool(st.Params())
	blockMemPool.Store = chainStore

	blockchain.DefaultLedger = &ledger // fixme

	committee := crstate.NewCommittee(st.Params())
	ledger.Committee = committee

	arbiters, err := state.NewArbitrators(st.Params(), committee,
		func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := blockchain.DefaultLedger.Store.
				GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		},
		committee.TryUpdateCRMemberInactivity,
		committee.TryRevertCRMemberInactivity,
		committee.TryUpdateCRMemberIllegal,
		committee.TryRevertCRMemberIllegal)
	if err != nil {
		printErrorAndExit(err)
	}
	ledger.Arbitrators = arbiters // fixme

	chain, err := blockchain.New(chainStore, st.Params(), arbiters.State, committee)
	if err != nil {
		printErrorAndExit(err)
	}
	if err := chain.Init(interrupt.C); err != nil {
		printErrorAndExit(err)
	}
	if err := chain.MigrateOldDB(interrupt.C, pgBar.Start,
		pgBar.Increase, dataDir, st.Params()); err != nil {
		printErrorAndExit(err)
	}
	pgBar.Stop()
	ledger.Blockchain = chain // fixme
	blockMemPool.Chain = chain
	arbiters.RegisterFunction(chain.GetHeight, chain.GetBestBlockHash,
		func(height uint32) (*types.Block, error) {
			hash, err := chain.GetBlockHash(height)
			if err != nil {
				return nil, err
			}
			block, err := chainStore.GetFFLDB().GetBlock(hash)
			if err != nil {
				return nil, err
			}
			blockchain.CalculateTxsFee(block.Block)
			return block.Block, nil
		}, chain.UTXOCache.GetTxReference)

	routesCfg := &routes.Config{TimeSource: chain.TimeSource}
	if act != nil {
		routesCfg.PID = act.PublicKeyBytes()
		routesCfg.Addr = fmt.Sprintf("%s:%d",
			st.Params().DPoSIPAddress,
			st.Params().DPoSDefaultPort)
		routesCfg.Sign = act.Sign
	}

	route := routes.New(routesCfg)
	server, err := elanet.NewServer(dataDir, &elanet.Config{
		Chain:          chain,
		ChainParams:    st.Params(),
		PermanentPeers: st.Params().PermanentPeers,
		TxMemPool:      txMemPool,
		BlockMemPool:   blockMemPool,
		Routes:         route,
	}, nodePrefix+Version)
	if err != nil {
		printErrorAndExit(err)
	}
	routesCfg.IsCurrent = server.IsCurrent
	routesCfg.RelayAddr = server.RelayInventory
	blockMemPool.IsCurrent = server.IsCurrent

	committee.RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   chain.UTXOCache.GetTxReference,
		GetUTXO:                          chainStore.GetFFLDB().GetUTXO,
		GetHeight:                        chainStore.GetHeight,
		CreateCRAppropriationTransaction: chain.CreateCRCAppropriationTransaction,
		CreateCRAssetsRectifyTransaction: chain.CreateCRAssetsRectifyTransaction,
		CreateCRRealWithdrawTransaction:  chain.CreateCRRealWithdrawTransaction,
		IsCurrent:                        server.IsCurrent,
		Broadcast: func(msg p2p.Message) {
			server.BroadcastMessage(msg)
		},
		AppendToTxpool: txMemPool.AppendToTxPool,
	})

	var arbitrator *dpos.Arbitrator
	if act != nil {
		dlog.Init(flagDataDir, uint8(st.Config().PrintLevel),
			st.Config().MaxPerLogSize, st.Config().MaxLogsSize)
		arbitrator, err = dpos.NewArbitrator(act, dpos.Config{
			EnableEventLog: true,
			Chain:          chain,
			ChainParams:    st.Params(),
			Arbitrators:    arbiters,
			Server:         server,
			TxMemPool:      txMemPool,
			BlockMemPool:   blockMemPool,
			Broadcast: func(msg p2p.Message) {
				server.BroadcastMessage(msg)
			},
			AnnounceAddr: route.AnnounceAddr,
		})
		if err != nil {
			printErrorAndExit(err)
		}
		routesCfg.OnCipherAddr = arbitrator.OnCipherAddr
		servers.Arbiter = arbitrator
		arbitrator.Start()
		defer arbitrator.Stop()
	}

	servers.Compile = Version
	servers.Config = st.Config()
	servers.ChainParams = st.Params()
	servers.Chain = chain
	servers.Store = chainStore
	servers.TxMemPool = txMemPool
	servers.Server = server
	servers.Arbiters = arbiters
	servers.Pow = pow.NewService(&pow.Config{
		PayToAddr:   st.Config().PowConfiguration.PayToAddr,
		MinerInfo:   st.Config().PowConfiguration.MinerInfo,
		Chain:       chain,
		ChainParams: st.Params(),
		TxMemPool:   txMemPool,
		BlkMemPool:  blockMemPool,
		BroadcastBlock: func(block *types.Block) {
			hash := block.Hash()
			server.RelayInventory(msg.NewInvVect(msg.InvTypeBlock, &hash), block)
		},
		Arbitrators: arbiters,
	})

	st.Params().CkpManager.SetNeedSave(true)
	// initialize producer state after arbiters has initialized.
	if err = chain.InitCheckpoint(interrupt.C, pgBar.Start,
		pgBar.Increase); err != nil {
		printErrorAndExit(err)
	}
	pgBar.Stop()

	// Add small cross chain transactions to transaction pool
	txs, _ := chain.GetDB().GetSmallCrossTransferTxs()
	for _, tx := range txs {
		if err := txMemPool.AppendToTxPoolWithoutEvent(tx); err != nil {
			continue
		}
	}

	log.Info("Start the P2P networks")
	server.Start()
	defer server.Stop()

	log.Info("Start services")
	if st.Config().EnableRPC {
		go httpjsonrpc.StartRPCServer()
	}
	if st.Config().HttpRestStart {
		go httprestful.StartServer()
	}
	if st.Config().HttpWsStart {
		go httpwebsocket.Start()
	}
	if st.Config().HttpInfoStart {
		go httpnodeinfo.StartServer()
	}

	go printSyncState(chain, server)

	waitForSyncFinish(server, interrupt.C)
	if interrupt.Interrupted() {
		return
	}
	log.Info("Start consensus")
	if st.Config().PowConfiguration.AutoMining {
		log.Info("Start POW Services")
		go servers.Pow.Start()
	}
	servers.Pow.ListenForRevert()

	<-interrupt.C
}

func printErrorAndExit(err error) {
	log.Error(err)
	os.Exit(-1)
}

func waitForSyncFinish(server elanet.Server, interrupt <-chan struct{}) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

out:
	for {
		select {
		case <-ticker.C:
			if server.IsCurrent() {
				break out
			}

		case <-interrupt:
			break out
		}
	}
}

func printSyncState(bc *blockchain.BlockChain, server elanet.Server) {
	statlog := elalog.NewBackend(logger.Writer()).Logger("STAT",
		elalog.LevelInfo)

	ticker := time.NewTicker(printStateInterval)
	defer ticker.Stop()

	for range ticker.C {
		var buf bytes.Buffer
		buf.WriteString("-> ")
		buf.WriteString(strconv.FormatUint(uint64(bc.GetHeight()), 10))
		peers := server.ConnectedPeers()
		buf.WriteString(" [")
		for i, p := range peers {
			buf.WriteString(strconv.FormatUint(uint64(p.ToPeer().Height()), 10))
			buf.WriteString(" ")
			buf.WriteString(p.ToPeer().String())
			if i != len(peers)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("]")
		statlog.Info(buf.String())
	}
}

func getTransaction(txType common2.TxType) (txn interfaces.Transaction, err error) {
	// todo refactor me
	switch txType {
	case common2.CoinBase:
		txn = new(transaction2.CoinBaseTransaction)

	case common2.RegisterAsset:
		txn = new(transaction2.RegisterAssetTransaction)

	case common2.IllegalProposalEvidence:
		txn = new(transaction2.IllegalProposalTransaction)

	case common2.IllegalVoteEvidence:
		txn = new(transaction2.IllegalVoteTransaction)

	case common2.IllegalBlockEvidence:
		txn = new(transaction2.IllegalBlockTransaction)

	case common2.IllegalSidechainEvidence:
		txn = new(transaction2.IllegalSideChainTransaction)

	case common2.InactiveArbitrators:
		txn = new(transaction2.InactiveArbitratorsTransaction)

	default:
		return nil, errors.New("invalid transaction type")
	}

	return txn, nil
}

func getTransactionByBytes(r io.Reader) (interfaces.Transaction, error) {
	flagByte, err := common.ReadBytes(r, 1)
	if err != nil {
		return nil, err
	}

	var version common2.TransactionVersion
	var txType common2.TxType
	if common2.TransactionVersion(flagByte[0]) >= common2.TxVersion09 {
		version = common2.TransactionVersion(flagByte[0])
		txTypeBytes, err := common.ReadBytes(r, 1)
		if err != nil {
			return nil, err
		}
		txType = common2.TxType(txTypeBytes[0])
	} else {
		version = common2.TxVersionDefault
		txType = common2.TxType(flagByte[0])
	}

	tx, err := getTransaction(txType)
	if err != nil {
		return nil, err
	}
	tx.SetVersion(version)
	tx.SetTxType(txType)

	return tx, nil
}

func createTransaction(
	version common2.TransactionVersion,
	txType common2.TxType,
	payloadVersion byte,
	payload interfaces.Payload,
	attributes []*common2.Attribute,
	inputs []*common2.Input,
	outputs []*common2.Output,
	lockTime uint32,
	programs []*pg.Program,
) interfaces.Transaction {
	txn, err := functions.GetTransactionByTxType(txType)
	if err != nil {
		fmt.Println(err)
	}
	txn.SetVersion(version)
	txn.SetTxType(txType)
	txn.SetPayloadVersion(payloadVersion)
	txn.SetPayload(payload)
	txn.SetAttributes(attributes)
	txn.SetInputs(inputs)
	txn.SetOutputs(outputs)
	txn.SetLockTime(lockTime)
	txn.SetPrograms(programs)
	return txn
}

func getTransactionContextParameters(
	transaction interfaces.Transaction,
	blockHeight uint32,
	cfg interface{},
	bc interface{}) interfaces.Parameters {
	return &transaction2.TransactionParameters{
		Transaction: transaction,
		BlockHeight: blockHeight,
		Config:      cfg.(*config.Params),
		BlockChain:  bc.(*blockchain.BlockChain),
	}
}
