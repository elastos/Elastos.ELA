package elaapi

import (
	. "Elastos.ELA/common"
	tx "Elastos.ELA/core/transaction"
	"encoding/hex"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaUTXOTxInputTypeName = "utxotxinput"

// Registers my person type to given L.
func RegisterUTXOTxInputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaUTXOTxInputTypeName)
	L.SetGlobal("utxotxinput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newUTXOTxInput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), utxotxinputMethods))
}

// Constructor
func newUTXOTxInput(L *lua.LState) int {
	referIDStr := L.ToString(1)
	referIdx := L.ToInt(2)
	sequence := L.ToInt(3)
	referIDSlice, _ := hex.DecodeString(referIDStr)
	referIDSlice = BytesReverse(referIDSlice)
	var referID Uint256
	copy(referID[:], referIDSlice[0:32])
	input := &tx.UTXOTxInput{
		ReferTxID:          referID,
		ReferTxOutputIndex: uint16(referIdx),
		Sequence:           uint32(sequence),
	}
	ud := L.NewUserData()
	ud.Value = input
	L.SetMetatable(ud, L.GetTypeMetatable(luaUTXOTxInputTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *UTXOTxInput and returns this *UTXOTxInput.
func checkUTXOTxInput(L *lua.LState, idx int) *tx.UTXOTxInput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*tx.UTXOTxInput); ok {
		return v
	}
	L.ArgError(1, "UTXOTxInput expected")
	return nil
}

var utxotxinputMethods = map[string]lua.LGFunction{
	"get": utxotxinputGet,
}

// Getter and setter for the Person#Name
func utxotxinputGet(L *lua.LState) int {
	p := checkUTXOTxInput(L, 1)
	fmt.Println(p)

	return 0
}
