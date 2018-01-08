package httprestful

import (
	. "Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	. "Elastos.ELA/net/servers"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	. "Elastos.ELA/errors"
)

const (
	Api_Getconnectioncount  = "/api/v1/node/connectioncount"
	Api_GetblockTxsByHeight = "/api/v1/block/transactions/height/:height"
	Api_Getblockbyheight    = "/api/v1/block/details/height/:height"
	Api_Getblockbyhash      = "/api/v1/block/details/hash/:hash"
	Api_Getblockheight      = "/api/v1/block/height"
	Api_Getblockhash        = "/api/v1/block/hash/:height"
	Api_GetTotalIssued      = "/api/v1/totalissued/:assetid"
	Api_Gettransaction      = "/api/v1/transaction/:hash"
	Api_Getasset            = "/api/v1/asset/:hash"
	Api_GetBalanceByAddr    = "/api/v1/asset/balances/:addr"
	Api_GetBalancebyAsset   = "/api/v1/asset/balance/:addr/:assetid"
	Api_GetUTXObyAsset      = "/api/v1/asset/utxo/:addr/:assetid"
	Api_GetUTXObyAddr       = "/api/v1/asset/utxos/:addr"
	Api_SendRawTx           = "/api/v1/transaction"
	Api_GetTransactionPool  = "/api/v1/transactionpool"
	Api_SendRcdTxByTrans    = "/api/v1/custom/transaction/record"
	Api_GetStateUpdate      = "/api/v1/stateupdate/:namespace/:key"
	Api_WebsocketState      = "/api/v1/config/websocket/state"
	Api_Restart             = "/api/v1/restart"
	Api_GetContract         = "/api/v1/contract/:hash"
)

var node = NodeForServers

const TlsPort = 443

type Action struct {
	sync.RWMutex
	name    string
	handler func(map[string]interface{}) map[string]interface{}
}

type restServer struct {
	router   *Router
	listener net.Listener
	server   *http.Server
	postMap  map[string]Action
	getMap   map[string]Action
}

type ApiServer interface {
	Start() error
	Stop()
}

func StartServer() {
	rest := InitRestServer()
	rest.Start()
}

func InitRestServer() ApiServer {
	rt := &restServer{}
	rt.router = &Router{}
	rt.initializeMethod()
	rt.initGetHandler()
	rt.initPostHandler()
	return rt
}

func (rt *restServer) Start() error {
	if Parameters.HttpRestPort == 0 {
		log.Fatal("Not configure HttpRestPort port ")
		return nil
	}

	if Parameters.HttpRestPort%1000 == TlsPort {
		var err error
		rt.listener, err = rt.initTlsListen()
		if err != nil {
			log.Error("Https Cert: ", err.Error())
			return err
		}
	} else {
		var err error
		rt.listener, err = net.Listen("tcp", ":"+strconv.Itoa(Parameters.HttpRestPort))
		if err != nil {
			log.Fatal("net.Listen: ", err.Error())
			return err
		}
	}
	rt.server = &http.Server{Handler: rt.router}
	err := rt.server.Serve(rt.listener)

	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
		return err
	}

	return nil
}

func (rt *restServer) initializeMethod() {

	getMethodMap := map[string]Action{
		Api_Getconnectioncount:  {name: "getconnectioncount", handler: GetConnectionCount},
		Api_GetblockTxsByHeight: {name: "getblocktransactionsbyheight", handler: GetBlockTxsByHeight},
		Api_Getblockbyheight:    {name: "getblockbyheight", handler: GetBlockByHeight},
		Api_Getblockbyhash:      {name: "getblockbyhash", handler: GetBlockByHash},
		Api_Getblockheight:      {name: "getblockheight", handler: GetBlockHeight},
		Api_Getblockhash:        {name: "getblockhash", handler: GetBlockHash},
		Api_GetTransactionPool:  {name: "gettransactionpool", handler: GetTransactionPool},
		Api_Gettransaction:      {name: "gettransaction", handler: GetTransactionByHash},
		Api_Getasset:            {name: "getasset", handler: GetAssetByHash},
		Api_GetUTXObyAddr:       {name: "getutxobyaddr", handler: GetUnspends},
		Api_GetUTXObyAsset:      {name: "getutxobyasset", handler: GetUnspendOutput},
		Api_GetBalanceByAddr:    {name: "getbalancebyaddr", handler: GetBalanceByAddr},
		Api_GetBalancebyAsset:   {name: "getbalancebyasset", handler: GetBalanceByAsset},
		Api_Restart:             {name: "restart", handler: rt.Restart},
	}

	postMethodMap := map[string]Action{
		Api_SendRawTx: {name: "sendrawtransaction", handler: SendRawTransaction},
	}
	rt.postMap = postMethodMap
	rt.getMap = getMethodMap
}

