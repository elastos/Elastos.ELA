package elaapi

import (
	. "ELA/common"
	tx "ELA/core/transaction"
	"encoding/hex"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaTxOutputTypeName = "txoutput"

// Registers my person type to given L.
func RegisterTxOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaTxOutputTypeName)
	L.SetGlobal("txoutput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newTxOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), txoutputMethods))
}

// Constructor
func newTxOutput(L *lua.LState) int {
	assetIDStr := L.ToString(1)
	assetIDSlice, _ := hex.DecodeString(assetIDStr)
	assetIDSlice = BytesReverse(assetIDSlice)
	var assetID Uint256
	copy(assetID[:], assetIDSlice[0:32])

	value := L.ToInt64(2)
	programHashStr := L.ToString(3)
	programHash, _ := ToScriptHash(programHashStr)

	output := &tx.TxOutput{
		AssetID:     assetID,
		Value:       Fixed64(value),
		ProgramHash: programHash,
	}
	ud := L.NewUserData()
	ud.Value = output
	L.SetMetatable(ud, L.GetTypeMetatable(luaTxOutputTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *TxOutput and returns this *TxOutput.
func checkTxOutput(L *lua.LState, idx int) *tx.TxOutput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*tx.TxOutput); ok {
		return v
	}
	L.ArgError(1, "TxOutput expected")
	return nil
}

var txoutputMethods = map[string]lua.LGFunction{
	"get": txoutputGet,
}

// Getter and setter for the Person#Name
func txoutputGet(L *lua.LState) int {
	p := checkTxOutput(L, 1)
	fmt.Println(p)

	return 0
}
