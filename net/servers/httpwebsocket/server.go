package httpwebsocket

import (
	. "Elastos.ELA/common"
	. "Elastos.ELA/common/config"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/events"
	"Elastos.ELA/common/log"
	. "Elastos.ELA/net/servers"
	. "Elastos.ELA/errors"
	"bytes"
	"net"
	"crypto/tls"
	"strconv"
	"sync"
	"net/http"
	"time"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
)

var WebSocketServer *WsServer

var (
	PushBlockFlag    = true
	PushRawBlockFlag = false
	PushBlockTxsFlag = false
	PushNewTxsFlag   = true
)

type Handler func(map[string]interface{}) map[string]interface{}

type WsServer struct {
	sync.RWMutex
	Upgrader    websocket.Upgrader
	listener    net.Listener
	server      *http.Server
	SessionList *SessionList
	ActionMap   map[string]Handler
}

func StartServer() {
	WebSocketServer = &WsServer{
		Upgrader:    websocket.Upgrader{},
		SessionList: &SessionList{OnlineList: make(map[string]*Session)},
	}
	WebSocketServer.Start()

	ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, SendBlock2WSclient)
	ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventNewTransactionPutInPool, SendTransaction2WSclient)
}

func (ws *WsServer) Start() {
	ws.initializeMethods()
	ws.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	if Parameters.HttpWsPort%1000 == TlsPort {
		var err error
		ws.listener, err = ws.initTlsListen()
		if err != nil {
			log.Error("Https Cert: ", err.Error())
		}
	} else {
		var err error
		ws.listener, err = net.Listen("tcp", ":"+strconv.Itoa(Parameters.HttpWsPort))
		if err != nil {
			log.Fatal("net.Listen: ", err.Error())
		}
	}
	var done = make(chan bool)
	go ws.checkSessionsTimeout(done)

	ws.server = &http.Server{Handler: http.HandlerFunc(ws.webSocketHandler)}
	err := ws.server.Serve(ws.listener)

	done <- true
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

func (ws *WsServer) initializeMethods() {
	heartbeat := func(cmd map[string]interface{}) map[string]interface{} {
		return ResponsePack("heartbeat", Success, cmd["Userid"])
	}

	getsessioncount := func(cmd map[string]interface{}) map[string]interface{} {
		return ResponsePack("getsessioncount", Success,len(ws.SessionList.OnlineList))
	}
	ws.ActionMap = map[string]Handler{
		"getconnectioncount": GetConnectionCount,
		"getblockbyheight":   GetBlockByHeight,
		"getblockbyhash":     GetBlockByHash,
		"getblockheight":     GetBlockHeight,
		"gettransaction":     GetTransactionByHash,
		"getasset":           GetAssetByHash,
		"getunspendoutput":   GetUnspendOutput,
		"sendrawtransaction": SendRawTransaction,
		"heartbeat":          heartbeat,
		"getsessioncount":    getsessioncount,
	}
}

func (ws *WsServer) Stop() {
	ws.server.Shutdown(context.Background())
	log.Info("Close websocket ")
}

func (ws *WsServer) checkSessionsTimeout(done chan bool) {
	ticker := time.NewTicker(time.Second * Parameters.Configuration.WsHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var closeList []*Session
			ws.SessionList.ForEachSession(func(v *Session) {
				if v.SessionTimeoverCheck() {
					resp := ResponsePack("checksessionstimeout", SessionExpired, "")
					ws.response(v.SessionId, resp)
					closeList = append(closeList, v)
				}
			})
			for _, s := range closeList {
				ws.SessionList.CloseSession(s)
			}
		case <-done:
			return
		}
	}

}

//webSocketHandler
func (ws *WsServer) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	wsConn, err := ws.Upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Error("websocket Upgrader: ", err)
		return
	}
	defer wsConn.Close()

	newSession := &Session{
		Connection: wsConn,
		LastActive: time.Now().Unix(),
		SessionId:  uuid.NewUUID().String(),
	}
	ws.SessionList.OnlineList[newSession.SessionId] = newSession

	defer func() {
		ws.SessionList.CloseSession(newSession)
	}()

	for {
		_, bysMsg, err := wsConn.ReadMessage()
		if err == nil {
			if ws.OnDataHandle(newSession, bysMsg, r) {
				newSession.LastActive = time.Now().Unix()
			}
			continue
		}
		e, ok := err.(net.Error)
		if !ok || !e.Timeout() {
			log.Error("websocket conn:", err)
			return
		}
	}
}

