// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	math "math/rand"
	"sort"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction2.GetTransaction
	functions.GetTransactionByBytes = transaction2.GetTransactionByBytes
	functions.CreateTransaction = transaction2.CreateTransaction
	functions.GetTransactionParameters = transaction2.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

type act interface {
	RedeemScript() []byte
	ProgramHash() *common.Uint168
	Sign(data []byte) ([]byte, error)
}

type account struct {
	private      []byte
	public       *crypto.PublicKey
	redeemScript []byte
	programHash  *common.Uint168
}

func init() {
	testing.Init()
}

func (a *account) RedeemScript() []byte {
	return a.redeemScript
}

func (a *account) ProgramHash() *common.Uint168 {
	return a.programHash
}

func (a *account) Sign(data []byte) ([]byte, error) {
	return sign(a.private, data)
}

type multiAccount struct {
	accounts     []*account
	redeemScript []byte
	programHash  *common.Uint168
}

type schnorAccount struct {
	multiAccount
	sumPublicKey [33]byte
	privateKeys  []*big.Int
}

func (a *multiAccount) RedeemScript() []byte {
	return a.redeemScript
}

func (a *multiAccount) ProgramHash() *common.Uint168 {
	return a.programHash
}

func (a *multiAccount) Sign(data []byte) ([]byte, error) {
	var signatures []byte
	for _, act := range a.accounts {
		signature, err := sign(act.private, data)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, signature...)
	}
	return signatures, nil
}

func TestCheckCheckSigSignature(t *testing.T) {
	var tx interfaces.Transaction

	tx = buildTx()
	data := getData(tx)
	act := newAccount(t)
	signature, err := act.Sign(data)
	if err != nil {
		t.Errorf("Generate signature failed, error %s", err.Error())
	}

	// Normal
	err = blockchain.CheckStandardSignature(program.Program{Code: act.redeemScript, Parameter: signature}, data)
	assert.NoError(t, err, "[CheckChecksigSignature] failed, %v", err)

	// invalid signature length
	var fakeSignature = make([]byte, crypto.SignatureScriptLength-math.Intn(64)-1)
	rand.Read(fakeSignature)
	err = blockchain.CheckStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, data)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid signature length")
	assert.Equal(t, "invalid signature length", err.Error())

	// invalid signature content
	fakeSignature = make([]byte, crypto.SignatureScriptLength)
	err = blockchain.CheckStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, data)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid signature content")
	assert.Equal(t, "[Validation], Verify failed.", err.Error())

	// invalid data content
	err = blockchain.CheckStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, nil)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid data content")
	assert.Equal(t, "[Validation], Verify failed.", err.Error())
}

