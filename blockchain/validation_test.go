// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	math "math/rand"
	"sort"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

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
	sumpk        [33]byte
	pks          []*big.Int
}

type schnorAccount struct {
	*multiAccount
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
	var tx *types.Transaction

	tx = buildTx()
	data := getData(tx)
	act := newAccount(t)
	signature, err := act.Sign(data)
	if err != nil {
		t.Errorf("Generate signature failed, error %s", err.Error())
	}

	// Normal
	err = checkStandardSignature(program.Program{Code: act.redeemScript, Parameter: signature}, data)
	assert.NoError(t, err, "[CheckChecksigSignature] failed, %v", err)

	// invalid signature length
	var fakeSignature = make([]byte, crypto.SignatureScriptLength-math.Intn(64)-1)
	rand.Read(fakeSignature)
	err = checkStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, data)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid signature length")
	assert.Equal(t, "invalid signature length", err.Error())

	// invalid signature content
	fakeSignature = make([]byte, crypto.SignatureScriptLength)
	err = checkStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, data)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid signature content")
	assert.Equal(t, "[Validation], Verify failed.", err.Error())

	// invalid data content
	err = checkStandardSignature(program.Program{Code: act.redeemScript, Parameter: fakeSignature}, nil)
	assert.Error(t, err, "[CheckChecksigSignature] with invalid data content")
	assert.Equal(t, "[Validation], Verify failed.", err.Error())
}

