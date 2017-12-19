package main

import (
	"os"
	"runtime"
	"time"

	"ELA/account"
	"ELA/common/config"
	"ELA/common/log"
	"ELA/consensus/pow"
	"ELA/core/ledger"
	"ELA/core/store/ChainStore"
	"ELA/core/transaction"
	"ELA/crypto"
	"ELA/net"
	"ELA/net/httpjsonrpc"
	"ELA/net/httpnodeinfo"
	"ELA/net/httprestful"
	"ELA/net/httpwebsocket"
	"ELA/net/protocol"
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

func startConsensus(client account.Client, noder protocol.Noder) bool {
	log.Info("Start POW Services")
	powServices := pow.NewPowService(client, "logPow", noder)
	httpjsonrpc.RegistPowService(powServices)
	if config.Parameters.PowConfiguration.AutoMining {
		go powServices.Start()
	}
	return true

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
	crypto.SetAlg(config.Parameters.EncryptAlg)
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
	noder = net.StartProtocol(acct.PublicKey)
	httpjsonrpc.RegistRpcNode(noder)
	time.Sleep(3 * time.Second)
	noder.StartSync()
	noder.SyncNodeHeight()
	if !startConsensus(client, noder) {
		goto ERROR
	}

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