func TestCheckMultiSigSignature(t *testing.T) {
	var tx interfaces.Transaction

	tx = buildTx()
	data := getData(tx)

	act := newMultiAccount(math.Intn(2)+3, t)
	signature, err := act.Sign(data)
	assert.NoError(t, err, "Generate signature failed, error %v", err)

	// Normal
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature}, data)
	assert.NoError(t, err, "[CheckMultisigSignature] failed, %v", err)

	// invalid redeem script M < 1
	fakeCode := make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[0] = fakeCode[0] - fakeCode[0] + crypto.PUSH1 - 1
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] code with M < 1 passed")
	assert.Equal(t, "invalid multi sign script code", err.Error())

	// invalid redeem script M > N
	copy(fakeCode, act.redeemScript)
	fakeCode[0] = fakeCode[len(fakeCode)-2] - crypto.PUSH1 + 2
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] code with M > N passed")
	assert.Equal(t, "invalid multi sign script code", err.Error())

	// invalid redeem script length not enough
	copy(fakeCode, act.redeemScript)
	for len(fakeCode) >= crypto.MinMultiSignCodeLength {
		fakeCode = append(fakeCode[:1], fakeCode[crypto.PublicKeyScriptLength:]...)
	}
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid length code passed")
	assert.Equal(t, "not a valid multi sign transaction code, length not enough", err.Error())

	// invalid redeem script N not equal to public keys count
	fakeCode = make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[len(fakeCode)-2] = fakeCode[len(fakeCode)-2] + 1
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid redeem script N not equal to public keys count")
	assert.Equal(t, "invalid multi sign public key script count", err.Error())

	// invalid redeem script wrong public key
	fakeCode = make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[2] = 0x01
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid redeem script wrong public key")
	assert.Equal(t, "the encodeData format is error", err.Error())

	// invalid signature length not match
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature[1+math.Intn(64):]}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature length not match")
	assert.Equal(t, "invalid multi sign signatures, length not match", err.Error())

	// invalid signature not enough
	cut := len(signature)/crypto.SignatureScriptLength - int(act.redeemScript[0]-crypto.PUSH1)
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature[65*cut:]}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature not enough")
	assert.Equal(t, "invalid signatures, not enough signatures", err.Error())

	// invalid signature too many
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: act.redeemScript,
		Parameter: append(signature[:65], signature...)}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature too many")
	assert.Equal(t, "invalid signatures, too many signatures", err.Error())

	// invalid signature duplicate
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: act.redeemScript,
		Parameter: append(signature[:65], signature[:len(signature)-65]...)}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature duplicate")
	assert.Equal(t, "duplicated signatures", err.Error())

	// invalid signature fake signature
	signature, err = newMultiAccount(math.Intn(2)+3, t).Sign(data)
	assert.NoError(t, err, "Generate signature failed, error %v", err)
	err = blockchain.CheckMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature fake signature")
}

func TestSchnorrRunProgramsOrigin(t *testing.T) {
	var testCases = []struct {
		d           string //private key
		pk          string //public key
		m           string //message
		sig         string //expect sign
		result      bool   //expect result
		err         error  //expect err
		description string //expect description. not used for now
	}{
		{
			"0000000000000000000000000000000000000000000000000000000000000001",
			"0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"787A848E71043D280C50470E8E1532B2DD5D20EE912A45DBDD2BD1DFBF187EF67031A98831859DC34DFFEEDDA86831842CCD0079E1F92AF177F7F22CC1DCED05",
			true,
			nil,
			"",
		},
		{
			"B7E151628AED2A6ABF7158809CF4F3C762E7160F38B4DA56A784D9045190CFEF",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			true,
			nil,
			"",
		},
		{
			"C90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B14E5C7",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"5E2D58D8B3BCDF1ABADEC7829054F90DDA9805AAB56C77333024B9D0A508B75C",
			"00DA9B08172A9B6F0466A2DEFD817F2D7AB437E0D253CB5395A963866B3574BE00880371D01766935B92D2AB4CD5C8A2A5837EC57FED7660773A05F0DE142380",
			true,
			nil,
			"",
		},
		{
			"6d6c66873739bc7bfb3526629670d0ea357e92cc4581490d62779ae15f6b787b",
			"026d7f1d87ab3bbc8bc01f95d9aece1e659d6e33c880f8efa65facf83e698bbbf7",
			"b2f0cd8ecb23c1710903f872c31b0fd37e15224af457722a87c5e0c7f50fffb3",
			"68ca1cc46f291a385e7c255562068357f964532300beadffb72dd93668c0c1cac8d26132eb3200b86d66de9c661a464c6b2293bb9a9f5b966e53ca736c7e504f",
			true,
			nil,
			"",
		},
	}

	pks := []*big.Int{}
	var (
		m  [32]byte
		pk [33]byte
	)

	Pxs, Pys := []*big.Int{}, []*big.Int{}
	for i, test := range testCases {
		if test.d == "" {
			continue
		}

		privKey := decodePrivateKey(test.d, t)
		MyPublicKey := decodePublicKey(test.pk, t)
		fmt.Println("MyPublicKey", MyPublicKey)
		pubKey, err := crypto.DecodePoint(MyPublicKey[:])
		fmt.Println(err)
		fmt.Println(pubKey)

		pks = append(pks, privKey)

		if i == 0 {
			m = decodeMessage(test.m, t)
		}

		Px, Py := crypto.Curve.ScalarBaseMult(privKey.Bytes())
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}
	t.Run("Can sign and verify two aggregated signatures over same message", func(t *testing.T) {

		Px, Py := crypto.Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		//this is the sum pk elliptic.
		copy(pk[:], crypto.Marshal(crypto.Curve, Px, Py))

		publicKey1, _ := crypto.DecodePoint(pk[:])
		fmt.Println(publicKey1)
		//var err error
		var tx interfaces.Transaction
		//var acts []act
		var hashes []common.Uint168
		var programs []*program.Program

		tx = buildTx()
		data := getData(tx)
		msg := common.Sha256D(data)
		sig, err := crypto.AggregateSignatures(pks[:2], msg)
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:2], m, err)
		}

		publicKey, _ := crypto.DecodePoint(pk[:])
		redeemscript, err := contract.CreateSchnorrRedeemScript(publicKey)
		if err != nil {
			fmt.Println("err ", err)
		}

		c, err := contract.CreateSchnorrContract(publicKey)
		if err != nil {
			t.Errorf("Create standard contract failed, error %s", err.Error())
		}

		programHash := c.ToProgramHash()
		hashes = append(hashes, *programHash)
		programs = append(programs, &program.Program{Code: redeemscript, Parameter: sig[:]})
		err = blockchain.RunPrograms(data, hashes[0:1], programs)
		assert.NoError(t, err, "[RunProgram] passed with 1 checksig program")

	})
}

