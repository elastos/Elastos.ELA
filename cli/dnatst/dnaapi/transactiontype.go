package dnaapi

import (
	"DNA_POW/account"
	"DNA_POW/core/contract"
	"DNA_POW/core/signature"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	"bytes"
	"encoding/hex"

	"fmt"

	"github.com/yuin/gopher-lua"
)

const luaTransactionTypeName = "transaction"

// Registers my person type to given L.
func RegisterTransactionType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaTransactionTypeName)
	L.SetGlobal("transaction", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newTransaction))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), transactionMethods))
}

// Constructor
//	TxType         TransactionType
//	PayloadVersion byte
//	Payload        Payload
//	Attributes     []*TxAttribute
//	UTXOInputs     []*UTXOTxInput
//	BalanceInputs  []*BalanceTxInput
//	Outputs        []*TxOutput
//	LockTime       uint32
func newTransaction(L *lua.LState) int {
	txType := tx.TransactionType(L.ToInt(1))
	payloadVersion := byte(L.ToInt(2))
	ud := L.CheckUserData(3)
	var pload tx.Payload
	switch ud.Value.(type) {
	case *payload.CoinBase:
		pload, _ = ud.Value.(*payload.CoinBase)
	case *payload.RegisterAsset:
		pload, _ = ud.Value.(*payload.RegisterAsset)
	case *payload.TransferAsset:
		pload, _ = ud.Value.(*payload.TransferAsset)
	case *payload.Record:
		pload, _ = ud.Value.(*payload.Record)
	case *payload.DeployCode:
		pload, _ = ud.Value.(*payload.DeployCode)
	}

	lockTime := uint32(L.ToInt(4))

	txn := &tx.Transaction{
		TxType:         txType,
		PayloadVersion: payloadVersion,
		Payload:        pload,
		Attributes:     []*tx.TxAttribute{},
		UTXOInputs:     []*tx.UTXOTxInput{},
		BalanceInputs:  []*tx.BalanceTxInput{},
		Outputs:        []*tx.TxOutput{},
		LockTime:       lockTime,
	}
	udn := L.NewUserData()
	udn.Value = txn
	L.SetMetatable(udn, L.GetTypeMetatable(luaTransactionTypeName))
	L.Push(udn)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Transaction and returns this *Transaction.
func checkTransaction(L *lua.LState, idx int) *tx.Transaction {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*tx.Transaction); ok {
		return v
	}
	L.ArgError(1, "Transaction expected")
	return nil
}

var transactionMethods = map[string]lua.LGFunction{
	"appendtxin":    transactionAppendInput,
	"appendtxout":   transactionAppendOutput,
	"appendattr":    transactionAppendAttribute,
	"appendbalance": transactionAppendBalance,
	"get":           transactionGet,
	"sign":          transactionSign,
	"hash":          transactionHash,
	"serialize":     transactionSerialize,
	"deserialize":   transactionDeserialize,
}

// Getter and setter for the Person#Name
func transactionGet(L *lua.LState) int {
	p := checkTransaction(L, 1)
	fmt.Println(p)

	return 0
}

func transactionAppendInput(L *lua.LState) int {
	p := checkTransaction(L, 1)
	input := checkUTXOTxInput(L, 2)
	p.UTXOInputs = append(p.UTXOInputs, input)

	return 0
}

func transactionAppendAttribute(L *lua.LState) int {
	p := checkTransaction(L, 1)
	attr := checkTxAttribute(L, 2)
	p.Attributes = append(p.Attributes, attr)
	return 0
}

func transactionAppendOutput(L *lua.LState) int {
	p := checkTransaction(L, 1)
	output := checkTxOutput(L, 2)
	p.Outputs = append(p.Outputs, output)

	return 0
}

func transactionAppendBalance(L *lua.LState) int {
	p := checkTransaction(L, 1)
	balance := checkBalanceTxInput(L, 2)
	p.BalanceInputs = append(p.BalanceInputs, balance)

	return 0
}

func transactionHash(L *lua.LState) int {
	tx := checkTransaction(L, 1)
	h := tx.Hash()
	hash := h.ToArrayReverse()

	L.Push(lua.LString(hex.EncodeToString(hash)))

	return 1
}

func transactionSign(L *lua.LState) int {
	tx := checkTransaction(L, 1)
	wallet := checkClient(L, 2)
	acc, _ := wallet.GetDefaultAccount()

	signTransaction(acc, tx)
	return 0
}

func signTransaction(signer *account.Account, txn *tx.Transaction) error {
	signature, err := signature.SignBySigner(txn, signer)
	if err != nil {
		fmt.Println("SignBySigner failed")
		return err
	}
	transactionContract, err := contract.CreateSignatureContract(signer.PubKey())
	if err != nil {
		fmt.Println("CreateSignatureContract failed")
		return err
	}
	transactionContractContext := &contract.ContractContext{
		Data:       txn,
		Codes:      make([][]byte, 1),
		Parameters: make([][][]byte, 1),
	}

	if err := transactionContractContext.AddContract(transactionContract, signer.PubKey(), signature); err != nil {
		fmt.Println("saveContract failed")
		return err
	}
	txn.SetPrograms(transactionContractContext.GetPrograms())
	return nil
}

func transactionSerialize(L *lua.LState) int {
	txn := checkTransaction(L, 1)

	var buffer bytes.Buffer
	txn.Serialize(&buffer)
	txHex := hex.EncodeToString(buffer.Bytes())

	L.Push(lua.LNumber(len(buffer.Bytes())))
	L.Push(lua.LString(txHex))
	return 2
}

func transactionDeserialize(L *lua.LState) int {
	txn := checkTransaction(L, 1)
	txSlice, _ := hex.DecodeString(L.ToString(2))

	txn.Deserialize(bytes.NewReader(txSlice))

	return 1
}
