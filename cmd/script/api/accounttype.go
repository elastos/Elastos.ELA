// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/account"
	lua "github.com/yuin/gopher-lua"
)

const luaAccountTypeName = "account"

func RegisterAccountType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaAccountTypeName)
	L.SetGlobal("account", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newAccount))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), accountMethods))
}

func newAccount(L *lua.LState) int {
	privateKeysTable := L.ToTable(1)
	var accs []*account.Account
	privateKeysTable.ForEach(func(i, value lua.LValue) {
		acc := new(account.Account)
		privateKey, _ := hex.DecodeString(lua.LVAsString(value))
		acc.PrivateKey = privateKey
		accs = append(accs, acc)
	})
	sa := account.NewSchnorrAggregateAccount(accs)
	ud := L.NewUserData()
	ud.Value = sa
	L.SetMetatable(ud, L.GetTypeMetatable(luaAccountTypeName))
	L.Push(ud)

	return 1
}

var accountMethods = map[string]lua.LGFunction{
	"get_account": getAccount,
	"get_address": getAccountAddr,
}

func getAccount(L *lua.LState) int {
	sa, err := checkAccount(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(sa)

	return 0
}

func getAccountAddr(L *lua.LState) int {
	sa, err := checkAccount(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("-----ToAddress-------")
	addr, _ := sa.ProgramHash.ToAddress()
	fmt.Println("-----ToAddress-------")
	L.Push(lua.LString(addr))

	return 1
}

func checkAccount(L *lua.LState, idx int) (*account.SchnorAccount, error) {
	v := L.Get(idx)
	if ud, ok := v.(*lua.LUserData); ok {
		if v, ok := ud.Value.(*account.SchnorAccount); ok {
			return v, nil
		}
	}

	return nil, errors.New("account expected")
}
