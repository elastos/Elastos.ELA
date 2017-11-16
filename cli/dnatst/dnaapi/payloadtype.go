package dnaapi

import (
	. "DNA_POW/common"
	"DNA_POW/core/transaction/payload"
	"DNA_POW/crypto"
	"encoding/hex"
	"fmt"

	"github.com/yuin/gopher-lua"
)

const (
	luaBookKeeperTypeName     = "bookkeeper"
	luaBookKeepingTypeName    = "bookkeeping"
	luaCoinBaseTypeName       = "coinbase"
	luaIssueAssetTypeName     = "issueasset"
	luaTransferAssetTypeName  = "transferasset"
	luaRegisterAssetTypeName  = "registerasset"
	luaRecordTypeName         = "record"
	luaDataFileTypeName       = "datafile"
	luaPrivacyPayloadTypeName = "privacypayload"
	luaDeployCodeTypeName     = "deploycode"
)

// Registers my person type to given L.
func RegisterBookKeeperType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaBookKeeperTypeName)
	L.SetGlobal("bookkeeper", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newBookKeeper))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), bookkeeperMethods))
}

// Constructor
func newBookKeeper(L *lua.LState) int {
	pubkeyStr := L.ToString(1)
	action := L.ToInt(2)
	certStr := L.ToString(3)
	issuerStr := L.ToString(4)

	pubkeyBytes, _ := hex.DecodeString(pubkeyStr)
	pubkey, _ := crypto.DecodePoint(pubkeyBytes)

	cert, _ := hex.DecodeString(certStr)

	issuerBytes, _ := hex.DecodeString(issuerStr)
	issuer, _ := crypto.DecodePoint(issuerBytes)

	bkper := &payload.BookKeeper{
		PubKey: pubkey,
		Action: payload.BookKeeperAction(action),
		Cert:   cert,
		Issuer: issuer,
	}
	ud := L.NewUserData()
	ud.Value = bkper
	L.SetMetatable(ud, L.GetTypeMetatable(luaBookKeeperTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *BookKeeper and returns this *BookKeeper.
func checkBookKeeper(L *lua.LState, idx int) *payload.BookKeeper {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.BookKeeper); ok {
		return v
	}
	L.ArgError(1, "BookKeeper expected")
	return nil
}

var bookkeeperMethods = map[string]lua.LGFunction{
	"get": bookkeeperGet,
}

// Getter and setter for the Person#Name
func bookkeeperGet(L *lua.LState) int {
	p := checkBookKeeper(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterBookKeepingType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaBookKeepingTypeName)
	L.SetGlobal("bookkeeping", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newBookKeeping))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), bookkeepingMethods))
}

// Constructor
func newBookKeeping(L *lua.LState) int {
	nonce := L.ToInt64(1)

	bkping := &payload.BookKeeping{
		Nonce: uint64(nonce),
	}
	ud := L.NewUserData()
	ud.Value = bkping
	L.SetMetatable(ud, L.GetTypeMetatable(luaBookKeepingTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *BookKeeping and
// returns this *BookKeeping.
func checkBookKeeping(L *lua.LState, idx int) *payload.BookKeeping {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.BookKeeping); ok {
		return v
	}
	L.ArgError(1, "BookKeeping expected")
	return nil
}

var bookkeepingMethods = map[string]lua.LGFunction{
	"get": bookkeepingGet,
}

// Getter and setter for the Person#Name
func bookkeepingGet(L *lua.LState) int {
	p := checkBookKeeping(L, 1)
	fmt.Println(p)

	return 0
}

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
func RegisterIssueAssetType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaIssueAssetTypeName)
	L.SetGlobal("issueasset", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newIssueAsset))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), issueassetMethods))
}

