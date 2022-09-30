package transaction

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/auxpow"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	log2 "github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/stretchr/testify/suite"
	"math/big"
	"math/rand"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type txValidatorTestSuite struct {
	suite.Suite

	ELA               int64
	foundationAddress common.Uint168
	HeightVersion1    uint32
	CurrentHeight     uint32
	Chain             *blockchain.BlockChain
	OriginalLedger    *blockchain.Ledger
}

func init() {
	testing.Init()

	functions.GetTransactionByTxType = GetTransaction
	functions.GetTransactionByBytes = GetTransactionByBytes
	functions.CreateTransaction = CreateTransaction
	functions.GetTransactionParameters = GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

type txValidatorSpecialTxTestSuite struct {
	suite.Suite

	originalLedger     *blockchain.Ledger
	arbitrators        *state.ArbitratorsMock
	arbitratorsPriKeys [][]byte
	Chain              *blockchain.BlockChain
}

func (s *txValidatorSpecialTxTestSuite) SetupSuite() {
	arbitratorsStr := []string{
		"023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
		"030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
		"0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
		"03e281f89d85b3a7de177c240c4961cb5b1f2106f09daa42d15874a38bbeae85dd",
		"0393e823c2087ed30871cbea9fa5121fa932550821e9f3b17acef0e581971efab0",
	}
	arbitratorsPrivateKeys := []string{
		"e372ca1032257bb4be1ac99c4861ec542fd55c25c37f5f58ba8b177850b3fdeb",
		"e6deed7e23406e2dce7b01e85bcb33872a47b6200ca983fcf0540dff284923b0",
		"4441968d02a5df4dbc08ca11da2acc86c980e5fe9ff250450a80fd7421d2b0f1",
		"0b14a04e203301809feccc61dbf4e745203a3263d29a4b4091aaa138ba5fb26d",
		"0c11ebca60af2a09ac13dd84fd29c03b99cd086a08a69a9e5b87255fd9cf2eee",
		"ad44a6d5a5d1f7cafa2fa82c719108e9814ff5c71078e1cafa9f734343a2f806",
	}
	log2.NewDefault(test.NodeLogPath, 0, 0, 0)

	s.arbitrators = &state.ArbitratorsMock{
		CurrentArbitrators: make([]state.ArbiterMember, 0),
		MajorityCount:      3,
	}
	for _, v := range arbitratorsStr {
		a, _ := common.HexStringToBytes(v)
		ar, _ := state.NewOriginArbiter(a)
		s.arbitrators.CurrentArbitrators = append(
			s.arbitrators.CurrentArbitrators, ar)
	}
	s.arbitrators.Snapshot = []*state.CheckPoint{
		{
			CurrentArbitrators: s.arbitrators.CurrentArbitrators,
		},
	}

	for _, v := range arbitratorsPrivateKeys {
		a, _ := common.HexStringToBytes(v)
		s.arbitratorsPriKeys = append(s.arbitratorsPriKeys, a)
	}

	chainStore, err := blockchain.NewChainStore(filepath.Join(test.DataPath, "special"), &config.DefaultParams)
	if err != nil {
		s.Error(err)
	}
	s.Chain, err = blockchain.New(chainStore, &config.DefaultParams,
		state.NewState(&config.DefaultParams, nil, nil, nil, nil,
			nil, nil,
			nil, nil, nil, nil, nil), nil)
	if err != nil {
		s.Error(err)
	}
	blockchain.DefaultLedger = &blockchain.Ledger{Arbitrators: s.arbitrators, Store: chainStore}
	s.originalLedger = blockchain.DefaultLedger
}

type transactionSuite struct {
	suite.Suite

	InputNum   int
	OutputNum  int
	AttrNum    int
	ProgramNum int
}

func (s *transactionSuite) SetupSuite() {
	s.InputNum = 10
	s.OutputNum = 10
	s.AttrNum = 10
	s.ProgramNum = 10
}

func (s *txValidatorSpecialTxTestSuite) TearDownSuite() {
	s.Chain.GetDB().Close()
	blockchain.DefaultLedger = s.originalLedger
}

func TestTxValidatorSpecialTxSuite(t *testing.T) {
	suite.Run(t, new(txValidatorSpecialTxTestSuite))
}

func (s *txValidatorTestSuite) SetupSuite() {
	log2.NewDefault(test.NodeLogPath, 0, 0, 0)

	params := &config.DefaultParams
	blockchain.FoundationAddress = params.Foundation
	s.foundationAddress = params.Foundation

	chainStore, err := blockchain.NewChainStore(filepath.Join(test.DataPath, "txvalidator"), params)
	if err != nil {
		s.Error(err)
	}
	s.Chain, err = blockchain.New(chainStore, params,
		state.NewState(params, nil, nil, nil,
			func() bool { return false },
			nil, nil,
			nil, nil, nil, nil, nil),
		crstate.NewCommittee(params))
	if err != nil {
		s.Error(err)
	}
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          chainStore.GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})

	if err := s.Chain.Init(nil); err != nil {
		s.Error(err)
	}

	arbiters, err := state.NewArbitrators(params,
		nil, nil, nil,
		nil, nil, nil, nil, nil)
	if err != nil {
		s.Fail("initialize arbitrator failed")
	}
	arbiters.RegisterFunction(chainStore.GetHeight,
		func() *common.Uint256 { return &common.Uint256{} },
		func(height uint32) (*types.Block, error) {
			return nil, nil
		}, nil)
	blockchain.DefaultLedger = &blockchain.Ledger{Arbitrators: arbiters, Store: chainStore, Committee: s.Chain.GetCRCommittee()}
	s.OriginalLedger = blockchain.DefaultLedger
}

