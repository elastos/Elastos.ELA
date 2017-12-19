package elaapi

import (
	"bytes"
	"encoding/hex"
	"fmt"

	. "ELA/common"
	"ELA/core/auxpow"
	"ELA/core/ledger"
	tx "ELA/core/transaction"

	"github.com/yuin/gopher-lua"
)

const (
	luaBlockdataTypeName = "blockdata"
	luaBlockTypeName     = "block"
)

// Registers my person type to given L.
func RegisterBlockdataType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaBlockdataTypeName)
	L.SetGlobal("blockdata", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newBlockdata))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), blockdataMethods))
}

func U256FromString(in string) Uint256 {
	ss, _ := hex.DecodeString(in)
	ss = BytesReverse(ss)
	var ret Uint256
	copy(ret[:], ss[0:32])
	return ret
}

// Constructor
func newBlockdata(L *lua.LState) int {
	version := uint32(L.ToInt(1))
	preHashStr := L.ToString(2)
	preHash := U256FromString(preHashStr)
	txRootStr := L.ToString(3)
	txRoot := U256FromString(txRootStr)
	timestamp := uint32(L.ToInt(4))
	bits := uint32(L.ToInt(5))
	height := uint32(L.ToInt(6))
	nonce := uint32(L.ToInt(7))

	bd := &ledger.Blockdata{
		Version:          version,
		PrevBlockHash:    preHash,
		TransactionsRoot: txRoot,
		Timestamp:        timestamp,
		Bits:             bits,
		Height:           height,
		Nonce:            nonce,
		AuxPow:           auxpow.AuxPow{},
	}
	ud := L.NewUserData()
	ud.Value = bd
	L.SetMetatable(ud, L.GetTypeMetatable(luaBlockdataTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Blockdata and returns this *Blockdata.
func checkBlockdata(L *lua.LState, idx int) *ledger.Blockdata {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*ledger.Blockdata); ok {
		return v
	}
	L.ArgError(1, "Blockdata expected")
	return nil
}

var blockdataMethods = map[string]lua.LGFunction{
	"get": blockdataGet,
}

// Getter and setter for the Person#Name
func blockdataGet(L *lua.LState) int {
	p := checkBlockdata(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterBlockType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaBlockTypeName)
	L.SetGlobal("block", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newBlock))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), blockMethods))
}

// Constructor
func newBlock(L *lua.LState) int {
	header := checkBlockdata(L, 1)

	bd := &ledger.Block{
		Blockdata:    header,
		Transactions: []*tx.Transaction{},
	}
	ud := L.NewUserData()
	ud.Value = bd
	L.SetMetatable(ud, L.GetTypeMetatable(luaBlockTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Block and returns this *Block.
func checkBlock(L *lua.LState, idx int) *ledger.Block {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*ledger.Block); ok {
		return v
	}
	L.ArgError(1, "Block expected")
	return nil
}

var blockMethods = map[string]lua.LGFunction{
	"get":          blockGet,
	"getPrevHash":  blockGetPrevHash,
	"getTxRoot":    blockGetTxRoot,
	"getTimeStamp": blockGetTimeStamp,
	"getHeight":    blockGetHeight,
	"getBits":      blockGetBits,
	"appendtx":     blockAppendTx,
	"updataRoot":   blockUpdataRoot,
	"hash":         blockHash,
	"serialize":    blockSerialize,
	"deserialize":  blockDeserialize,
	"mining":       blockMining,
}

func blockAppendTx(L *lua.LState) int {
	b := checkBlock(L, 1)
	p := checkTransaction(L, 2)
	b.Transactions = append(b.Transactions, p)

	return 0
}

// Getter and setter for the Person#Name
func blockGet(L *lua.LState) int {
	b := checkBlock(L, 1)
	fmt.Println(b)

	return 0
}

func blockUpdataRoot(L *lua.LState) int {
	b := checkBlock(L, 1)
	b.RebuildMerkleRoot()

	return 0
}

func blockHash(L *lua.LState) int {
	b := checkBlock(L, 1)
	h := b.Hash()

	L.Push(lua.LString(hex.EncodeToString(h[:])))

	return 1
}

func blockSerialize(L *lua.LState) int {
	b := checkBlock(L, 1)
	var buffer bytes.Buffer
	b.Serialize(&buffer)
	blockHex := hex.EncodeToString(buffer.Bytes())

	L.Push(lua.LNumber(len(buffer.Bytes())))
	L.Push(lua.LString(blockHex))
	return 2
}

func blockDeserialize(L *lua.LState) int {
	b := checkBlock(L, 1)
	block, _ := hex.DecodeString(L.ToString(2))

	b.Deserialize(bytes.NewReader(block))

	return 0
}

func blockMining(L *lua.LState) int {
	block := checkBlock(L, 1)
	header := block.Blockdata
	targetDifficulty := ledger.CompactToBig(header.Bits)

	//TODO if block already mined

	for i := uint32(0); i <= ^uint32(0); i++ {
		header.Nonce = i
		hash := header.Hash()
		if ledger.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
			break
		}
	}

	return 0
}

func blockGetPrevHash(L *lua.LState) int {
	block := checkBlock(L, 1)
	L.Push(lua.LString(hex.EncodeToString(block.Blockdata.PrevBlockHash[:])))

	return 1
}

func blockGetTxRoot(L *lua.LState) int {
	block := checkBlock(L, 1)
	L.Push(lua.LString(hex.EncodeToString(block.Blockdata.TransactionsRoot[:])))

	return 1
}
func blockGetTimeStamp(L *lua.LState) int {
	block := checkBlock(L, 1)
	L.Push(lua.LNumber(block.Blockdata.Timestamp))

	return 1
}
func blockGetHeight(L *lua.LState) int {
	block := checkBlock(L, 1)
	L.Push(lua.LNumber(block.Blockdata.Height))

	return 1
}

func blockGetBits(L *lua.LState) int {
	block := checkBlock(L, 1)
	L.Push(lua.LNumber(block.Blockdata.Bits))

	return 1
}
