package httpjsonrpc

import (
	"strconv"
	"net/http"

	"Elastos.ELA/common/log"
	. "Elastos.ELA/common/config"
	. "Elastos.ELA/net/servers"
	"io/ioutil"
	"encoding/json"
	"sync"
)

//an instance of the multiplexer
var mainMux ServeMux

//multiplexer that keeps track of every function to be called on specific rpc call
type ServeMux struct {
	sync.RWMutex
	m map[string]func(map[string]interface{}) map[string]interface{}
}

func init() {
	mainMux.m = make(map[string]func(map[string]interface{}) map[string]interface{})
}

func StartRPCServer() {
	http.HandleFunc("/", Handle)

	// get interfaces
	HandleFunc("getblock", GetBlockByHash)
	HandleFunc("getcurrentheight", GetCurrentHeight)
	HandleFunc("getblockhashbyheight", GetBlockHashByHeight)
	HandleFunc("getconnectioncount", GetConnectionCount)
	HandleFunc("gettxpool", GetTransactionPool)
	HandleFunc("getrawtransaction", GetRawTransaction)
	HandleFunc("getneighbor", GetNeighbor)
	HandleFunc("getnodestate", GetNodeState)

	// set interfaces
	HandleFunc("setloglevel", SetLogLevel)
	HandleFunc("sendrawtransaction", SendRawTransaction)
	HandleFunc("submitblock", SubmitBlock)

	// mining interfaces
	HandleFunc("getinfo", GetInfo)
	HandleFunc("help", AuxHelp)
	HandleFunc("submitauxblock", SubmitAuxBlock)
	HandleFunc("createauxblock", CreateAuxBlock)
	HandleFunc("togglecpumining", ToggleCpuMining)
	HandleFunc("manualmining", ManualCpuMining)

	// TODO: only listen to localhost
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

//a function to register functions to be called for specific rpc calls
func HandleFunc(pattern string, handler func(map[string]interface{}) map[string]interface{}) {
	mainMux.m[pattern] = handler
}

//this is the funciton that should be called in order to answer an rpc call
//should be registered like "http.HandleFunc("/", httpjsonrpc.Handle)"
func Handle(w http.ResponseWriter, r *http.Request) {
	mainMux.RLock()
	defer mainMux.RUnlock()
	//JSON RPC commands should be POSTs
	if r.Method != "POST" {
		log.Warn("HTTP JSON RPC Handle - Method!=\"POST\"")
		return
	}

	//check if there is Request Body to read
	if r.Body == nil {

		log.Warn("HTTP JSON RPC Handle - Request body is nil")
		return
	}

	//read the body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - ioutil.ReadAll: ", err)
		return
	}
	request := make(map[string]interface{})
	err = json.Unmarshal(body, &request)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - json.Unmarshal: ", err)
		return
	}

	//get the corresponding function
	function, ok := mainMux.m[request["method"].(string)]
	if ok {
		response := function(request["params"].(map[string]interface{}))
		data, err := json.Marshal(map[string]interface{}{
			"jsonpc": "2.0",
			"result": response["result"],
			"id":     request["id"],
		})
		if err != nil {
			log.Error("HTTP JSON RPC Handle - json.Marshal: ", err)
			return
		}
		w.Write(data)
	} else {
		//if the function does not exist
		log.Warn("HTTP JSON RPC Handle - No function to call for ", request["method"])
		data, err := json.Marshal(map[string]interface{}{
			"result": nil,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
				"data":    "The called method was not found on the server",
			},
			"id": request["id"],
		})
		if err != nil {
			log.Error("HTTP JSON RPC Handle - json.Marshal: ", err)
			return
		}
		w.Write(data)
	}
}