//TestAggregateSignatures TestSchnorrRunProgramsOrigin
func TestAggregateSignatures(t *testing.T) {
	var testCases = []struct {
		d           string //private key
		pk          string //public key
		m           string //message
		sig         string //expect sign
		result      bool   //expect result
		err         error  //expect err
		description string //expect description. not used for now
	}{
		{
			"0000000000000000000000000000000000000000000000000000000000000001",
			"0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"787A848E71043D280C50470E8E1532B2DD5D20EE912A45DBDD2BD1DFBF187EF67031A98831859DC34DFFEEDDA86831842CCD0079E1F92AF177F7F22CC1DCED05",
			true,
			nil,
			"",
		},
		{
			"B7E151628AED2A6ABF7158809CF4F3C762E7160F38B4DA56A784D9045190CFEF",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			true,
			nil,
			"",
		},
		{
			"C90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B14E5C7",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"5E2D58D8B3BCDF1ABADEC7829054F90DDA9805AAB56C77333024B9D0A508B75C",
			"00DA9B08172A9B6F0466A2DEFD817F2D7AB437E0D253CB5395A963866B3574BE00880371D01766935B92D2AB4CD5C8A2A5837EC57FED7660773A05F0DE142380",
			true,
			nil,
			"",
		},
		{
			"6d6c66873739bc7bfb3526629670d0ea357e92cc4581490d62779ae15f6b787b",
			"026d7f1d87ab3bbc8bc01f95d9aece1e659d6e33c880f8efa65facf83e698bbbf7",
			"b2f0cd8ecb23c1710903f872c31b0fd37e15224af457722a87c5e0c7f50fffb3",
			"68ca1cc46f291a385e7c255562068357f964532300beadffb72dd93668c0c1cac8d26132eb3200b86d66de9c661a464c6b2293bb9a9f5b966e53ca736c7e504f",
			true,
			nil,
			"",
		},
	}

	pks := []*big.Int{}
	var (
		m  [32]byte
		pk [33]byte
	)

	Pxs, Pys := []*big.Int{}, []*big.Int{}
	for i, test := range testCases {
		if test.d == "" {
			continue
		}

		privKey := decodePrivateKey(test.d, t)
		pks = append(pks, privKey)

		if i == 0 {
			m = decodeMessage(test.m, t)
		}

		Px, Py := crypto.Curve.ScalarBaseMult(privKey.Bytes())
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}

	t.Run("Can sign and verify two aggregated signatures over same message", func(t *testing.T) {
		sig, err := crypto.AggregateSignatures(pks[:2], m)
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:2], m, err)
		}

		Px, Py := crypto.Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		copy(pk[:], crypto.Marshal(crypto.Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "03c0cba209687e8f213f9b00ba0202d98ef17d189bd0d4c5decd7382b7a63bc64e"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := crypto.SchnorrVerify(pk, m, sig)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, sig, err)
		}

		// then
		if !observed {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, sig, observed, true)
		}
	})

	t.Run("Can sign and verify two more aggregated signatures over same message", func(t *testing.T) {
		sig, err := crypto.AggregateSignatures(pks[1:3], m)
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[1:3], m, err)
		}

		Px, Py := crypto.Curve.Add(Pxs[1], Pys[1], Pxs[2], Pys[2])
		copy(pk[:], crypto.Marshal(crypto.Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "038c47ce8f4f20fd041a25ef78e872448340b1dc28f6539d4fe0126018f1b8e0ea"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := crypto.SchnorrVerify(pk, m, sig)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, sig, err)
		}

		// then
		if !observed {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, sig, observed, true)
		}
	})

	t.Run("Can sign and verify three aggregated signatures over same message", func(t *testing.T) {
		sig, err := crypto.AggregateSignatures(pks[:3], m)
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:3], m, err)
		}

		Px, Py := crypto.Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		Px, Py = crypto.Curve.Add(Px, Py, Pxs[2], Pys[2])
		copy(pk[:], crypto.Marshal(crypto.Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "02adde346abd5b690dae4ebbb74e59ea0e41cdc00c8a5b05b0057678e5e98f9f2e"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := crypto.SchnorrVerify(pk, m, sig)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, sig, err)
		}

		// then
		if !observed {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, sig, observed, true)
		}
	})

	t.Run("Can aggregate and verify example in README", func(t *testing.T) {
		privKey1 := decodePrivateKey("8e7b372c1ceea18883032d6cdbf67e7dee5e7507c30012af81cfb7e9b60c00cc", t)
		privKey2 := decodePrivateKey("3adfeb9c654863522b74cd0feeb5744102067e6a1a1867bb1dfdc080a2716858", t)
		m := decodeMessage("243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89", t)

		pks := []*big.Int{privKey1, privKey2}
		aggregatedSignature, err := crypto.AggregateSignatures(pks, m)
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks, m, err)
		}

		// verifying an aggregated signature
		pubKey1 := decodePublicKey("031c7ce0c8d4812b12a9a8988697f45351d83cba72ea9e16f227445599666ea415", t)
		pubKey2 := decodePublicKey("03460cfbe83d295c072c547e831ce224dd903f888a183a35d2888b7ab3a8666054", t)

		P1x, P1y := crypto.Unmarshal(crypto.Curve, pubKey1[:])
		P2x, P2y := crypto.Unmarshal(crypto.Curve, pubKey2[:])
		Px, Py := crypto.Curve.Add(P1x, P1y, P2x, P2y)

		copy(pk[:], crypto.Marshal(crypto.Curve, Px, Py))

		observed := hex.EncodeToString(pk[:])
		expected := "03486ca0cb4360d2b8c4be796772f77815932e18d3fe29ca81b5b30c41a1f3272e"

		// then
		if observed != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observed, expected)
		}

		result, err := crypto.SchnorrVerify(pk, m, aggregatedSignature)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, aggregatedSignature, err)
		}

		// then
		if !result {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, aggregatedSignature, observed, true)
		}
	})
}

