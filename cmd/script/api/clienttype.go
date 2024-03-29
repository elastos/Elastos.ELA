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
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/yuin/gopher-lua"
)

const luaClientTypeName = "client"

func RegisterClientType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaClientTypeName)
	L.SetGlobal("client", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newClient))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), clientMethods))
}

// Constructor
func newClient(L *lua.LState) int {
	name := L.ToString(1)
	pwd := L.ToString(2)
	create := L.ToBool(3)
	var wallet *account.Client
	if create {
		wallet, _ = account.Create(name, []byte(pwd))
	} else {
		wallet, _ = account.Open(name, []byte(pwd))
	}

	ud := L.NewUserData()
	ud.Value = wallet
	L.SetMetatable(ud, L.GetTypeMetatable(luaClientTypeName))
	L.Push(ud)

	return 1
}

func checkClient(L *lua.LState, idx int) (*account.Client, error) {
	v := L.Get(idx)
	if ud, ok := v.(*lua.LUserData); ok {
		if v, ok := ud.Value.(*account.Client); ok {
			return v, nil
		}
	}

	return nil, errors.New("wallet expected")
}

var clientMethods = map[string]lua.LGFunction{
	"get":                 clientGet,
	"get_address":         getWalletAddr,
	"get_s_address":       getSWalletAddr,
	"get_s_multi_address": getMultiSWalletAddr,
	"get_publickey":       getWalletPubkey,
}

// Getter and setter for the Person#Name
func clientGet(L *lua.LState) int {
	p, err := checkClient(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(p)

	return 0
}

func getSWalletAddr(L *lua.LState) int {
	wallet, err := checkClient(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	acc := wallet.GetMainAccount()
	codeHash := acc.ProgramHash.ToCodeHash()
	depositHash := common.Uint168FromCodeHash(byte(contract.PrefixDPoSV2), codeHash)
	sAddress, err := depositHash.ToAddress()
	if err != nil {
		fmt.Println(err.Error())
	}
	L.Push(lua.LString(sAddress))
	return 1
}

func getMultiSWalletAddr(L *lua.LState) int {
	wallet, err := checkClient(L, 1)
	m := L.ToInt64(2)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	pks := make([]*crypto.PublicKey, 0)
	accs := wallet.GetAccounts()
	for _, acc := range accs {
		pks = append(pks, acc.PublicKey)
	}

	multiCode, err := contract.CreateMultiSigRedeemScript(int(m), pks)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	ct, err := contract.CreateStakeContractByCode(multiCode)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	stakeProgramHash := ct.ToProgramHash()
	sAddress, err := stakeProgramHash.ToAddress()
	if err != nil {
		fmt.Println(err.Error())
	}
	L.Push(lua.LString(sAddress))
	return 1
}

func getWalletAddr(L *lua.LState) int {
	wallet, err := checkClient(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	acc := wallet.GetMainAccount()
	addr, _ := acc.ProgramHash.ToAddress()

	L.Push(lua.LString(addr))

	return 1
}

func getWalletPubkey(L *lua.LState) int {
	wallet, err := checkClient(L, 1)
	if err != nil {
		fmt.Println(err.Error())
	}
	acc := wallet.GetMainAccount()
	pubkey, _ := acc.PublicKey.EncodePoint(true)
	L.Push(lua.LString(hex.EncodeToString(pubkey)))

	return 1
}