func (ws *WsServer) IsValidMsg(reqMsg map[string]interface{}) bool {
	if _, ok := reqMsg["Hash"].(string); !ok && reqMsg["Hash"] != nil {
		return false
	}
	if _, ok := reqMsg["Addr"].(string); !ok && reqMsg["Addr"] != nil {
		return false
	}
	if _, ok := reqMsg["Assetid"].(string); !ok && reqMsg["Assetid"] != nil {
		return false
	}
	return true
}

func (ws *WsServer) OnDataHandle(currentSession *Session, bysMsg []byte, r *http.Request) bool {

	var req = make(map[string]interface{})

	if err := json.Unmarshal(bysMsg, &req); err != nil {
		resp := ResponsePack("", IllegalDataFormat, "")
		ws.response(currentSession.SessionId, resp)
		log.Error("websocket OnDataHandle:", err)
		return false
	}
	actionName := req["Action"].(string)

	action, ok := ws.ActionMap[actionName]
	if !ok {
		resp := ResponsePack(actionName, InvalidMethod, "")
		ws.response(currentSession.SessionId, resp)
		return false
	}
	if !ws.IsValidMsg(req) {
		resp := ResponsePack(actionName, InvalidParams, "")
		ws.response(currentSession.SessionId, resp)
		return true
	}
	if height, ok := req["Height"].(float64); ok {
		req["Height"] = strconv.FormatInt(int64(height), 10)
	}
	if raw, ok := req["Raw"].(float64); ok {
		req["Raw"] = strconv.FormatInt(int64(raw), 10)
	}

	resp := action(req)
	resp["Action"] = actionName

	ws.response(currentSession.SessionId, resp)

	return true
}

func (ws *WsServer) response(sessionId string, resp map[string]interface{}) {
	resp["Desc"] = ErrMap[resp["Error"].(ErrCode)]
	data, err := json.Marshal(resp)
	if err != nil {
		log.Error("Websocket response:", err)
		return
	}
	ws.SessionList.OnlineList[sessionId].Send(data)
}

func SendTransaction2WSclient(v interface{}) {
	if PushNewTxsFlag {
		go func() {
			WebSocketServer.PushResult("sendnewtransaction", v)
		}()
	}
}

func SendBlock2WSclient(v interface{}) {
	if PushBlockFlag {
		go func() {
			WebSocketServer.PushResult("sendblock", v)
		}()
	}
	if PushRawBlockFlag {
		go func() {
			WebSocketServer.PushResult("sendrawblock", v)
		}()
	}
	if PushBlockTxsFlag {
		go func() {
			WebSocketServer.PushResult("sendblocktransactions", v)
		}()
	}
}

func (ws *WsServer) PushResult(action string, v interface{}) {
	var result interface{}
	switch action {
	case "sendblock":
		if block, ok := v.(*ledger.Block); ok {
			result = GetBlockInfo(block)
		}
	case "sendrawblock":
		if block, ok := v.(*ledger.Block); ok {
			w := bytes.NewBuffer(nil)
			block.Serialize(w)
			result = BytesToHexString(w.Bytes())
		}
	case "sendblocktransactions":
		if block, ok := v.(*ledger.Block); ok {
			result = GetBlockTransactions(block)
		}
	case "sendnewtransaction":
		if trx, ok := v.(*transaction.Transaction); ok {
			result = TransArrayByteToHexString(trx)
		}
	default:
		log.Error("httpwebsocket/server.go in pushresult function: unknown action")
	}

	resp := ResponsePack(action, Success, result)

	data, err := json.Marshal(resp)
	if err != nil {
		log.Error("Websocket PushResult:", err)
		return
	}
	ws.broadcast(data)
}

func (ws *WsServer) broadcast(data []byte) error {
	ws.SessionList.ForEachSession(func(v *Session) {
		v.Send(data)
	})
	return nil
}

func (ws *WsServer) initTlsListen() (net.Listener, error) {

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

	log.Info("TLS listen port is ", strconv.Itoa(Parameters.HttpWsPort))
	listener, err := tls.Listen("tcp", ":"+strconv.Itoa(Parameters.HttpWsPort), tlsConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return listener, nil
}