func TestSchnorrRunPrograms(t *testing.T) {
	var err error
	var tx interfaces.Transaction
	var hashes []common.Uint168
	var programs []*program.Program
	var acts []act
	tx = buildTx()
	data := getData(tx)
	msg := common.Sha256D(data)
	//aggregate num from 2 to 36
	aggregateNum := math.Intn(34) + 2
	//random schnorrAccountNum to test schnorr sign
	schnorrAccountNum := 100
	var act *schnorAccount
	init := func(schnorrAccountNum int) {
		hashes = make([]common.Uint168, 0, schnorrAccountNum)
		programs = make([]*program.Program, 0, schnorrAccountNum)
		for i := 0; i < schnorrAccountNum; i++ {
			act = newSchnorrMultiAccount(aggregateNum, t)
			acts = append(acts, act)
			hashes = append(hashes, *act.ProgramHash())
			sig, err := crypto.AggregateSignatures(act.privateKeys[:aggregateNum], msg)
			if err != nil {
				fmt.Println("AggregateSignatures fail")
			}
			programs = append(programs, &program.Program{Code: act.RedeemScript(), Parameter: sig[:]})
		}
	}
	init(schnorrAccountNum)

	for index := 0; index < schnorrAccountNum; index++ {
		err = blockchain.RunPrograms(data, hashes[index:index+1], programs[index:index+1])
		if err != nil {
			fmt.Printf("AggregateSignatures index %d fail err %s \n", index, err.Error())
		} else {
			fmt.Printf("AggregateSignatures index %d passed \n", index)
		}
	}
	assert.NoError(t, err, "[RunProgram] passed with 1 checksig program")
}

