package httpwebsocket

import (
	. "Elastos.ELA/common"
	. "Elastos.ELA/common/config"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/events"
	. "Elastos.ELA/net/httpjsonrpc"
	"Elastos.ELA/net/httprestful/common"
	Err "Elastos.ELA/net/httprestful/error"
	"Elastos.ELA/net/httpwebsocket/websocket"
	. "Elastos.ELA/net/protocol"
	"bytes"
)

var ws *websocket.WsServer
var (
	pushBlockFlag    bool = true
	pushRawBlockFlag bool = false
	pushBlockTxsFlag bool = false
	pushNewTxsFlag   bool = true
)

func StartServer(n Noder) {
	common.SetNode(n)
	ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, SendBlock2WSclient)
	ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventNewTransactionPutInPool, SendTransaction2WSclient)
	go func() {
		ws = websocket.InitWsServer(common.CheckAccessToken)
		ws.Start()
	}()
}

func SendTransaction2WSclient(v interface{}) {
	if Parameters.HttpWsPort != 0 && pushNewTxsFlag {
		go func() {
			PushNewTransaction(v)
		}()
	}
}

func SendBlock2WSclient(v interface{}) {
	if Parameters.HttpWsPort != 0 && pushBlockFlag {
		go func() {
			PushBlock(v)
		}()
	}
	if Parameters.HttpWsPort != 0 && pushBlockTxsFlag {
		go func() {
			PushBlockTransactions(v)
		}()
	}
}
func Stop() {
	if ws == nil {
		return
	}
	ws.Stop()
}
func ReStartServer() {
	if ws == nil {
		ws = websocket.InitWsServer(common.CheckAccessToken)
		ws.Start()
		return
	}
	ws.Restart()
}
func GetWsPushBlockFlag() bool {
	return pushBlockFlag
}
func SetWsPushBlockFlag(b bool) {
	pushBlockFlag = b
}
func GetPushRawBlockFlag() bool {
	return pushRawBlockFlag
}
func SetPushRawBlockFlag(b bool) {
	pushRawBlockFlag = b
}
func GetPushBlockTxsFlag() bool {
	return pushBlockTxsFlag
}
func SetPushBlockTxsFlag(b bool) {
	pushBlockTxsFlag = b
}
func SetPushNewTxsFlag(b bool) {
	pushNewTxsFlag = b
}
func SetTxHashMap(txhash string, sessionid string) {
	if ws == nil {
		return
	}
	ws.SetTxHashMap(txhash, sessionid)
}

func PushBlock(v interface{}) {
	if ws == nil {
		return
	}
	resp := common.ResponsePack(Err.SUCCESS)
	if block, ok := v.(*ledger.Block); ok {
		if pushRawBlockFlag {
			w := bytes.NewBuffer(nil)
			block.Serialize(w)
			resp["Result"] = BytesToHexString(w.Bytes())
		} else {
			resp["Result"] = common.GetBlockInfo(block)
		}
		resp["Action"] = "sendrawblock"
		ws.PushResult(resp)
	}
}

func PushNewTransaction(v interface{}) {
	if ws == nil {
		return
	}
	resp := common.ResponsePack(Err.SUCCESS)
	if trx, ok := v.(*transaction.Transaction); ok {
		if pushNewTxsFlag {
			resp["Result"] = TransArryByteToHexString(trx)
		}
		resp["Action"] = "sendblocktransactions"
		ws.PushResult(resp)
	}
}

func PushBlockTransactions(v interface{}) {
	if ws == nil {
		return
	}
	resp := common.ResponsePack(Err.SUCCESS)
	if block, ok := v.(*ledger.Block); ok {
		if pushBlockTxsFlag {
			resp["Result"] = common.GetBlockTransactions(block)
		}
		resp["Action"] = "sendblocktransactions"
		ws.PushResult(resp)
	}
}
