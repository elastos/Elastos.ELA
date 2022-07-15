// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/config/settings"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/types"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos"
	"github.com/elastos/Elastos.ELA/dpos/account"
	dlog "github.com/elastos/Elastos.ELA/dpos/log"
	msg2 "github.com/elastos/Elastos.ELA/dpos/p2p/msg"
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
)

const (
	// dataPath indicates the path storing the chain data.
	dataPath = "data"

	// nodeLogPath indicates the path storing the node log.
	nodeLogPath = "logs/node"

	// checkpointPath indicates the path storing the checkpoint data.
	checkpointPath = "checkpoints"

	// nodePrefix indicates the prefix of node version.
	nodePrefix = "ela-"
)

var (
	// Version generated when build program.
	Version string

	// GoVersion version at build.
	GoVersion string

	// The interval to print out peer-to-peer network state.
	printStateInterval = time.Minute
)

func main() {

	// Setting config
	setting := settings.NewSettings()
	config := setting.SetupConfig(true, "Copyright (c) 2017-"+
		fmt.Sprint(time.Now().Year())+" The Elastos Foundation", nodePrefix+Version+GoVersion)

	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// This value was arrived at with the help of profiling live usage.
	if config.MemoryFirst {
		debug.SetGCPercent(10)
	}

	// Init logger
	setupLog(config)

	// Debug
	json.MarshalIndent(config, "", "\t")

	// Start Node
	startNode(config)
}

