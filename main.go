package main

import (
	"os"
	"time"
	"runtime"
	"Elastos.ELA/net/node"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/net/protocol"
	"Elastos.ELA/consensus/pow"
	"Elastos.ELA/common/config"
	"Elastos.ELA/net/httpjsonrpc"
	"Elastos.ELA/net/httprestful"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/core/store/ChainStore"
	"Elastos.ELA/net/httprestful/common"
	"Elastos.ELA/net/httpwebsocket"
	"Elastos.ELA/net/httpnodeinfo"
)

const (
	DefaultMultiCoreNum = 4
)

func init() {
	log.Init(log.Path, log.Stdout)
	var coreNum int
	if config.Parameters.MultiCoreNum > DefaultMultiCoreNum {
		coreNum = int(config.Parameters.MultiCoreNum)
	} else {
		coreNum = DefaultMultiCoreNum
	}
	log.Debug("The Core number is ", coreNum)
	runtime.GOMAXPROCS(coreNum)
}

func handleLogFile() {
	go func() {
		for {
			time.Sleep(6 * time.Second)
			log.Trace("BlockHeight = ", ledger.DefaultLedger.Blockchain.BlockHeight)
			ledger.DefaultLedger.Blockchain.DumpState()
			bc := ledger.DefaultLedger.Blockchain
			log.Info("[", len(bc.Index), len(bc.BlockCache), len(bc.Orphans), "]")
			//ledger.DefaultLedger.Blockchain.DumpState()
			isNeedNewFile := log.CheckIfNeedNewFile()
			if isNeedNewFile {
				log.ClosePrintLog()
				log.Init(log.Path, os.Stdout)
			}
		} //for end
	}()

}

func startConsensus(noder protocol.Noder) {
	httpjsonrpc.Pow = pow.NewPowService("logPow", noder)
	if config.Parameters.PowConfiguration.AutoMining {
		log.Info("Start POW Services")
		go httpjsonrpc.Pow.Start()
	}
}

func main() {
	//var blockChain *ledger.Blockchain
	var err error
	var noder protocol.Noder
	log.Trace("Node version: ", config.Version)
	log.Info("1. BlockChain init")
	ledger.DefaultLedger = new(ledger.Ledger)
	ledger.DefaultLedger.Store, err = ChainStore.NewLedgerStore()
	defer ledger.DefaultLedger.Store.Close()
	if err != nil {
		log.Fatal("open LedgerStore err:", err)
		os.Exit(1)
	}
	ledger.DefaultLedger.Store.InitLedgerStore(ledger.DefaultLedger)
	transaction.TxStore = ledger.DefaultLedger.Store
	_, err = ledger.NewBlockchainWithGenesisBlock()
	if err != nil {
		log.Fatal(err, "BlockChain generate failed")
		goto ERROR
	}

	log.Info("2. Start the P2P networks")
	noder = node.InitNode()
	noder.WaitForSyncFinish()

	httpjsonrpc.RegistRpcNode(noder)
	startConsensus(noder)

	handleLogFile()

	log.Info("4. --Start the RPC service")
	StartServers(noder)
	select {}
ERROR:
	os.Exit(1)
}

func StartServers(noder protocol.Noder) {
	common.SetNode(noder)
	go httpjsonrpc.StartRPCServer()
	go httprestful.StartServer()
	go httpwebsocket.StartServer()
	if config.Parameters.HttpInfoStart {
		go httpnodeinfo.StartServer(noder)
	}
}