func (rt *restServer) getPath(url string) string {

	if strings.Contains(url, strings.TrimRight(Api_GetblockTxsByHeight, ":height")) {
		return Api_GetblockTxsByHeight
	} else if strings.Contains(url, strings.TrimRight(Api_Getblockbyheight, ":height")) {
		return Api_Getblockbyheight
	} else if strings.Contains(url, strings.TrimRight(Api_Getblockhash, ":height")) {
		return Api_Getblockhash
	} else if strings.Contains(url, strings.TrimRight(Api_Getblockbyhash, ":hash")) {
		return Api_Getblockbyhash
	} else if strings.Contains(url, strings.TrimRight(Api_GetTotalIssued, ":assetid")) {
		return Api_GetTotalIssued
	} else if strings.Contains(url, strings.TrimRight(Api_Gettransaction, ":hash")) {
		return Api_Gettransaction
	} else if strings.Contains(url, strings.TrimRight(Api_GetContract, ":hash")) {
		return Api_GetContract
	} else if strings.Contains(url, strings.TrimRight(Api_GetBalanceByAddr, ":addr")) {
		return Api_GetBalanceByAddr
	} else if strings.Contains(url, strings.TrimRight(Api_GetBalancebyAsset, ":addr/:assetid")) {
		return Api_GetBalancebyAsset
	} else if strings.Contains(url, strings.TrimRight(Api_GetUTXObyAddr, ":addr")) {
		return Api_GetUTXObyAddr
	} else if strings.Contains(url, strings.TrimRight(Api_GetUTXObyAsset, ":addr/:assetid")) {
		return Api_GetUTXObyAsset
	} else if strings.Contains(url, strings.TrimRight(Api_Getasset, ":hash")) {
		return Api_Getasset
	} else if strings.Contains(url, strings.TrimRight(Api_GetStateUpdate, ":namespace/:key")) {
		return Api_GetStateUpdate
	}
	return url
}

func (rt *restServer) getParams(r *http.Request, url string, req map[string]interface{}) map[string]interface{} {
	switch url {
	case Api_Getconnectioncount:
		break
	case Api_GetblockTxsByHeight:
		req["Height"] = getParam(r, "height")
		break
	case Api_Getblockbyheight:
		req["Raw"] = r.FormValue("raw")
		req["Height"] = getParam(r, "height")
		break
	case Api_Getblockbyhash:
		req["Raw"] = r.FormValue("raw")
		req["Hash"] = getParam(r, "hash")
		break
	case Api_Getblockheight:
		break
	case Api_GetTransactionPool:
		break
	case Api_Getblockhash:
		req["Height"] = getParam(r, "height")
		break
	case Api_GetTotalIssued:
		req["Assetid"] = getParam(r, "assetid")
		break
	case Api_Gettransaction:
		req["Hash"] = getParam(r, "hash")
		req["Raw"] = r.FormValue("raw")
		break
	case Api_GetContract:
		req["Hash"] = getParam(r, "hash")
		req["Raw"] = r.FormValue("raw")
		break
	case Api_Getasset:
		req["Hash"] = getParam(r, "hash")
		req["Raw"] = r.FormValue("raw")
		break
	case Api_GetBalancebyAsset:
		req["Addr"] = getParam(r, "addr")
		req["Assetid"] = getParam(r, "assetid")
		break
	case Api_GetBalanceByAddr:
		req["Addr"] = getParam(r, "addr")
		break
	case Api_GetUTXObyAddr:
		req["Addr"] = getParam(r, "addr")
		break
	case Api_GetUTXObyAsset:
		req["Addr"] = getParam(r, "addr")
		req["Assetid"] = getParam(r, "assetid")
		break
	case Api_Restart:
		break
	case Api_SendRawTx:
		userid := r.FormValue("userid")
		if len(userid) == 0 {
			req["Userid"] = getParam(r, "userid")
		}
		break
	case Api_SendRcdTxByTrans:
		req["Raw"] = r.FormValue("raw")
		break
	case Api_GetStateUpdate:
		req["Namespace"] = getParam(r, "namespace")
		req["Key"] = getParam(r, "key")
		break
	case Api_WebsocketState:
		break
	default:
	}
	return req
}