func TestRunPrograms(t *testing.T) {
	var err error
	var tx interfaces.Transaction
	var acts []act
	var hashes []common.Uint168
	var programs []*program.Program

	tx = buildTx()
	data := getData(tx)
	// Normal
	num := math.Intn(90) + 10
	acts = make([]act, 0, num)
	init := func() {
		hashes = make([]common.Uint168, 0, num)
		programs = make([]*program.Program, 0, num)
		for i := 0; i < num; i++ {
			if math.Uint32()%2 == 0 {
				act := newAccount(t)
				acts = append(acts, act)
			} else {
				mact := newMultiAccount(math.Intn(2)+3, t)
				acts = append(acts, mact)
			}
			hashes = append(hashes, *acts[i].ProgramHash())
			signature, err := acts[i].Sign(data)
			if err != nil {
				t.Errorf("Generate signature failed, error %s", err.Error())
			}
			programs = append(programs, &program.Program{Code: acts[i].RedeemScript(), Parameter: signature})
		}
	}
	init()

	// 1 loop checksig
	var index int
	for i, act := range acts {
		switch act.(type) {
		case *account:
			index = i
			break
		}
	}
	err = blockchain.RunPrograms(data, hashes[index:index+1], programs[index:index+1])
	assert.NoError(t, err, "[RunProgram] passed with 1 checksig program")

	// 1 loop multisig
	for i, act := range acts {
		switch act.(type) {
		case *multiAccount:
			index = i
			break
		}
	}
	err = blockchain.RunPrograms(data, hashes[index:index+1], programs[index:index+1])
	assert.NoError(t, err, "[RunProgram] passed with 1 multisig program")

	// multiple programs
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.NoError(t, err, "[RunProgram] passed with multiple programs")

	// hashes count not equal to programs count
	init()
	removeIndex := math.Intn(num)
	hashes = append(hashes[:removeIndex], hashes[removeIndex+1:]...)
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with unmathed hashes")
	assert.Equal(t, "the number of data hashes is different with number of programs", err.Error())

	// With no programs
	init()
	programs = []*program.Program{}
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with no programs")
	assert.Equal(t, "the number of data hashes is different with number of programs", err.Error())

	// With unmatched hashes
	init()
	for i := 0; i < num; i++ {
		rand.Read(hashes[math.Intn(num)][:])
	}
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with unmathed hashes")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With disordered hashes
	init()
	common.SortProgramHashByCodeHash(hashes)
	sort.Sort(sort.Reverse(blockchain.ByHash(programs)))
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with disordered hashes")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With random no code
	init()
	for i := 0; i < num; i++ {
		programs[math.Intn(num)].Code = nil
	}
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with random no code")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With random no parameter
	init()
	for i := 0; i < num; i++ {
		index := math.Intn(num)
		programs[index].Parameter = nil
	}
	err = blockchain.RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with random no parameter")
}