func TestCheckMultiSigSignature(t *testing.T) {
	var tx *types.Transaction

	tx = buildTx()
	data := getData(tx)

	act := newMultiAccount(math.Intn(2)+3, t)
	signature, err := act.Sign(data)
	assert.NoError(t, err, "Generate signature failed, error %v", err)

	// Normal
	err = checkMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature}, data)
	assert.NoError(t, err, "[CheckMultisigSignature] failed, %v", err)

	// invalid redeem script M < 1
	fakeCode := make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[0] = fakeCode[0] - fakeCode[0] + crypto.PUSH1 - 1
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] code with M < 1 passed")
	assert.Equal(t, "invalid multi sign script code", err.Error())

	// invalid redeem script M > N
	copy(fakeCode, act.redeemScript)
	fakeCode[0] = fakeCode[len(fakeCode)-2] - crypto.PUSH1 + 2
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] code with M > N passed")
	assert.Equal(t, "invalid multi sign script code", err.Error())

	// invalid redeem script length not enough
	copy(fakeCode, act.redeemScript)
	for len(fakeCode) >= crypto.MinMultiSignCodeLength {
		fakeCode = append(fakeCode[:1], fakeCode[crypto.PublicKeyScriptLength:]...)
	}
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid length code passed")
	assert.Equal(t, "not a valid multi sign transaction code, length not enough", err.Error())

	// invalid redeem script N not equal to public keys count
	fakeCode = make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[len(fakeCode)-2] = fakeCode[len(fakeCode)-2] + 1
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid redeem script N not equal to public keys count")
	assert.Equal(t, "invalid multi sign public key script count", err.Error())

	// invalid redeem script wrong public key
	fakeCode = make([]byte, len(act.redeemScript))
	copy(fakeCode, act.redeemScript)
	fakeCode[2] = 0x01
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid redeem script wrong public key")
	assert.Equal(t, "the encodeData format is error", err.Error())

	// invalid signature length not match
	err = checkMultiSigSignatures(program.Program{Code: fakeCode, Parameter: signature[1+math.Intn(64):]}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature length not match")
	assert.Equal(t, "invalid multi sign signatures, length not match", err.Error())

	// invalid signature not enough
	cut := len(signature)/crypto.SignatureScriptLength - int(act.redeemScript[0]-crypto.PUSH1)
	err = checkMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature[65*cut:]}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature not enough")
	assert.Equal(t, "invalid signatures, not enough signatures", err.Error())

	// invalid signature too many
	err = checkMultiSigSignatures(program.Program{Code: act.redeemScript,
		Parameter: append(signature[:65], signature...)}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature too many")
	assert.Equal(t, "invalid signatures, too many signatures", err.Error())

	// invalid signature duplicate
	err = checkMultiSigSignatures(program.Program{Code: act.redeemScript,
		Parameter: append(signature[:65], signature[:len(signature)-65]...)}, data)
	assert.Error(t, err, "[CheckMultisigSignature] invalid signature duplicate")
	assert.Equal(t, "duplicated signatures", err.Error())

	// invalid signature fake signature
	signature, err = newMultiAccount(math.Intn(2)+3, t).Sign(data)
	assert.NoError(t, err, "Generate signature failed, error %v", err)
	err = checkMultiSigSignatures(program.Program{Code: act.redeemScript, Parameter: signature}, data)
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
		{
			"",
			"03DEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34",
			"4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703",
			"00000000000000000000003B78CE563F89A0ED9414F5AA28AD0D96D6795F9C6302A8DC32E64E86A333F20EF56EAC9BA30B7246D6D25E22ADB8C6BE1AEB08D49D",
			true,
			nil,
			"",
		},
		{
			"",
			"031B84C5567B126440995D3ED5AABA0565D71E1834604819FF9C17F5E9D5DD078F",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"52818579ACA59767E3291D91B76B637BEF062083284992F2D95F564CA6CB4E3530B1DA849C8E8304ADC0CFE870660334B3CFC18E825EF1DB34CFAE3DFC5D8187",
			true,
			nil,
			"test fails if jacobi symbol of x(R) instead of y(R) is used",
		},
		{
			"",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
			"570DD4CA83D4E6317B8EE6BAE83467A1BF419D0767122DE409394414B05080DCE9EE5F237CBD108EABAE1E37759AE47F8E4203DA3532EB28DB860F33D62D49BD",
			true,
			nil,
			"test fails if msg is reduced",
		},
		{
			"",
			"03EEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34",
			"4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703",
			"00000000000000000000003B78CE563F89A0ED9414F5AA28AD0D96D6795F9C6302A8DC32E64E86A333F20EF56EAC9BA30B7246D6D25E22ADB8C6BE1AEB08D49D",
			false,
			errors.New("signature verification failed"),
			"public key not on the curve",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1DFA16AEE06609280A19B67A24E1977E4697712B5FD2943914ECD5F730901B4AB7",
			false,
			errors.New("signature verification failed"),
			"incorrect R residuosity",
		},
		{
			"",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"5E2D58D8B3BCDF1ABADEC7829054F90DDA9805AAB56C77333024B9D0A508B75C",
			"00DA9B08172A9B6F0466A2DEFD817F2D7AB437E0D253CB5395A963866B3574BED092F9D860F1776A1F7412AD8A1EB50DACCC222BC8C0E26B2056DF2F273EFDEC",
			false,
			errors.New("signature verification failed"),
			"negated message hash",
		},
		{
			"",
			"0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"787A848E71043D280C50470E8E1532B2DD5D20EE912A45DBDD2BD1DFBF187EF68FCE5677CE7A623CB20011225797CE7A8DE1DC6CCD4F754A47DA6C600E59543C",
			false,
			errors.New("signature verification failed"),
			"negated s value",
		},
		{
			"",
			"03DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"negated public key",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"00000000000000000000000000000000000000000000000000000000000000009E9D01AF988B5CEDCE47221BFA9B222721F3FA408915444A4B489021DB55775F",
			false,
			errors.New("signature verification failed"),
			"sG - eP is infinite. Test fails in single verification if jacobi(y(inf)) is defined as 1 and x(inf) as 0",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"0000000000000000000000000000000000000000000000000000000000000001D37DDF0254351836D84B1BD6A795FD5D523048F298C4214D187FE4892947F728",
			false,
			errors.New("signature verification failed"),
			"sG - eP is infinite. Test fails in single verification if jacobi(y(inf)) is defined as 1 and x(inf) as 1",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"4A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"sig[0:32] is not an X coordinate on the curve",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFC2F1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("r is larger than or equal to field size"),
			"sig[0:32] is equal to field size",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1DFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
			false,
			errors.New("s is larger than or equal to curve order"),
			"sig[32:64] is equal to curve order",
		},
		{
			"",
			"6d6c66873739bc7bfb3526629670d0ea",
			"b2f0cd8ecb23c1710903f872c31b0fd37e15224af457722a87c5e0c7f50fffb3",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"public key is only 16 bytes",
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

		Px, Py := Curve.ScalarBaseMult(privKey.Bytes())
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}
	t.Run("Can sign and verify two aggregated signatures over same message", func(t *testing.T) {

		Px, Py := Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		//this is the sum pk elliptic.
		copy(pk[:], Marshal(Curve, Px, Py))

		///////////////////////////////////
		PxNew, PyNew := Unmarshal(Curve, pk[:])
		if PxNew == nil || PyNew == nil || !Curve.IsOnCurve(PxNew, PyNew) {
			fmt.Println("signature verification failed")
		}
		if Px.Cmp(PxNew) != 0 || Py.Cmp(PyNew) != 0 {
			fmt.Println("Px != PxNew || Py != PyNew")
		}
		////////////////////////////
		publicKey1, _ := crypto.DecodePoint(pk[:])
		fmt.Println(publicKey1)
		//var err error
		var tx *types.Transaction
		//var acts []act
		var hashes []common.Uint168
		var programs []*program.Program

		tx = buildTx()
		data := getData(tx)
		sig, err := AggregateSignatures(pks[:2], data[:])
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:2], m, err)
		}

		publicKey, _ := crypto.DecodePoint(pk[:])
		var pubkeys []*crypto.PublicKey
		pubkeys = append(pubkeys, publicKey)
		redeemscript, err := contract.CreateSchnorrMultiSigRedeemScript(pubkeys)
		if err != nil {
			fmt.Println("err ", err)
		}

		c, err := contract.CreateSchnorrMultiSigContract(pubkeys)
		if err != nil {
			t.Errorf("Create standard contract failed, error %s", err.Error())
		}

		programHash := c.ToProgramHash()
		hashes = append(hashes, *programHash)
		programs = append(programs, &program.Program{Code: redeemscript, Parameter: sig[:]})
		err = RunPrograms(data, hashes[0:1], programs)
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
		{
			"",
			"03DEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34",
			"4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703",
			"00000000000000000000003B78CE563F89A0ED9414F5AA28AD0D96D6795F9C6302A8DC32E64E86A333F20EF56EAC9BA30B7246D6D25E22ADB8C6BE1AEB08D49D",
			true,
			nil,
			"",
		},
		{
			"",
			"031B84C5567B126440995D3ED5AABA0565D71E1834604819FF9C17F5E9D5DD078F",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"52818579ACA59767E3291D91B76B637BEF062083284992F2D95F564CA6CB4E3530B1DA849C8E8304ADC0CFE870660334B3CFC18E825EF1DB34CFAE3DFC5D8187",
			true,
			nil,
			"test fails if jacobi symbol of x(R) instead of y(R) is used",
		},
		{
			"",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
			"570DD4CA83D4E6317B8EE6BAE83467A1BF419D0767122DE409394414B05080DCE9EE5F237CBD108EABAE1E37759AE47F8E4203DA3532EB28DB860F33D62D49BD",
			true,
			nil,
			"test fails if msg is reduced",
		},
		{
			"",
			"03EEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34",
			"4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703",
			"00000000000000000000003B78CE563F89A0ED9414F5AA28AD0D96D6795F9C6302A8DC32E64E86A333F20EF56EAC9BA30B7246D6D25E22ADB8C6BE1AEB08D49D",
			false,
			errors.New("signature verification failed"),
			"public key not on the curve",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1DFA16AEE06609280A19B67A24E1977E4697712B5FD2943914ECD5F730901B4AB7",
			false,
			errors.New("signature verification failed"),
			"incorrect R residuosity",
		},
		{
			"",
			"03FAC2114C2FBB091527EB7C64ECB11F8021CB45E8E7809D3C0938E4B8C0E5F84B",
			"5E2D58D8B3BCDF1ABADEC7829054F90DDA9805AAB56C77333024B9D0A508B75C",
			"00DA9B08172A9B6F0466A2DEFD817F2D7AB437E0D253CB5395A963866B3574BED092F9D860F1776A1F7412AD8A1EB50DACCC222BC8C0E26B2056DF2F273EFDEC",
			false,
			errors.New("signature verification failed"),
			"negated message hash",
		},
		{
			"",
			"0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"787A848E71043D280C50470E8E1532B2DD5D20EE912A45DBDD2BD1DFBF187EF68FCE5677CE7A623CB20011225797CE7A8DE1DC6CCD4F754A47DA6C600E59543C",
			false,
			errors.New("signature verification failed"),
			"negated s value",
		},
		{
			"",
			"03DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"negated public key",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"00000000000000000000000000000000000000000000000000000000000000009E9D01AF988B5CEDCE47221BFA9B222721F3FA408915444A4B489021DB55775F",
			false,
			errors.New("signature verification failed"),
			"sG - eP is infinite. Test fails in single verification if jacobi(y(inf)) is defined as 1 and x(inf) as 0",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"0000000000000000000000000000000000000000000000000000000000000001D37DDF0254351836D84B1BD6A795FD5D523048F298C4214D187FE4892947F728",
			false,
			errors.New("signature verification failed"),
			"sG - eP is infinite. Test fails in single verification if jacobi(y(inf)) is defined as 1 and x(inf) as 1",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"4A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"sig[0:32] is not an X coordinate on the curve",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFC2F1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("r is larger than or equal to field size"),
			"sig[0:32] is equal to field size",
		},
		{
			"",
			"02DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
			"243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1DFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
			false,
			errors.New("s is larger than or equal to curve order"),
			"sig[32:64] is equal to curve order",
		},
		{
			"",
			"6d6c66873739bc7bfb3526629670d0ea",
			"b2f0cd8ecb23c1710903f872c31b0fd37e15224af457722a87c5e0c7f50fffb3",
			"2A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D1E51A22CCEC35599B8F266912281F8365FFC2D035A230434A1A64DC59F7013FD",
			false,
			errors.New("signature verification failed"),
			"public key is only 16 bytes",
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

		Px, Py := Curve.ScalarBaseMult(privKey.Bytes())
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}

	t.Run("Can sign and verify two aggregated signatures over same message", func(t *testing.T) {
		sig, err := AggregateSignatures(pks[:2], m[:])
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:2], m, err)
		}

		Px, Py := Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		copy(pk[:], Marshal(Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "03c0cba209687e8f213f9b00ba0202d98ef17d189bd0d4c5decd7382b7a63bc64e"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := verify(pk, m[:], sig)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, sig, err)
		}

		// then
		if !observed {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, sig, observed, true)
		}
	})

	t.Run("Can sign and verify two more aggregated signatures over same message", func(t *testing.T) {
		sig, err := AggregateSignatures(pks[1:3], m[:])
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[1:3], m, err)
		}

		Px, Py := Curve.Add(Pxs[1], Pys[1], Pxs[2], Pys[2])
		copy(pk[:], Marshal(Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "038c47ce8f4f20fd041a25ef78e872448340b1dc28f6539d4fe0126018f1b8e0ea"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := verify(pk, m[:], sig)
		if err != nil {
			t.Fatalf("Unexpected error from Verify(%x, %x, %x): %v", pk, m, sig, err)
		}

		// then
		if !observed {
			t.Fatalf("Verify(%x, %x, %x) = %v, want %v", pk, m, sig, observed, true)
		}
	})

	t.Run("Can sign and verify three aggregated signatures over same message", func(t *testing.T) {
		sig, err := AggregateSignatures(pks[:3], m[:])
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:3], m, err)
		}

		Px, Py := Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
		Px, Py = Curve.Add(Px, Py, Pxs[2], Pys[2])
		copy(pk[:], Marshal(Curve, Px, Py))

		observedSum := hex.EncodeToString(pk[:])
		expected := "02adde346abd5b690dae4ebbb74e59ea0e41cdc00c8a5b05b0057678e5e98f9f2e"

		// then
		if observedSum != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observedSum, expected)
		}

		observed, err := verify(pk, m[:], sig)
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
		aggregatedSignature, err := AggregateSignatures(pks, m[:])
		expected := "436f519c7c3a32b794a0252d1af33c34f4a14e4e54282d004983a41b0b12270773057726f0bdd41917998a55ab1d0f5ca1ea01d50b1a6731060351e78b9a7e00"
		observed := hex.EncodeToString(aggregatedSignature[:])

		// then
		if observed != expected {
			t.Fatalf("AggregateSignatures(%x, %x) = %s, want %s", pks, m, observed, expected)
		}
		if err != nil {
			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks, m, err)
		}

		// verifying an aggregated signature
		pubKey1 := decodePublicKey("031c7ce0c8d4812b12a9a8988697f45351d83cba72ea9e16f227445599666ea415", t)
		pubKey2 := decodePublicKey("03460cfbe83d295c072c547e831ce224dd903f888a183a35d2888b7ab3a8666054", t)

		P1x, P1y := Unmarshal(Curve, pubKey1[:])
		P2x, P2y := Unmarshal(Curve, pubKey2[:])
		Px, Py := Curve.Add(P1x, P1y, P2x, P2y)

		copy(pk[:], Marshal(Curve, Px, Py))

		observed = hex.EncodeToString(pk[:])
		expected = "03486ca0cb4360d2b8c4be796772f77815932e18d3fe29ca81b5b30c41a1f3272e"

		// then
		if observed != expected {
			t.Fatalf("Sum of public keys, %s, want %s", observed, expected)
		}

		result, err := verify(pk, m[:], aggregatedSignature)
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
	var tx *types.Transaction
	var hashes []common.Uint168
	var programs []*program.Program

	tx = buildTx()
	data := getData(tx)
	num := 1
	var act *multiAccount
	init := func() {
		hashes = make([]common.Uint168, 0, num)
		programs = make([]*program.Program, 0, num)
		act = newSchnorrMultiAccount(2, t)
		hashes = append(hashes, *act.ProgramHash())
		//<<<<<<< HEAD
		//
		//		p1 := new(big.Int).SetBytes(act.accounts[0].private)
		//		p2 := new(big.Int).SetBytes(act.accounts[1].private)
		//		pks := []*big.Int{}
		//		pks = append(pks, p1)
		//		pks = append(pks, p2)
		//
		//		signature, err := AggregateSignatures(pks[:], data)
		//		if err != nil {
		//			t.Fatalf("Unexpected error from AggregateSignatures(%x, %x): %v", pks[:], data, err)
		//		}
		//		programs = append(programs, &program.Program{Code: act.RedeemScript(), Parameter: signature[:]})
		//		//}
		//=======
		sig, err := AggregateSignatures(act.pks[:2], data[:])
		if err != nil {
			fmt.Println("AggregateSignatures fail")
		}
		programs = append(programs, &program.Program{Code: act.RedeemScript(), Parameter: sig[:]})
		//>>>>>>> Schnor sign function was finished
	}
	init()
	err = RunPrograms(data, hashes[0:1], programs[0:1])
	assert.NoError(t, err, "[RunProgram] passed with 1 checksig program")
}

