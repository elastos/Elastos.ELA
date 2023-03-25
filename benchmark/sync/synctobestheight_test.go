// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package sync

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/config/settings"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/types"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet"
	"github.com/elastos/Elastos.ELA/elanet/netsync"
	"github.com/elastos/Elastos.ELA/elanet/peer"
	"github.com/elastos/Elastos.ELA/elanet/routes"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA/p2p/connmgr"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/pow"
	"github.com/elastos/Elastos.ELA/servers"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/utils/elalog"
	"github.com/elastos/Elastos.ELA/utils/signal"
	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/elastos/Elastos.ELA/wallet"

	"github.com/urfave/cli"
)

const (
	configPath     = "config.json"
	magic          = "201912"
	dataPath       = "data"
	checkpointPath = "checkpoints"

	srcAppName  = "ela"
	srcNodePort = "20086"

	dstNodePort    = "20087"
	dstDataDir     = "temp"
	dstNodeLogPath = "logs/node"
)

var (
	srcApp = startSrcNode()
	srcDir string

	dstSettings = initDstSettings()
	dstContext  *cli.Context
	logger      *log.Logger
)

func Benchmark_Sync_ToBestHeight(b *testing.B) {
	startDstNode()
	endNodes()
}

func startDstNode() {
	// Enable profiling server if requested.
	if dstSettings.ProfilePort != 0 {
		go utils.StartPProf(dstSettings.ProfilePort,
			dstSettings.ProfileHost)
	}

	flagDataDir := dstContext.String("datadir")
	dataDir := filepath.Join(flagDataDir, dataPath)

	ckpManager := checkpoint.NewManager(dstSettings)
	ckpManager.SetDataPath(filepath.Join(dataDir, checkpointPath))

	var interrupt = signal.NewInterrupt()

	ledger := blockchain.Ledger{}

	// Initializes the foundation address
	blockchain.FoundationAddress = *dstSettings.FoundationProgramHash

	chainStore, err := blockchain.NewChainStore(dataDir, dstSettings)
	if err != nil {
		logger.Error(err)
		return
	}
	defer chainStore.Close()
	ledger.Store = chainStore

	txMemPool := mempool.NewTxPool(dstSettings, ckpManager)
	blockMemPool := mempool.NewBlockPool(dstSettings)
	blockMemPool.Store = chainStore

	blockchain.DefaultLedger = &ledger

	committee := crstate.NewCommittee(dstSettings, ckpManager)
	ledger.Committee = committee

	arbiters, err := state.NewArbitrators(dstSettings, committee,
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
		}, nil, nil,
		nil, nil, nil, nil,
		ckpManager)
	if err != nil {
		logger.Error(err)
		return
	}
	ledger.Arbitrators = arbiters

	chain, err := blockchain.New(chainStore, dstSettings, arbiters.State, committee, ckpManager)
	if err != nil {
		logger.Error(err)
		return
	}
	if err := chain.Init(interrupt.C); err != nil {
		logger.Error(err)
		return
	}
	if err := chain.MigrateOldDB(interrupt.C, func(uint32) {},
		func() {}, dataDir, dstSettings); err != nil {
		logger.Error(err)
		return
	}
	ledger.Blockchain = chain
	blockMemPool.Chain = chain
	arbiters.RegisterFunction(chain.GetHeight,
		func() *common.Uint256 { return chain.BestChain.Hash },
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

	route := routes.New(routesCfg)
	server, err := elanet.NewServer(dataDir, &elanet.Config{
		Chain:          chain,
		ChainParams:    dstSettings,
		PermanentPeers: dstSettings.PermanentPeers,
		TxMemPool:      txMemPool,
		BlockMemPool:   blockMemPool,
		Routes:         route,
	}, "")
	if err != nil {
		logger.Error(err)
		return
	}
	routesCfg.IsCurrent = server.IsCurrent
	routesCfg.RelayAddr = server.RelayInventory
	blockMemPool.IsCurrent = server.IsCurrent

	committee.RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   chain.UTXOCache.GetTxReference,
		GetUTXO:                          chainStore.GetFFLDB().GetUTXO,
		GetHeight:                        chainStore.GetHeight,
		CreateCRAppropriationTransaction: chain.CreateCRCAppropriationTransaction,
		IsCurrent:                        server.IsCurrent,
		Broadcast: func(msg p2p.Message) {
			server.BroadcastMessage(msg)
		},
		AppendToTxpool: txMemPool.AppendToTxPool,
	})

	wal := wallet.NewWallet()
	wallet.Store = chainStore
	wallet.ChainParam = dstSettings
	wallet.Chain = chain

	ckpManager.Register(wal)

	servers.Compile = "benchmark"
	servers.ChainParams = dstSettings
	servers.Chain = chain
	servers.Store = chainStore
	servers.TxMemPool = txMemPool
	servers.Server = server
	servers.Arbiters = arbiters
	servers.Wallet = wal
	servers.Pow = pow.NewService(&pow.Config{
		PayToAddr:   dstSettings.PowConfiguration.PayToAddr,
		MinerInfo:   dstSettings.PowConfiguration.MinerInfo,
		Chain:       chain,
		ChainParams: dstSettings,
		TxMemPool:   txMemPool,
		BlkMemPool:  blockMemPool,
		BroadcastBlock: func(block *types.Block) {
			hash := block.Hash()
			server.RelayInventory(msg.NewInvVect(msg.InvTypeBlock, &hash), block)
		},
		Arbitrators: arbiters,
	})

	// initialize producer state after arbiters has initialized.
	if err = chain.InitCheckpoint(interrupt.C, func(uint32) {},
		func() {}); err != nil {
		logger.Error(err)
		return
	}

	log.Info("Start the P2P networks")
	server.Start()
	defer server.Stop()

	go printSyncState(chain, server)
	waitForSyncFinish(server, interrupt.C)
}

