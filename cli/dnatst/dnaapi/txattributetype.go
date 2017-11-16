package dnaapi

import (
	tx "DNA_POW/core/transaction"
	"encoding/hex"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaTxAttributeTypeName = "txattribute"

// Registers my person type to given L.
func RegisterTxAttributeType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaTxAttributeTypeName)
	L.SetGlobal("txattribute", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newTxAttribute))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), txattributeMethods))
}

// Constructor
func newTxAttribute(L *lua.LState) int {
	usage := L.ToInt(1)
	dataStr := L.ToString(2)
	size := L.ToInt(3)
	data, _ := hex.DecodeString(dataStr)

	txAttr := &tx.TxAttribute{
		Usage: tx.TransactionAttributeUsage(usage),
		Data:  data,
		Size:  uint32(size),
	}
	ud := L.NewUserData()
	ud.Value = txAttr
	L.SetMetatable(ud, L.GetTypeMetatable(luaTxAttributeTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *TxAttribute and returns this *TxAttribute.
func checkTxAttribute(L *lua.LState, idx int) *tx.TxAttribute {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*tx.TxAttribute); ok {
		return v
	}
	L.ArgError(1, "TxAttribute expected")
	return nil
}

var txattributeMethods = map[string]lua.LGFunction{
	"get": txattributeGet,
}

// Getter and setter for the Person#Name
func txattributeGet(L *lua.LState) int {
	p := checkTxAttribute(L, 1)
	fmt.Println(p)

	return 0
}
