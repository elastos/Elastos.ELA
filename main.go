package main

import (
	"os"
	"runtime"
	"time"

	"DNA_POW/account"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/consensus/dbft"
	"DNA_POW/consensus/pow"
	"DNA_POW/core/ledger"
	"DNA_POW/core/store/ChainStore"
	"DNA_POW/core/transaction"
	"DNA_POW/crypto"
	"DNA_POW/net"
	"DNA_POW/net/httpjsonrpc"
	"DNA_POW/net/httpnodeinfo"
	"DNA_POW/net/httprestful"
	"DNA_POW/net/httpwebsocket"
	"DNA_POW/net/protocol"
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

func handleLogFile(consensus string) {
	switch consensus {

	case "pow":
		/* TODO */
		fallthrough
	case "dbft":
		go func() {
			for {
				time.Sleep(dbft.GenBlockTime)
				log.Trace("BlockHeight = ", ledger.DefaultLedger.Blockchain.BlockHeight)
				log.Trace("BlockHeight = ", ledger.DefaultLedger.Store.GetHeight())
				//ledger.DefaultLedger.Blockchain.DumpState()
				isNeedNewFile := log.CheckIfNeedNewFile()
				if isNeedNewFile == true {
					log.ClosePrintLog()
					log.Init(log.Path, os.Stdout)
				}
			} //for end
		}()
	}
}

func startConsensus(client account.Client, noder protocol.Noder) bool {
	if protocol.SERVICENODENAME != config.Parameters.NodeType {
		if config.Parameters.ConsensusType == "pow" &&
			config.Parameters.PowConfiguration.Switch == "enable" {
			log.Info("Start POW Services")
			powServices := pow.NewPowService(client, "logPow", noder)
			httpjsonrpc.RegistPowService(powServices)
			isAuxPow := config.Parameters.PowConfiguration.CoMining
			if !isAuxPow {
				go powServices.Start()
			} else {
				//aux pow
			}
			handleLogFile("pow")
			time.Sleep(5 * time.Second)
			return true
		} else if config.Parameters.ConsensusType == "dbft" {
			log.Info("5. Start DBFT Services")
			dbftServices := dbft.NewDbftService(client, "logdbft", noder)
			httpjsonrpc.RegistDbftService(dbftServices)
			go dbftServices.Start()
			handleLogFile("dbft")
			time.Sleep(5 * time.Second)
			return true
		} else {
			log.Fatal("Start consensus ERROR,consensusType is: ", config.Parameters.ConsensusType)
			return false
		}
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

	if len(config.Parameters.BookKeepers) < account.DefaultBookKeeperCount {
		log.Fatal("At least ", account.DefaultBookKeeperCount, " BookKeepers should be set at config.json")
		os.Exit(1)
	}

	log.Info("0. Loading the Ledger")
	ledger.DefaultLedger = new(ledger.Ledger)
	ledger.DefaultLedger.Store, err = ChainStore.NewLedgerStore()
	defer ledger.DefaultLedger.Store.Close()
	if err != nil {
		log.Fatal("open LedgerStore err:", err)
		os.Exit(1)
	}
	//ledger.DefaultLedger.State = ledger.NewState(ledger.DefaultLedger)
	ledger.DefaultLedger.Store.InitLedgerStore(ledger.DefaultLedger)
	transaction.TxStore = ledger.DefaultLedger.Store
	crypto.SetAlg(config.Parameters.EncryptAlg)

	log.Info("1. BlockChain init")
	ledger.StandbyBookKeepers = account.GetBookKeepers()
	log.Trace("1.NewBlockchainWithGenesisBlock")
	//blockChain, err = ledger.NewBlockchainWithGenesisBlock(ledger.StandbyBookKeepers)
	_, err = ledger.NewBlockchainWithGenesisBlock(ledger.StandbyBookKeepers)
	if err != nil {
		log.Fatal(err, "  BlockChain generate failed")
		goto ERROR
	}
	//	ledger.DefaultLedger.Blockchain = blockChain

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
	log.Info("The Node's PublicKey ", acct.PublicKey)
	log.Info("3. Start the P2P networks")
	noder = net.StartProtocol(acct.PublicKey)
	log.Info("3.1 Start the P2P networks")
	httpjsonrpc.RegistRpcNode(noder)
	log.Info("3.2 Start the P2P networks")
	time.Sleep(10 * time.Second)
	noder.SyncNodeHeight()
	log.Info("3.3 Start the P2P networks")
	noder.WaitForFourPeersStart()
	log.Info("3.4 Start the P2P networks")
	noder.WaitForSyncBlkFinish()
	if !startConsensus(client, noder) {
		goto ERROR
	}

	log.Info("4. --Start the RPC interface")
	go httpjsonrpc.StartRPCServer()
	go httpjsonrpc.StartLocalServer()
	go httprestful.StartServer(noder)
	go httpwebsocket.StartServer(noder)
	if config.Parameters.HttpInfoStart {
		go httpnodeinfo.StartServer(noder)
	}
	select {}
ERROR:
	os.Exit(1)
}
