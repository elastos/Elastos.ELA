package elaapi

import (
	"ELA/core/asset"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaAssetTypeName = "asset"

// Registers my person type to given L.
func RegisterAssetType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaAssetTypeName)
	L.SetGlobal("asset", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newAsset))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), assetMethods))
}

// Constructor
func newAsset(L *lua.LState) int {
	name := L.ToString(1)
	description := L.ToString(2)
	precision := byte(L.ToInt(3))
	assetType := asset.AssetType(L.ToInt(4))
	recordType := asset.AssetRecordType(L.ToInt(5))

	at := &asset.Asset{
		Name:        name,
		Description: description,
		Precision:   precision,
		AssetType:   assetType,
		RecordType:  recordType,
	}
	ud := L.NewUserData()
	ud.Value = at
	L.SetMetatable(ud, L.GetTypeMetatable(luaAssetTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Asset and returns this *Asset.
func checkAsset(L *lua.LState, idx int) *asset.Asset {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*asset.Asset); ok {
		return v
	}
	L.ArgError(1, "Asset expected")
	return nil
}

var assetMethods = map[string]lua.LGFunction{
	"get": assetGet,
}

// Getter and setter for the Person#Name
func assetGet(L *lua.LState) int {
	p := checkAsset(L, 1)
	fmt.Println(p)

	return 0
}
