// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/crypto"

	lua "github.com/yuin/gopher-lua"
)

const luaAggPubTypeName = "aggpub"

func RegisterAggPubType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaAggPubTypeName)
	L.SetGlobal("aggpub", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newAggPub))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), aggpubMethods))
}

func newAggPub(L *lua.LState) int {
	pubKeysTable := L.ToTable(1)
	var pubs [][]byte
	pubKeysTable.ForEach(func(i, value lua.LValue) {
		pub, _ := hex.DecodeString(lua.LVAsString(value))
		pubs = append(pubs, pub)
	})
	apub, err := crypto.AggregatePublickeys(pubs)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	ud := L.NewUserData()
	ud.Value = apub
	L.SetMetatable(ud, L.GetTypeMetatable(luaAggPubTypeName))
	L.Push(ud)

	return 1
}

var aggpubMethods = map[string]lua.LGFunction{
	"get_aggpub": getAggPub,
}

func getAggPub(L *lua.LState) int {
	sa, err := checkAggPub(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	pub := hex.EncodeToString(sa)
	fmt.Println("-----ToAggPub-------" + pub)
	L.Push(lua.LString(pub))

	return 1
}

func checkAggPub(L *lua.LState, idx int) ([]byte, error) {
	v := L.Get(idx)
	if ud, ok := v.(*lua.LUserData); ok {
		if v, ok := ud.Value.([]byte); ok {
			return v, nil
		}
	}

	return nil, errors.New("account expected")
}
