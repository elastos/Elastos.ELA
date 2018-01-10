package httpjsonrpc

import (
	"strconv"
	"net/http"

	"Elastos.ELA/common/log"
	. "Elastos.ELA/common/config"
	. "Elastos.ELA/net/servers"
	"io/ioutil"
	"encoding/json"
)

//an instance of the multiplexer
var mainMux map[string]func(map[string]interface{}) map[string]interface{}

func StartRPCServer() {
	mainMux = make(map[string]func(map[string]interface{}) map[string]interface{})

	http.HandleFunc("/", Handle)

	mainMux["getblock"] 			= GetBlockByHash
	mainMux["getcurrentheight"] 	= GetCurrentHeight
	mainMux["getblockhashbyheight"] = GetBlockHashByHeight
	mainMux["getconnectioncount"] 	= GetConnectionCount
	mainMux["gettxpool"] 			= GetTransactionPool
	mainMux["getrawtransaction"] 	= GetRawTransaction
	mainMux["getneighbor"] 			= GetNeighbor
	mainMux["getnodestate"] 		= GetNodeState
	mainMux["setloglevel"] 			= SetLogLevel
	mainMux["sendrawtransaction"] 	= SendRawTransaction
	mainMux["submitblock"] 			= SubmitBlock

	// mining interfaces
	mainMux["getinfo"] 				= GetInfo
	mainMux["help"] 				= AuxHelp
	mainMux["submitauxblock"] 		= SubmitAuxBlock
	mainMux["createauxblock"] 		= CreateAuxBlock
	mainMux["togglecpumining"] 		= ToggleCpuMining
	mainMux["manualmining"] 		= ManualCpuMining

	// TODO: only listen to localhost
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

//this is the funciton that should be called in order to answer an rpc call
//should be registered like "http.AddMethod("/", httpjsonrpc.Handle)"
func Handle(w http.ResponseWriter, r *http.Request) {
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
	function, ok := mainMux[request["method"].(string)]
	if ok {
		response := function(request["params"].(map[string]interface{}))
		data, err := json.Marshal(map[string]interface{}{
			"jsonpc": "2.0",
			"code":   response["Error"],
			"result": response["Result"],
		})
		if err != nil {
			log.Error("HTTP JSON RPC Handle - json.Marshal: ", err)
			return
		}
		w.Write(data)
	} else {
		//if the function does not exist
		log.Warn("HTTP JSON RPC Handle - No function to call for ", request["method"])
		data, _ := json.Marshal(map[string]interface{}{
			"result": nil,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
				"data":    "The method" + request["method"].(string) + "was not found",
			},
		})
		w.Write(data)
	}
}