func startSrcNode() *exec.Cmd {
	_, filename, _, _ := runtime.Caller(0)
	srcDir = path.Dir(filename)

	app := exec.Command(path.Join(srcDir, srcAppName), getSrcRunArgs()...)
	if err := app.Start(); err != nil {
		fmt.Println(err)
		return nil
	}

	return app
}

func endNodes() {
	os.RemoveAll(dstDataDir)
	if srcApp != nil && srcApp.Process != nil {
		srcApp.Process.Kill()
	}
}

func getPeerAddr(port string) string {
	return "127.0.0.1" + ":" + port
}

func getSrcRunArgs() []string {
	return []string{
		"--magic", magic,
		"--conf", path.Join(srcDir, configPath),
		"--datadir", path.Join(srcDir, test.DataDir),
		"--peers", getPeerAddr(dstNodePort),
		"--port", srcNodePort,
		"--arbiter", "false",
		"--server", "false",
		"--automining", "false",
	}
}

func initDstSettings() *config.Configuration {
	setting := settings.NewSettings()
	config := setting.SetupConfig(true)
	setupLog(config)
	return config
}

func waitForSyncFinish(server elanet.Server, interrupt <-chan struct{}) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

out:
	for {
		select {
		case <-ticker.C:
			if len(server.ConnectedPeers()) > 0 && server.IsCurrent() {
				break out
			}

		case <-interrupt:
			break out
		}
	}
}

func setupLog(s *config.Configuration) {
	flagDataDir := config.DataDir
	if s.DataDir != "" {
		flagDataDir = s.DataDir
	}
	path := filepath.Join(flagDataDir, dstNodeLogPath)
	logger = log.NewDefault(path, uint8(s.PrintLevel),
		s.MaxPerLogSize, s.MaxLogsSize)

	addrmgr.UseLogger(logger)
	connmgr.UseLogger(logger)
	netsync.UseLogger(logger)
	peer.UseLogger(logger)
	routes.UseLogger(logger)
	elanet.UseLogger(logger)
	state.UseLogger(logger)
	crstate.UseLogger(logger)
}

func printSyncState(bc *blockchain.BlockChain, server elanet.Server) {
	statlog := elalog.NewBackend(logger.Writer()).Logger("STAT",
		elalog.LevelInfo)

	ticker := time.NewTicker(time.Second * 20)
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