// Constructor
func newIssueAsset(L *lua.LState) int {
	ia := &payload.IssueAsset{}
	ud := L.NewUserData()
	ud.Value = ia
	L.SetMetatable(ud, L.GetTypeMetatable(luaIssueAssetTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *IssueAsset and
// returns this *IssueAsset.
func checkIssueAsset(L *lua.LState, idx int) *payload.IssueAsset {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.IssueAsset); ok {
		return v
	}
	L.ArgError(1, "IssueAsset expected")
	return nil
}

var issueassetMethods = map[string]lua.LGFunction{
	"get": issueassetGet,
}

// Getter and setter for the Person#Name
func issueassetGet(L *lua.LState) int {
	p := checkIssueAsset(L, 1)
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
	pubkeyStr := L.ToString(3)
	pubkeyBytes, _ := hex.DecodeString(pubkeyStr)
	issuer, _ := crypto.DecodePoint(pubkeyBytes)

	cntlStr := L.ToString(4)
	//cntlSlice, _ := hex.DecodeString(cntlStr)
	cntl, _ := ToScriptHash(cntlStr)

	ra := &payload.RegisterAsset{
		Asset:      ast,
		Amount:     amt,
		Issuer:     issuer,
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
func RegisterDataFileType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaDataFileTypeName)
	L.SetGlobal("datafile", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newDataFile))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), datafileMethods))
}

// Constructor
func newDataFile(L *lua.LState) int {
	ipfsPath := L.ToString(1)
	fileName := L.ToString(2)
	note := L.ToString(3)
	pubkeyStr := L.ToString(4)
	pubkeyBytes, _ := hex.DecodeString(pubkeyStr)
	issuer, _ := crypto.DecodePoint(pubkeyBytes)

	df := &payload.DataFile{
		IPFSPath: ipfsPath,
		Filename: fileName,
		Note:     note,
		Issuer:   issuer,
	}
	ud := L.NewUserData()
	ud.Value = df
	L.SetMetatable(ud, L.GetTypeMetatable(luaDataFileTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *DataFile and
// returns this *DataFile.
func checkDataFile(L *lua.LState, idx int) *payload.DataFile {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.DataFile); ok {
		return v
	}
	L.ArgError(1, "DataFile expected")
	return nil
}

var datafileMethods = map[string]lua.LGFunction{
	"get": datafileGet,
}

// Getter and setter for the Person#Name
func datafileGet(L *lua.LState) int {
	p := checkDataFile(L, 1)
	fmt.Println(p)

	return 0
}

// Registers my person type to given L.
func RegisterPrivacyPayloadType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaPrivacyPayloadTypeName)
	L.SetGlobal("privacypayload", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newPrivacyPayload))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), privacypayloadMethods))
}

// Constructor
func newPrivacyPayload(L *lua.LState) int {
	wallet := checkClient(L, 1)
	acc, _ := wallet.GetDefaultAccount()
	toPubkeyBytes, _ := hex.DecodeString(L.ToString(2))
	toPubkey, _ := crypto.DecodePoint(toPubkeyBytes)
	data, _ := hex.DecodeString(L.ToString(3))

	priv := &payload.PrivacyPayload{
		PayloadType: payload.RawPayload,
		EncryptType: payload.ECDH_AES256,
		EncryptAttr: &payload.EcdhAes256{
			FromPubkey: acc.PublicKey,
			ToPubkey:   toPubkey,
		},
	}
	priv.Payload, _ = priv.EncryptAttr.Encrypt(data, acc.PrivateKey)

	ud := L.NewUserData()
	ud.Value = priv
	L.SetMetatable(ud, L.GetTypeMetatable(luaPrivacyPayloadTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *PrivacyPayload and
// returns this *PrivacyPayload.
func checkPrivacyPayload(L *lua.LState, idx int) *payload.PrivacyPayload {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*payload.PrivacyPayload); ok {
		return v
	}
	L.ArgError(1, "PrivacyPayload expected")
	return nil
}

var privacypayloadMethods = map[string]lua.LGFunction{
	"get": privacypayloadGet,
}

// Getter and setter for the Person#Name
func privacypayloadGet(L *lua.LState) int {
	p := checkPrivacyPayload(L, 1)
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