func TestSchnorrPxPyToPublic(t *testing.T) {
	nA := newAccount(t)
	initialPublicKey, _ := nA.public.EncodePoint(true)
	i := hex.EncodeToString(initialPublicKey)
	Px, Py := crypto.Curve.ScalarBaseMult(nA.private)
	comparePubKey := crypto.Marshal(crypto.Curve, Px, Py)
	c := hex.EncodeToString(comparePubKey)
	if i != c {
		t.Fatalf("[TestSchnorrPxPyToPublic] public key not equals")
	}
}

func TestSchnorrPublicToPxPy(t *testing.T) {
	nA := newAccount(t)
	Px, Py := crypto.Curve.ScalarBaseMult(nA.private)
	pubKey := crypto.Marshal(crypto.Curve, Px, Py)
	c := hex.EncodeToString(pubKey)
	cBuf, _ := hex.DecodeString(c)
	cPx, cPy := crypto.Unmarshal(crypto.Curve, cBuf)

	if Px.String() != cPx.String() || Py.String() != cPy.String() {
		t.Fatalf("[TestSchnorrPublicToPxPy] px py not equals")
	}
}

func newAccount(t *testing.T) *account {
	a := new(account)
	var err error
	a.private, a.public, err = crypto.GenerateKeyPair()
	if err != nil {
		t.Errorf("Generate key pair failed, error %s", err.Error())
	}

	a.redeemScript, err = contract.CreateStandardRedeemScript(a.public)
	if err != nil {
		t.Errorf("Create standard redeem script failed, error %s", err.Error())
	}

	c, err := contract.CreateStandardContract(a.public)
	if err != nil {
		t.Errorf("Create standard contract failed, error %s", err.Error())
	}

	a.programHash = c.ToProgramHash()

	return a
}

func newSchnorrMultiAccount(num int, t *testing.T) *schnorAccount {
	ma := new(schnorAccount)

	Pxs, Pys := []*big.Int{}, []*big.Int{}
	for i := 0; i < num; i++ {
		newAccount := newAccount(t)
		ma.accounts = append(ma.accounts, newAccount)
		privKey := new(big.Int).SetBytes(newAccount.private)
		ma.privateKeys = append(ma.privateKeys, privKey)
		Px, Py := crypto.Curve.ScalarBaseMult(newAccount.private)
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}
	Px, Py := new(big.Int), new(big.Int)
	for i := 0; i < len(Pxs); i++ {
		Px, Py = crypto.Curve.Add(Px, Py, Pxs[i], Pys[i])
	}
	sumPublicKey := crypto.Marshal(crypto.Curve, Px, Py)
	publicKey, err := crypto.DecodePoint(sumPublicKey)
	ma.redeemScript, err = contract.CreateSchnorrRedeemScript(publicKey)
	if err != nil {
		t.Errorf("Create multisig redeem script failed, error %s", err.Error())
	}

	c, err := contract.CreateSchnorrContract(publicKey)
	if err != nil {
		t.Errorf("Create multi-sign contract failed, error %s", err.Error())
	}
	ma.programHash = c.ToProgramHash()
	return ma
}

func newMultiAccount(num int, t *testing.T) *multiAccount {
	ma := new(multiAccount)
	publicKeys := make([]*crypto.PublicKey, 0, num)
	for i := 0; i < num; i++ {
		ma.accounts = append(ma.accounts, newAccount(t))
		publicKeys = append(publicKeys, ma.accounts[i].public)
	}

	var err error
	ma.redeemScript, err = contract.CreateMultiSigRedeemScript(num/2+1, publicKeys)
	if err != nil {
		t.Errorf("Create multisig redeem script failed, error %s", err.Error())
	}

	c, err := contract.CreateMultiSigContract(num/2+1, publicKeys)
	if err != nil {
		t.Errorf("Create multi-sign contract failed, error %s", err.Error())
	}

	ma.programHash = c.ToProgramHash()

	return ma
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
	num := math.Intn(100) + 1
	inputs := make([]*common2.Input, 0, num)
	for i := 0; i < num; i++ {
		var txID common.Uint256
		rand.Read(txID[:])
		index := math.Intn(100)
		inputs = append(inputs, &common2.Input{
			Previous: *common2.NewOutPoint(txID, uint16(index)),
		})
	}
	return inputs
}

