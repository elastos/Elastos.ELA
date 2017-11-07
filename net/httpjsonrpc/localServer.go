package httpjsonrpc

import (
	"net/http"
	"strconv"

	. "DNA_POW/common/config"
	"DNA_POW/common/log"
)

const (
	localHost string = "127.0.0.1"
	LocalDir  string = "/local"
)

func StartLocalServer() {
	log.Debug()
	http.HandleFunc(LocalDir, Handle)

	HandleFunc("getneighbor", getNeighbor)
	HandleFunc("getnodestate", getNodeState)
	HandleFunc("startconsensus", startConsensus)
	HandleFunc("stopconsensus", stopConsensus)
	HandleFunc("sendsampletransaction", sendSampleTransaction)
	HandleFunc("setdebuginfo", setDebugInfo)

	HandleFunc("createwallet", createWallet)
	HandleFunc("openwallet", openWallet)
	HandleFunc("closewallet", closeWallet)
	HandleFunc("recoverwallet", recoverWallet)
	HandleFunc("getwalletkey", getWalletKey)
	HandleFunc("maketransfertxn", makeTransferTxn)
	HandleFunc("addaccount", addAccount)
	HandleFunc("deleteaccount", deleteAccount)
	HandleFunc("getbalance", getBalance)

	// TODO: only listen to local host
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpLocalPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