func (s *txValidatorTestSuite) TearDownSuite() {
	s.Chain.GetDB().Close()
	blockchain.DefaultLedger = s.OriginalLedger
}

func TestTxValidatorSuite(t *testing.T) {
	suite.Run(t, new(txValidatorTestSuite))
}

func getCodeByPubKeyStr(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}
func getCodeHexStr(publicKey string) string {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	codeHexStr := common.BytesToHexString(redeemScript)
	return codeHexStr
}

func randomUint256() *common.Uint256 {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	result, _ := common.Uint256FromBytes(randBytes)

	return result
}

func CreateTransactionByType(ori interfaces.Transaction, chain *blockchain.BlockChain) interfaces.Transaction {
	tx := functions.CreateTransaction(
		ori.Version(),
		ori.TxType(),
		ori.PayloadVersion(),
		ori.Payload(),
		ori.Attributes(),
		ori.Inputs(),
		ori.Outputs(),
		ori.LockTime(),
		ori.Programs(),
	)

	tx.SetParameters(&TransactionParameters{
		Transaction: tx,
		BlockHeight: chain.BestChain.Height,
		TimeStamp:   chain.BestChain.Timestamp,
		Config:      chain.GetParams(),
		BlockChain:  chain,
	})

	return tx
}

func randomString() string {
	a := make([]byte, 20)
	rand.Read(a)
	return common.BytesToHexString(a)
}

func randomBytes(len int) []byte {
	a := make([]byte, len)
	rand.Read(a)
	return a
}

func randomName(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randomUint168() *common.Uint168 {
	randBytes := make([]byte, 21)
	rand.Read(randBytes)
	result, _ := common.Uint168FromBytes(randBytes)

	return result
}

func getDepositAddress(publicKeyStr string) (*common.Uint168, error) {
	publicKey, _ := common.HexStringToBytes(publicKeyStr)
	hash, err := contract.PublicKeyToDepositProgramHash(publicKey)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func getValideCode(publicKeyStr string) []byte {
	publicKey1, _ := common.HexStringToBytes(publicKeyStr)
	pk1, _ := crypto.DecodePoint(publicKey1)
	ct1, _ := contract.CreateStandardContract(pk1)
	return ct1.Code
}

func getValidCID(publicKeyStr string) *common.Uint168 {
	code := getValideCode(publicKeyStr)
	return getCID(code)
}

func getCID(code []byte) *common.Uint168 {
	ct1, _ := contract.CreateCRIDContractByCode(code)
	return ct1.ToProgramHash()
}

func decodeMessage(m string, t *testing.T) (msg [32]byte) {
	message, err := hex.DecodeString(m)
	if err != nil && t != nil {
		t.Fatalf("Unexpected error from hex.DecodeString(%s): %v", m, err)
	}
	copy(msg[:], message)
	return
}

func decodePublicKey(pk string, t *testing.T) (pubKey [33]byte) {
	publicKey, err := hex.DecodeString(pk)
	if err != nil && t != nil {
		t.Fatalf("Unexpected error from hex.DecodeString(%s): %v", pk, err)
	}
	copy(pubKey[:], publicKey)
	return
}

func decodePrivateKey(d string, t *testing.T) *big.Int {
	privKey, ok := new(big.Int).SetString(d, 16)
	if !ok && t != nil {
		t.Fatalf("Unexpected error from new(big.Int).SetString(%s, 16)", d)
	}
	return privKey
}

func randomSignature() []byte {
	randBytes := make([]byte, 64)
	rand.Read(randBytes)

	return randBytes
}

func createBudgets(n int) []payload.Budget {
	budgets := make([]payload.Budget, 0)
	for i := 0; i < n; i++ {
		var budgetType = payload.NormalPayment
		if i == 0 {
			budgetType = payload.Imprest
		}
		if i == n-1 {
			budgetType = payload.FinalPayment
		}
		budget := &payload.Budget{
			Stage:  byte(i),
			Type:   budgetType,
			Amount: common.Fixed64((i + 1) * 1e8),
		}
		budgets = append(budgets, *budget)
	}
	return budgets
}

func randomFix64() common.Fixed64 {
	var randNum int64
	binary.Read(crand.Reader, binary.BigEndian, &randNum)
	return common.Fixed64(randNum)
}

func randomBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 0
}

func randomOutputInfo() *common2.OutputInfo {
	return &common2.OutputInfo{
		Recipient: *randomUint168(),
		Amount:    randomFix64(),
	}
}

func (s *txValidatorTestSuite) getCRMember(publicKeyStr, privateKeyStr, nickName string) *crstate.CRMember {
	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	code1 := getCodeByPubKeyStr(publicKeyStr1)
	did1, _ := blockchain.GetDIDFromCode(code1)

	crInfoPayload := payload.CRInfo{
		Code:     code1,
		DID:      *did1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1

	return &crstate.CRMember{
		Info: crInfoPayload,
	}
}

func buildTx() interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		randomInputs(),
		randomOutputs(),
		0,
		[]*program.Program{},
	)
	return tx
}

