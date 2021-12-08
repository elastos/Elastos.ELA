// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package api

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"math/rand"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/yuin/gopher-lua"
)

const luaHeaderTypeName = "header"

func RegisterHeaderType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaHeaderTypeName)
	L.SetGlobal("header", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newHeader))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), headerMethods))
}

// Constructor
func newHeader(L *lua.LState) int {
	version := uint32(L.ToInt(1))
	prevBlockHash := L.ToString(2)
	height := uint32(L.ToInt(3))

	hash, _ := common.Uint256FromHexString(prevBlockHash)

	header := &common2.Header{
		Version:    version,
		Previous:   *hash,
		MerkleRoot: common.EmptyHash,
		Timestamp:  uint32(time.Now().Unix()),
		Bits:       config.DefaultParams.PowLimitBits,
		Height:     height,
		Nonce:      rand.Uint32(),
	}

	ud := L.NewUserData()
	ud.Value = header
	L.SetMetatable(ud, L.GetTypeMetatable(luaBlockTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Attribute and returns this *Attribute.
func checkHeader(L *lua.LState, idx int) *common2.Header {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*common2.Header); ok {
		return v
	}
	L.ArgError(1, "DPosProposal expected")
	return nil
}

var headerMethods = map[string]lua.LGFunction{
	"hash":   headerHash,
	"height": headerHeight,
}

func headerHeight(L *lua.LState) int {
	h := checkHeader(L, 1)

	L.Push(lua.LString(string(rune(h.Height))))
	return 1
}

func headerHash(L *lua.LState) int {
	p := checkHeader(L, 1)
	h := p.Hash()

	L.Push(lua.LString(h.String()))

	return 1
}
