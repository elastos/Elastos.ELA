package httpjsonrpc

import (
	"strconv"
	"net/http"

	"Elastos.ELA/common/log"
	. "Elastos.ELA/common/config"
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
	HandleFunc("sendrawtransaction", sendRawTransaction)
	HandleFunc("submitblock", submitBlock)

	// mining interfaces
	HandleFunc("getinfo", getInfo)
	HandleFunc("help", auxHelp)
	HandleFunc("submitauxblock", submitAuxBlock)
	HandleFunc("createauxblock", createAuxBlock)
	HandleFunc("togglecpumining", toggleCpuMining)
	HandleFunc("manualmining", manualCpuMining)

	// TODO: only listen to localhost
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
