package elaapi

import (
	. "Elastos.ELA/common"
	tx "Elastos.ELA/core/transaction"
	"encoding/hex"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaBalanceTxInputTypeName = "balancetxinput"

// Registers my person type to given L.
func RegisterBalanceTxInputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaBalanceTxInputTypeName)
	L.SetGlobal("balancetxinput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newBalanceTxInput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), balancetxinputMethods))
}

// Constructor
func newBalanceTxInput(L *lua.LState) int {
	assetIDStr := L.ToString(1)
	value := L.ToInt64(2)
	programHashStr := L.ToString(3)

	assetIDSlice, _ := hex.DecodeString(assetIDStr)
	var assetID Uint256
	copy(assetID[:], assetIDSlice[0:32])

	//TODO programhash to address
	programHashSlice, _ := hex.DecodeString(programHashStr)
	var programHash Uint168
	copy(programHash[:], programHashSlice[0:20])

	balance := &tx.BalanceTxInput{
		AssetID:     assetID,
		Value:       Fixed64(value),
		ProgramHash: programHash,
	}
	ud := L.NewUserData()
	ud.Value = balance
	L.SetMetatable(ud, L.GetTypeMetatable(luaBalanceTxInputTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *BalanceTxInput and returns this *BalanceTxInput.
func checkBalanceTxInput(L *lua.LState, idx int) *tx.BalanceTxInput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*tx.BalanceTxInput); ok {
		return v
	}
	L.ArgError(1, "BalanceTxInput expected")
	return nil
}

var balancetxinputMethods = map[string]lua.LGFunction{
	"get": balancetxinputGet,
}

// Getter and setter for the Person#Name
func balancetxinputGet(L *lua.LState) int {
	p := checkBalanceTxInput(L, 1)
	fmt.Println(p)

	return 0
}