func startNode(cfg *config.Configuration) {
	log.Infof("Node version: %s, %s, %s", Version, GoVersion, cfg.ActiveNet)
	if cfg.ProfilePort != 0 {
		go utils.StartPProf(cfg.ProfilePort, cfg.ProfileHost)
	}

	flagDataDir := config.DataDir
	if cfg.DataDir != "" {
		flagDataDir = cfg.DataDir
	}
	dataDir := filepath.Join(flagDataDir, dataPath)

	ckpManager := checkpoint.NewManager(cfg)
	ckpManager.SetDataPath(filepath.Join(dataDir, checkpointPath))

	var acc account.Account
	if cfg.DPoSConfiguration.EnableArbiter {
		var err error
		var password []byte
		if cfg.Password != "" {
			password = []byte(cfg.Password)
		} else {
			password, err = utils.GetPassword()
		}
		if err != nil {
			printErrorAndExit(err)
		}
		acc, err = account.Open(password, cfg.WalletPath)
		if err != nil {
			printErrorAndExit(err)
		}
	}
	var interrupt = signal.NewInterrupt()

	// fixme remove singleton Ledger
	ledger := blockchain.Ledger{}

	// Initializes the foundation address
	blockchain.FoundationAddress = *cfg.FoundationProgramHash
	chainStore, err := blockchain.NewChainStore(dataDir, cfg)
	if err != nil {
		printErrorAndExit(err)
	}

	ledger.Store = chainStore // fixme

	txMemPool := mempool.NewTxPool(cfg, ckpManager)
	blockMemPool := mempool.NewBlockPool(cfg)
	blockMemPool.Store = chainStore

	blockchain.DefaultLedger = &ledger // fixme

	committee := crstate.NewCommittee(cfg, ckpManager)
	ledger.Committee = committee

	arbiters, err := state.NewArbitrators(cfg, committee, ledger.GetAmount,
		committee.TryUpdateCRMemberInactivity,
		committee.TryRevertCRMemberInactivity,
		committee.TryUpdateCRMemberIllegal,
		committee.TryRevertCRMemberIllegal,
		committee.UpdateCRInactivePenalty,
		committee.RevertUpdateCRInactivePenalty,
		ckpManager,
	)
	if err != nil {
		printErrorAndExit(err)
	}
	ledger.Arbitrators = arbiters // fixme

	chain, err := blockchain.New(chainStore, cfg,
		arbiters.State, committee, ckpManager)
	if err != nil {
		printErrorAndExit(err)
	}
	if err = chain.Init(interrupt.C); err != nil {
		printErrorAndExit(err)
	}
	if err = chain.MigrateOldDB(interrupt.C, pgBar.Start,
		pgBar.Increase, dataDir, cfg); err != nil {
		printErrorAndExit(err)
	}
	var chainStoreEx blockchain.IChainStoreExtend
	chainStoreEx, err = blockchain.NewChainStoreEx(chain, chainStore, filepath.Join(dataDir, "ext"))
	if err != nil {
		printErrorAndExit(err)
	}
	defer chainStore.Close()
	defer chainStoreEx.CloseEx()
	pgBar.Stop()

	ledger.Blockchain = chain // fixme
	blockMemPool.Chain = chain
	arbiters.RegisterFunction(chain.GetHeight, chain.GetBestBlockHash,
		chain.GetBlock, chain.UTXOCache.GetTxReference)

	routesCfg := &routes.Config{TimeSource: chain.TimeSource}
	if acc != nil {
		routesCfg.PID = acc.PublicKeyBytes()
		routesCfg.Addr = net.JoinHostPort(cfg.DPoSConfiguration.IPAddress,
			strconv.FormatUint(uint64(cfg.DPoSConfiguration.DPoSPort), 10))
		routesCfg.Sign = acc.Sign
	}

	route := routes.New(routesCfg)
	netServer, err := elanet.NewServer(dataDir, &elanet.Config{
		Chain:          chain,
		ChainParams:    cfg,
		PermanentPeers: cfg.PermanentPeers,
		TxMemPool:      txMemPool,
		BlockMemPool:   blockMemPool,
		Routes:         route,
	}, nodePrefix+Version)
	if err != nil {
		printErrorAndExit(err)
	}
	routesCfg.IsCurrent = netServer.IsCurrent
	routesCfg.RelayAddr = netServer.RelayInventory
	blockMemPool.IsCurrent = netServer.IsCurrent

	arbiters.State.RegisterFuncitons(&state.StateFuncsConfig{
		GetHeight: chainStore.GetHeight,
		IsCurrent: netServer.IsCurrent,
		Broadcast: func(msg p2p.Message) {
			netServer.BroadcastMessage(msg)
		},
		AppendToTxpool:                      txMemPool.AppendToTxPool,
		CreateDposV2RealWithdrawTransaction: chain.CreateDposV2RealWithdrawTransaction,
		CreateVotesRealWithdrawTransaction:  chain.CreateVotesRealWithdrawTransaction,
	})

	if acc != nil {
		dlog.Init(flagDataDir, uint8(cfg.PrintLevel), cfg.MaxPerLogSize, cfg.MaxLogsSize)
		arbitrator, err := dpos.NewArbitrator(acc, dpos.Config{
			EnableEventLog: true,
			Chain:          chain,
			ChainParams:    cfg,
			Arbitrators:    arbiters,
			Server:         netServer,
			TxMemPool:      txMemPool,
			BlockMemPool:   blockMemPool,
			Broadcast: func(msg p2p.Message) {
				netServer.BroadcastMessage(msg)
			},
			AnnounceAddr: route.AnnounceAddr,
			NodeVersion:  nodePrefix + Version,
			Addr:         routesCfg.Addr,
		})
		if err != nil {
			printErrorAndExit(err)
		}
		routesCfg.OnCipherAddr = arbitrator.OnCipherAddr
		servers.Arbiter = arbitrator
		arbitrator.Start()
		defer arbitrator.Stop()
	}

	committee.RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   chain.UTXOCache.GetTxReference,
		GetUTXO:                          chainStore.GetFFLDB().GetUTXO,
		GetHeight:                        chainStore.GetHeight,
		CreateCRAppropriationTransaction: chain.CreateCRCAppropriationTransaction,
		CreateCRAssetsRectifyTransaction: chain.CreateCRAssetsRectifyTransaction,
		CreateCRRealWithdrawTransaction:  chain.CreateCRRealWithdrawTransaction,
		IsCurrent:                        netServer.IsCurrent,
		Broadcast: func(msg p2p.Message) {
			netServer.BroadcastMessage(msg)
		},
		AppendToTxpool:     txMemPool.AppendToTxPool,
		GetCurrentArbiters: arbiters.GetCurrentArbitratorKeys,
	})

	servers.Compile = Version
	servers.ChainParams = cfg
	servers.Chain = chain
	servers.Store = chainStore
	servers.TxMemPool = txMemPool
	servers.Server = netServer
	servers.Arbiters = arbiters
	servers.Pow = pow.NewService(&pow.Config{
		PayToAddr:   cfg.PowConfiguration.PayToAddr,
		MinerInfo:   cfg.PowConfiguration.MinerInfo,
		Chain:       chain,
		ChainParams: cfg,
		TxMemPool:   txMemPool,
		BlkMemPool:  blockMemPool,
		BroadcastBlock: func(block *types.Block) {
			hash := block.Hash()
			netServer.RelayInventory(msg.NewInvVect(msg.InvTypeBlock, &hash), block)
		},
		Arbitrators: arbiters,
	})

	ckpManager.SetNeedSave(true)
	// initialize producer state after arbiters has initialized.
	if err = chain.InitCheckpoint(interrupt.C, pgBar.Start,
		pgBar.Increase); err != nil {
		printErrorAndExit(err)
	}
	pgBar.Stop()

	// todo remove me
	if chain.GetHeight() > cfg.DPoSV2StartHeight {
		msg2.SetPayloadVersion(msg2.DPoSV2Version)
	}

	// Add small cross chain transactions to transaction pool
	txs, _ := chain.GetDB().GetSmallCrossTransferTxs()
	for _, tx := range txs {
		if err := txMemPool.AppendToTxPoolWithoutEvent(tx); err != nil {
			continue
		}
	}

	log.Info("Start the P2P networks")
	netServer.Start()
	defer netServer.Stop()

	log.Info("Start services")
	if cfg.EnableRPC {
		go httpjsonrpc.StartRPCServer()
	}
	if cfg.HttpRestStart {
		go httprestful.StartServer()
	}
	if cfg.HttpWsStart {
		go httpwebsocket.Start()
	}
	if cfg.HttpInfoStart {
		go httpnodeinfo.StartServer()
	}

	go printSyncState(chain, netServer)

	waitForSyncFinish(netServer, interrupt.C)
	if interrupt.Interrupted() {
		return
	}
	log.Info("Start consensus")
	if cfg.PowConfiguration.AutoMining {
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
