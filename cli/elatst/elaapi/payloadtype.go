package elaapi

import (
	"encoding/hex"
	"fmt"

	. "Elastos.ELA/common"
	"Elastos.ELA/core/transaction/payload"

	"github.com/yuin/gopher-lua"
)

const (
	luaCoinBaseTypeName       = "coinbase"
	luaTransferAssetTypeName  = "transferasset"
	luaRegisterAssetTypeName  = "registerasset"
	luaRecordTypeName         = "record"
	luaDeployCodeTypeName     = "deploycode"
)

// Registers my person type to given L.
func RegisterCoinBaseType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaCoinBaseTypeName)
	L.SetGlobal("coinbase", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newCoinBase))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), coinbaseMethods))
}

// Constructor
func newCoinBase(L *lua.LState) int {
	data, _ := hex.DecodeString(L.ToString(1))
	cb := &payload.CoinBase{
		CoinbaseData: data,
	}
	ud := L.NewUserData()
	ud.Value = cb
	L.SetMetatable(ud, L.GetTypeMetatable(luaCoinBaseTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *CoinBase and
// returns this *CoinBase.
func checkCoinBase(L *lua.LState, idx int) *payload.CoinBase {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.CoinBase); ok {
		return v
	}
	L.ArgError(1, "CoinBase expected")
	return nil
}

var coinbaseMethods = map[string]lua.LGFunction{
	"get": coinbaseGet,
}

// Getter and setter for the Person#Name
func coinbaseGet(L *lua.LState) int {
	p := checkCoinBase(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterTransferAssetType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaTransferAssetTypeName)
	L.SetGlobal("transferasset", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newTransferAsset))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), transferassetMethods))
}

// Constructor
func newTransferAsset(L *lua.LState) int {
	ta := &payload.TransferAsset{}
	ud := L.NewUserData()
	ud.Value = ta
	L.SetMetatable(ud, L.GetTypeMetatable(luaTransferAssetTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *TransferAsset and
// returns this *TransferAsset.
func checkTransferAsset(L *lua.LState, idx int) *payload.TransferAsset {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.TransferAsset); ok {
		return v
	}
	L.ArgError(1, "TransferAsset expected")
	return nil
}

var transferassetMethods = map[string]lua.LGFunction{
	"get": transferassetGet,
}

// Getter and setter for the Person#Name
func transferassetGet(L *lua.LState) int {
	p := checkTransferAsset(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterRegisterAssetType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaRegisterAssetTypeName)
	L.SetGlobal("registerasset", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newRegisterAsset))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), registerassetMethods))
}

// Constructor
func newRegisterAsset(L *lua.LState) int {
	ast := checkAsset(L, 1)
	amt := Fixed64(L.ToInt(2))
	cntlStr := L.ToString(3)
	cntl, _ := ToScriptHash(cntlStr)

	ra := &payload.RegisterAsset{
		Asset:      ast,
		Amount:     amt,
		Controller: cntl,
	}
	ud := L.NewUserData()
	ud.Value = ra
	L.SetMetatable(ud, L.GetTypeMetatable(luaRegisterAssetTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *RegisterAsset and
// returns this *RegisterAsset.
func checkRegisterAsset(L *lua.LState, idx int) *payload.RegisterAsset {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.RegisterAsset); ok {
		return v
	}
	L.ArgError(1, "RegisterAsset expected")
	return nil
}

var registerassetMethods = map[string]lua.LGFunction{
	"get": registerassetGet,
}

// Getter and setter for the Person#Name
func registerassetGet(L *lua.LState) int {
	p := checkRegisterAsset(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterRecordType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaRecordTypeName)
	L.SetGlobal("record", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newRecord))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), recordMethods))
}

// Constructor
func newRecord(L *lua.LState) int {
	recordType := L.ToString(1)
	recordDataStr := L.ToString(2)
	recordData, _ := hex.DecodeString(recordDataStr)

	record := &payload.Record{
		RecordType: recordType,
		RecordData: recordData,
	}
	ud := L.NewUserData()
	ud.Value = record
	L.SetMetatable(ud, L.GetTypeMetatable(luaRecordTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Record and
// returns this *Record.
func checkRecord(L *lua.LState, idx int) *payload.Record {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.Record); ok {
		return v
	}
	L.ArgError(1, "Record expected")
	return nil
}

var recordMethods = map[string]lua.LGFunction{
	"get": recordGet,
}

// Getter and setter for the Person#Name
func recordGet(L *lua.LState) int {
	p := checkRecord(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterDeployCodeType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaDeployCodeTypeName)
	L.SetGlobal("deploycode", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newDeployCode))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), deploycodeMethods))
}

// Constructor
func newDeployCode(L *lua.LState) int {
	codes := checkFunctionCode(L, 1)
	name := L.ToString(2)
	version := L.ToString(3)
	author := L.ToString(4)
	email := L.ToString(5)
	descrip := L.ToString(6)

	dc := &payload.DeployCode{
		Code:        codes,
		Name:        name,
		CodeVersion: version,
		Author:      author,
		Email:       email,
		Description: descrip,
	}
	ud := L.NewUserData()
	ud.Value = dc
	L.SetMetatable(ud, L.GetTypeMetatable(luaDeployCodeTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *DeployCode and
// returns this *DeployCode.
func checkDeployCode(L *lua.LState, idx int) *payload.DeployCode {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.DeployCode); ok {
		return v
	}
	L.ArgError(1, "DeployCode expected")
	return nil
}

var deploycodeMethods = map[string]lua.LGFunction{
	"get": deploycodeGet,
}

// Getter and setter for the Person#Name
func deploycodeGet(L *lua.LState) int {
	p := checkDeployCode(L, 1)
	fmt.Println(p)

	return 0
}