func TestRunPrograms(t *testing.T) {
	var err error
	var tx *types.Transaction
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
	err = RunPrograms(data, hashes[index:index+1], programs[index:index+1])
	assert.NoError(t, err, "[RunProgram] passed with 1 checksig program")

	// 1 loop multisig
	for i, act := range acts {
		switch act.(type) {
		case *multiAccount:
			index = i
			break
		}
	}
	err = RunPrograms(data, hashes[index:index+1], programs[index:index+1])
	assert.NoError(t, err, "[RunProgram] passed with 1 multisig program")

	// multiple programs
	err = RunPrograms(data, hashes, programs)
	assert.NoError(t, err, "[RunProgram] passed with multiple programs")

	// hashes count not equal to programs count
	init()
	removeIndex := math.Intn(num)
	hashes = append(hashes[:removeIndex], hashes[removeIndex+1:]...)
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with unmathed hashes")
	assert.Equal(t, "the number of data hashes is different with number of programs", err.Error())

	// With no programs
	init()
	programs = []*program.Program{}
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with no programs")
	assert.Equal(t, "the number of data hashes is different with number of programs", err.Error())

	// With unmatched hashes
	init()
	for i := 0; i < num; i++ {
		rand.Read(hashes[math.Intn(num)][:])
	}
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with unmathed hashes")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With disordered hashes
	init()
	common.SortProgramHashByCodeHash(hashes)
	sort.Sort(sort.Reverse(byHash(programs)))
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with disordered hashes")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With random no code
	init()
	for i := 0; i < num; i++ {
		programs[math.Intn(num)].Code = nil
	}
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with random no code")
	assert.Equal(t, "the data hashes is different with corresponding program code", err.Error())

	// With random no parameter
	init()
	for i := 0; i < num; i++ {
		index := math.Intn(num)
		programs[index].Parameter = nil
	}
	err = RunPrograms(data, hashes, programs)
	assert.Error(t, err, "[RunProgram] passed with random no parameter")
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

func newAccountWithIndex(index int, t *testing.T) *account {
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

func newSchnorrMultiAccount(num int, t *testing.T) *multiAccount {
	ma := new(multiAccount)
	publicKeys := make([]*crypto.PublicKey, 0, num)
	//pks := []*big.Int{}

	Pxs, Pys := []*big.Int{}, []*big.Int{}
	for i := 0; i < num; i++ {
		newAccount := newAccountWithIndex(i, t)
		ma.accounts = append(ma.accounts, newAccount)
		hexPriKey := hex.EncodeToString(newAccount.private)
		privKey := decodePrivateKey(hexPriKey, t)
		ma.pks = append(ma.pks, privKey)

		//privateStr := hex.EncodeToString(ma.accounts[i].private)
		//privKey := decodePrivateKey(privateStr, t)
		//Px, Py := Curve.ScalarBaseMult(privKey.Bytes())
		Px, Py := Curve.ScalarBaseMult(newAccount.private)
		Pxs = append(Pxs, Px)
		Pys = append(Pys, Py)
	}
	var pk [33]byte

	Px, Py := Curve.Add(Pxs[0], Pys[0], Pxs[1], Pys[1])
	copy(pk[:], Marshal(Curve, Px, Py))
	copy(ma.sumpk[:], pk[:])

	///////////////////////////////////
	PxNew, PyNew := Unmarshal(Curve, pk[:])
	if PxNew == nil || PyNew == nil || !Curve.IsOnCurve(PxNew, PyNew) {
		fmt.Println("signature verification failed")
	}
	////////////////////////////
	publicKey, _ := crypto.DecodePoint(pk[:])
	//var pubkeys []*crypto.PublicKey
	publicKeys = append(publicKeys, publicKey)
	var err error
	ma.redeemScript, err = contract.CreateSchnorrMultiSigRedeemScript(publicKeys)
	if err != nil {
		t.Errorf("Create multisig redeem script failed, error %s", err.Error())
	}

	c, err := contract.CreateSchnorrMultiSigContract(publicKeys)
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

func buildTx() *types.Transaction {
	tx := new(types.Transaction)
	tx.TxType = types.TransferAsset
	tx.Payload = new(payload.TransferAsset)
	tx.Inputs = randomInputs()
	tx.Outputs = randomOutputs()
	return tx
}

func randomInputs() []*types.Input {
	num := math.Intn(100) + 1
	inputs := make([]*types.Input, 0, num)
	for i := 0; i < num; i++ {
		var txID common.Uint256
		rand.Read(txID[:])
		index := math.Intn(100)
		inputs = append(inputs, &types.Input{
			Previous: *types.NewOutPoint(txID, uint16(index)),
		})
	}
	return inputs
}

func randomOutputs() []*types.Output {
	num := math.Intn(100) + 1
	outputs := make([]*types.Output, 0, num)
	var asset common.Uint256
	rand.Read(asset[:])
	for i := 0; i < num; i++ {
		var addr common.Uint168
		rand.Read(addr[:])
		outputs = append(outputs, &types.Output{
			AssetID:     asset,
			Value:       common.Fixed64(math.Int63()),
			OutputLock:  0,
			ProgramHash: addr,
		})
	}
	return outputs
}

func getData(tx *types.Transaction) []byte {
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
	SortPrograms(programs)

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
		SortPrograms(programs)

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