func randomOutputs() []*common2.Output {
	num := math.Intn(100) + 1
	outputs := make([]*common2.Output, 0, num)
	var asset common.Uint256
	rand.Read(asset[:])
	for i := 0; i < num; i++ {
		var addr common.Uint168
		rand.Read(addr[:])
		outputs = append(outputs, &common2.Output{
			AssetID:     asset,
			Value:       common.Fixed64(math.Int63()),
			OutputLock:  0,
			ProgramHash: addr,
		})
	}
	return outputs
}

func getData(tx interfaces.Transaction) []byte {
	buf := new(bytes.Buffer)
	tx.SerializeUnsigned(buf)
	return buf.Bytes()
}

func sign(private []byte, data []byte) (signature []byte, err error) {
	signature, err = crypto.Sign(private, data)
	if err != nil {
		return signature, err
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signature)))
	buf.Write(signature)
	return buf.Bytes(), err
}

func TestSortPrograms(t *testing.T) {
	// invalid program code
	getInvalidCode := func() []byte {
		var code = make([]byte, 21)
	NEXT:
		rand.Read(code)
		switch code[len(code)-1] {
		case common.STANDARD, common.MULTISIG, common.CROSSCHAIN:
			goto NEXT
		}
		return code
	}
	programs := make([]*program.Program, 0, 10)
	for i := 0; i < 2; i++ {
		p := new(program.Program)
		p.Code = getInvalidCode()
		programs = append(programs, p)
	}
	blockchain.SortPrograms(programs)

	count := 100
	hashes := make([]common.Uint168, 0, count)
	programs = make([]*program.Program, 0, count)
	for i := 0; i < count; i++ {
		p := new(program.Program)
		randType := math.Uint32()
		switch randType % 3 {
		case 0: // CHECKSIG
			p.Code = make([]byte, crypto.PublicKeyScriptLength)
			rand.Read(p.Code)
			p.Code[len(p.Code)-1] = common.STANDARD
		case 1: // MULTISIG
			num := math.Intn(5) + 3
			p.Code = make([]byte, (crypto.PublicKeyScriptLength-1)*num+3)
			rand.Read(p.Code)
			p.Code[len(p.Code)-1] = common.MULTISIG
			//case 2: // CROSSCHAIN
			//	num := math.Intn(5) + 3
			//	p.Code = make([]byte, (crypto.PublicKeyScriptLength-1)*num+3)
			//	rand.Read(p.Code)
			//	p.Code[len(p.Code)-1] = common.CROSSCHAIN
			//}
			//c := contract.Contract{}
			//hash, err := crypto.ToProgramHash(p.Code)
			//if err != nil {
			//	t.Errorf("ToProgramHash failed, %s", err)
			//}
			//hashes = append(hashes, *hash)
			//programs = append(programs, p)
		}

		common.SortProgramHashByCodeHash(hashes)
		blockchain.SortPrograms(programs)

		//fixme
		//for i, hash := range hashes {
		//	programsHash, err := crypto.ToProgramHash(programs[i].Code)
		//	if err != nil {
		//		t.Errorf("ToProgramHash failed, %s", err)
		//	}
		//	if !hash.IsEqual(*programsHash) {
		//		t.Errorf("Hash %s not match with ProgramHash %s", hex.EncodeToString(hash[:]), hex.EncodeToString(programsHash[:]))
		//	}
		//
		//	t.Logf("Hash[%02d] %s match with ProgramHash[%02d] %s", i, hex.EncodeToString(hash[:]), i, hex.EncodeToString(programsHash[:]))
		//}
	}
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
