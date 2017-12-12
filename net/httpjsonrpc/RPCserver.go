package httpjsonrpc

import (
	. "DNA_POW/common/config"
	"DNA_POW/common/log"
	"net/http"
	"strconv"
)

func StartRPCServer() {
	http.HandleFunc("/", Handle)

	// get interfaces
	HandleFunc("getbestblockhash", getBestBlockHash)
	HandleFunc("getblock", getBlock)
	HandleFunc("getblockcount", getBlockCount)
	HandleFunc("getblockhash", getBlockHash)
	HandleFunc("getconnectioncount", getConnectionCount)
	HandleFunc("getrawmempool", getRawMemPool)
	HandleFunc("getrawtransaction", getRawTransaction)
	HandleFunc("getneighbor", getNeighbor)
	HandleFunc("getnodestate", getNodeState)
	HandleFunc("getversion", getVersion)

	// set interfaces
	HandleFunc("setdebuginfo", setDebugInfo)
	HandleFunc("sendtoaddress", sendToAddress)
	HandleFunc("sendbatchouttransaction", sendBatchOutTransaction)
	HandleFunc("sendrawtransaction", sendRawTransaction)
	HandleFunc("submitblock", submitBlock)
	HandleFunc("createmultisigtransaction", createMultiSignTransaction)
	HandleFunc("createbatchoutmultisigtransaction", createBatchOutMultiSignTransaction)
	HandleFunc("signmultisigtransaction", signMultiSignTransaction)

	// mining interfaces
	HandleFunc("getinfo", getInfo)
	HandleFunc("help", auxHelp)
	HandleFunc("submitauxblock", submitAuxBlock)
	HandleFunc("createauxblock", createAuxBlock)
	HandleFunc("togglecpumining", toggleCpuMining)
	HandleFunc("discretemining", discreteCpuMining)

	// wallet interfaces
	HandleFunc("addaccount", addAccount)
	HandleFunc("deleteaccount", deleteAccount)

	// TODO: only listen to localhost
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
