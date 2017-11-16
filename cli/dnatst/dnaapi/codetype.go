package dnaapi

import (
	"DNA_POW/core/code"
	"DNA_POW/core/contract"
	"encoding/hex"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

const luaFunctionCodeTypeName = "functioncode"

// Registers my person type to given L.
func RegisterFunctionCodeType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaFunctionCodeTypeName)
	L.SetGlobal("functioncode", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newFunctionCode))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), functioncodeMethods))
}

// Constructor
func newFunctionCode(L *lua.LState) int {
	codes, _ := hex.DecodeString(L.ToString(1))
	para, _ := hex.DecodeString(L.ToString(2))
	ret, _ := hex.DecodeString(L.ToString(3))

	paras := make([]contract.ContractParameterType, 0)
	for _, x := range para {
		paras = append(paras, contract.ContractParameterType(x))
	}
	rets := make([]contract.ContractParameterType, 0)
	for _, x := range ret {
		rets = append(rets, contract.ContractParameterType(x))
	}

	fc := &code.FunctionCode{
		Code:           codes,
		ParameterTypes: paras,
		ReturnTypes:    rets,
	}
	ud := L.NewUserData()
	ud.Value = fc
	L.SetMetatable(ud, L.GetTypeMetatable(luaFunctionCodeTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *FunctionCode and
// returns this *FunctionCode.
func checkFunctionCode(L *lua.LState, idx int) *code.FunctionCode {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*code.FunctionCode); ok {
		return v
	}
	L.ArgError(1, "FunctionCode expected")
	return nil
}

var functioncodeMethods = map[string]lua.LGFunction{
	"get": functioncodeGet,
}

// Getter and setter for the Person#Name
func functioncodeGet(L *lua.LState) int {
	p := checkFunctionCode(L, 1)
	fmt.Println(p)

	return 0
}
