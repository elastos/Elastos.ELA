package main

import (
	"os"
	"runtime"
	"time"

	"Elastos.ELA/account"
	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/consensus/pow"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/store/ChainStore"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/net/httpjsonrpc"
	"Elastos.ELA/net/httpnodeinfo"
	"Elastos.ELA/net/httprestful"
	"Elastos.ELA/net/httpwebsocket"
	"Elastos.ELA/net/node"
	"Elastos.ELA/net/protocol"
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

func startConsensus(client account.Client, noder protocol.Noder) {
	httpjsonrpc.Pow = pow.NewPowService(client, "logPow", noder)
	if config.Parameters.PowConfiguration.AutoMining {
		log.Info("Start POW Services")
		go httpjsonrpc.Pow.Start()
	}
}

func main() {
	var client account.Client
	var acct *account.Account
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

	log.Info("2. Open the account")
	client = account.GetClient()
	if client == nil {
		log.Fatal("Can't get local account.")
		goto ERROR
	}
	acct, err = client.GetDefaultAccount()
	if err != nil {
		log.Fatal(err)
		goto ERROR
	}
	httpjsonrpc.Wallet = client
	log.Info("3. Start the P2P networks")
	noder = node.InitNode(acct.PublicKey)
	noder.WaitForSyncFinish()

	httpjsonrpc.RegistRpcNode(noder)
	startConsensus(client, noder)

	handleLogFile()

	log.Info("4. --Start the RPC service")
	go httpjsonrpc.StartRPCServer()
	go httprestful.StartServer(noder)
	go httpwebsocket.StartServer(noder)
	if config.Parameters.HttpInfoStart {
		go httpnodeinfo.StartServer(noder)
	}
	select {}
ERROR:
	os.Exit(1)
}