func (rt *restServer) initGetHandler() {

	for k, _ := range rt.getMap {
		rt.router.Get(k, func(w http.ResponseWriter, r *http.Request) {

			var req = make(map[string]interface{})
			var resp map[string]interface{}

			url := rt.getPath(r.URL.Path)

			if h, ok := rt.getMap[url]; ok {
				req = rt.getParams(r, url, req)
				resp = h.handler(req)
				resp["Action"] = h.name
			} else {
				resp = ResponsePack(InvalidMethod)
			}
			rt.response(w, resp)
		})
	}
}

func (rt *restServer) initPostHandler() {
	for k, _ := range rt.postMap {
		rt.router.Post(k, func(w http.ResponseWriter, r *http.Request) {

			body, _ := ioutil.ReadAll(r.Body)
			defer r.Body.Close()

			var req = make(map[string]interface{})
			var resp map[string]interface{}

			url := rt.getPath(r.URL.Path)
			if h, ok := rt.postMap[url]; ok {
				if err := json.Unmarshal(body, &req); err == nil {
					req = rt.getParams(r, url, req)
					resp = h.handler(req)
					resp["Action"] = h.name
				} else {
					resp = ResponsePack(IllegalDataFormat)
					resp["Action"] = h.name
				}
			} else {
				resp = ResponsePack(InvalidMethod)
			}
			rt.response(w, resp)
		})
	}
	//Options
	for k, _ := range rt.postMap {
		rt.router.Options(k, func(w http.ResponseWriter, r *http.Request) {
			rt.write(w, []byte{})
		})
	}

}

func (rt *restServer) write(w http.ResponseWriter, data []byte) {
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json;charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func (rt *restServer) response(w http.ResponseWriter, resp map[string]interface{}) {
	resp["Desc"] = ErrMap[resp["Error"].(ErrCode)]
	data, err := json.Marshal(resp)
	if err != nil {
		log.Fatal("HTTP Handle - json.Marshal: %v", err)
		return
	}
	rt.write(w, data)
}

func (rt *restServer) Stop() {
	if rt.server != nil {
		rt.server.Shutdown(context.Background())
		log.Error("Close restful ")
	}
}

func (rt *restServer) Restart(cmd map[string]interface{}) map[string]interface{} {
	go func() {
		rt.Stop()
		rt.Start()
	}()

	var resp = ResponsePack(Success)
	return resp
}

func (rt *restServer) initTlsListen() (net.Listener, error) {

	CertPath := Parameters.RestCertPath
	KeyPath := Parameters.RestKeyPath

	// load cert
	cert, err := tls.LoadX509KeyPair(CertPath, KeyPath)
	if err != nil {
		log.Error("load keys fail", err)
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	log.Info("TLS listen port is ", strconv.Itoa(Parameters.HttpRestPort))
	listener, err := tls.Listen("tcp", ":"+strconv.Itoa(Parameters.HttpRestPort), tlsConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return listener, nil
}