func randomInputs() []*common2.Input {
	num := rand.Intn(100) + 1
	inputs := make([]*common2.Input, 0, num)
	for i := 0; i < num; i++ {
		var txID common.Uint256
		rand.Read(txID[:])
		index := rand.Intn(100)
		inputs = append(inputs, &common2.Input{
			Previous: *common2.NewOutPoint(txID, uint16(index)),
		})
	}
	return inputs
}

func randomOutputs() []*common2.Output {
	num := rand.Intn(100) + 1
	outputs := make([]*common2.Output, 0, num)
	var asset common.Uint256
	rand.Read(asset[:])
	for i := 0; i < num; i++ {
		var addr common.Uint168
		rand.Read(addr[:])
		outputs = append(outputs, &common2.Output{
			AssetID:     asset,
			Value:       common.Fixed64(rand.Int63()),
			OutputLock:  0,
			ProgramHash: addr,
		})
	}
	return outputs
}

func randomBlockHeader() *common2.Header {
	return &common2.Header{
		Version:    rand.Uint32(),
		Previous:   *randomUint256(),
		MerkleRoot: *randomUint256(),
		Timestamp:  rand.Uint32(),
		Bits:       rand.Uint32(),
		Nonce:      rand.Uint32(),
		Height:     rand.Uint32(),
		AuxPow: auxpow.AuxPow{
			AuxMerkleBranch: []common.Uint256{
				*randomUint256(),
				*randomUint256(),
			},
			AuxMerkleIndex: rand.Int(),
			ParCoinbaseTx: auxpow.BtcTx{
				Version: rand.Int31(),
				TxIn: []*auxpow.BtcTxIn{
					{
						PreviousOutPoint: auxpow.BtcOutPoint{
							Hash:  *randomUint256(),
							Index: rand.Uint32(),
						},
						SignatureScript: []byte(strconv.FormatUint(rand.Uint64(), 10)),
						Sequence:        rand.Uint32(),
					},
					{
						PreviousOutPoint: auxpow.BtcOutPoint{
							Hash:  *randomUint256(),
							Index: rand.Uint32(),
						},
						SignatureScript: []byte(strconv.FormatUint(rand.Uint64(), 10)),
						Sequence:        rand.Uint32(),
					},
				},
				TxOut: []*auxpow.BtcTxOut{
					{
						Value:    rand.Int63(),
						PkScript: []byte(strconv.FormatUint(rand.Uint64(), 10)),
					},
					{
						Value:    rand.Int63(),
						PkScript: []byte(strconv.FormatUint(rand.Uint64(), 10)),
					},
				},
				LockTime: rand.Uint32(),
			},
			ParCoinBaseMerkle: []common.Uint256{
				*randomUint256(),
				*randomUint256(),
			},
			ParMerkleIndex: rand.Int(),
			ParBlockHeader: auxpow.BtcHeader{
				Version:    rand.Uint32(),
				Previous:   *randomUint256(),
				MerkleRoot: *randomUint256(),
				Timestamp:  rand.Uint32(),
				Bits:       rand.Uint32(),
				Nonce:      rand.Uint32(),
			},
			ParentHash: *randomUint256(),
		},
	}
}

func randomPublicKey() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}

func TestTransactionSuite(t *testing.T) {
	suite.Run(t, new(transactionSuite))
}
