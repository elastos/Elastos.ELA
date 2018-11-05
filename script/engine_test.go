// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/elastos/Elastos.ELA.Utility/common"

	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	"github.com/elastos/Elastos.ELA/log"
)

func Test_LockTime(t *testing.T) {
	log.Init(0, 0, 0)

	builder := NewScriptBuilder()
	builder.AddInt64(1541313751)
	builder.AddOp(OP_CHECKLOCKTIMEVERIFY)

	hashBytes := []byte{0xc9, 0x97, 0xa5, 0xe5,
		0x6e, 0x10, 0x41, 0x02,
		0xfa, 0x20, 0x9c, 0x6a,
		0x85, 0x2d, 0xd9, 0x06,
		0x60, 0xa2, 0x0b, 0x2d,
		0x9c, 0x35, 0x24, 0x23,
		0xed, 0xce, 0x25, 0x85,
		0x7f, 0xcd, 0x37, 0x04}
	hash, _ := common.Uint256FromBytes(hashBytes)
	tx := core.Transaction{
		Inputs: []*core.Input{{
			Previous: core.OutPoint{
				TxID:  *hash,
				Index: 0,
			},
			Sequence: 1541313752,
		},
		},
		Outputs: []*core.Output{{
			Value: 1000000000,
		}},
		LockTime: 1541313753,
	}

	script, _ := builder.Script()
	vm, err := NewEngine(script, &tx, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}
	err = vm.Execute()
	if err != nil {
		t.Errorf("failed to Execute script: %v", err)
	}
}

func Test_Sequence(t *testing.T) {
	log.Init(0, 0, 0)

	builder := NewScriptBuilder()
	builder.AddInt64(1541313752)
	builder.AddOp(OP_CHECKSEQUENCEVERIFY)

	hashBytes := []byte{0xc9, 0x97, 0xa5, 0xe5,
		0x6e, 0x10, 0x41, 0x02,
		0xfa, 0x20, 0x9c, 0x6a,
		0x85, 0x2d, 0xd9, 0x06,
		0x60, 0xa2, 0x0b, 0x2d,
		0x9c, 0x35, 0x24, 0x23,
		0xed, 0xce, 0x25, 0x85,
		0x7f, 0xcd, 0x37, 0x04}
	hash, _ := common.Uint256FromBytes(hashBytes)
	tx := core.Transaction{
		Inputs: []*core.Input{{
			Previous: core.OutPoint{
				TxID:  *hash,
				Index: 0,
			},
			Sequence: 1541313752,
		},
		},
		Outputs: []*core.Output{{
			Value: 1000000000,
		}},
		LockTime: 0,
	}

	script, _ := builder.Script()
	vm, err := NewEngine(script, &tx, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}
	err = vm.Execute()
	if err != nil {
		t.Errorf("failed to Execute script: %v", err)
	}
}

//[... dummy [sig ...] numsigs [pubkey ...] numpubkeys] -> [... bool]
func TestMultisig(t *testing.T) {
	log.Init(0, 0, 0)
	hashBytes := []byte{0xc9, 0x97, 0xa5, 0xe5,
		0x6e, 0x10, 0x41, 0x02,
		0xfa, 0x20, 0x9c, 0x6a,
		0x85, 0x2d, 0xd9, 0x06,
		0x60, 0xa2, 0x0b, 0x2d,
		0x9c, 0x35, 0x24, 0x23,
		0xed, 0xce, 0x25, 0x85,
		0x7f, 0xcd, 0x37, 0x04}
	hash, _ := common.Uint256FromBytes(hashBytes)
	tx := core.Transaction{
		Inputs: []*core.Input{{
			Previous: core.OutPoint{
				TxID:  *hash,
				Index: 0,
			},
			Sequence: 4294967295,
		},
		},
		Outputs: []*core.Output{{
			Value: 1000000000,
		}},
		LockTime: 0,
	}
	privateKeys := make([][]byte, 0)
	pubkeys := make([]*crypto.PublicKey, 0)
	sigDatas := make([][]byte, 0)
	data := getTxData(&tx)
	count := 10
	for i := 0; i < count; i++ {
		priKey, pubkey, err := crypto.GenerateKeyPair()

		if err != nil {
			t.Errorf("failed to GenerateKeyPair: %v", err)
		}
		privateKeys = append(privateKeys, priKey)
		pubkeys = append(pubkeys, pubkey)
		signData, err := crypto.Sign(priKey, data)
		if err != nil {
			t.Errorf("failed to Sign data: %v", err)
		}
		sigDatas = append(sigDatas, signData)
	}

	builder := NewScriptBuilder()

	for i := 0; i < count; i++ {
		builder.addData(sigDatas[i])
	}
	builder.AddOp(OP_10)

	for i := 0; i < count; i++ {
		pubBytes, _ := pubkeys[i].EncodePoint(true)
		builder.addData(pubBytes)
	}
	builder.AddOp(OP_10)
	builder.AddOp(OP_CHECKMULTISIG)

	script, _ := builder.Script()
	vm, err := NewEngine(script, &tx, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}
	err = vm.Execute()
	if err != nil {
		fmt.Println(err)
	}
}

func TestCheckSig(t *testing.T) {
	//Stack transformation: [... signature pubkey] -> [... bool]
	log.Init(0, 0, 0)
	hashBytes := []byte{0xc9, 0x97, 0xa5, 0xe5,
		0x6e, 0x10, 0x41, 0x02,
		0xfa, 0x20, 0x9c, 0x6a,
		0x85, 0x2d, 0xd9, 0x06,
		0x60, 0xa2, 0x0b, 0x2d,
		0x9c, 0x35, 0x24, 0x23,
		0xed, 0xce, 0x25, 0x85,
		0x7f, 0xcd, 0x37, 0x04}
	hash, _ := common.Uint256FromBytes(hashBytes)
	tx := core.Transaction{
		Inputs: []*core.Input{{
			Previous: core.OutPoint{
				TxID:  *hash,
				Index: 0,
			},
			Sequence: 4294967295,
		},
		},
		Outputs: []*core.Output{{
			Value: 1000000000,
		}},
		LockTime: 0,
	}

	priKey, pubkey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Errorf("failed to GenerateKeyPair: %v", err)
	}
	signData, err := crypto.Sign(priKey, getTxData(&tx))
	if err != nil {
		t.Errorf("failed to Sign data: %v", err)
	}
	builder := NewScriptBuilder()
	builder.addData(signData)
	pubBytes, _ := pubkey.EncodePoint(true)
	builder.addData(pubBytes)
	builder.AddOp(OP_CHECKSIG)
	script, _ := builder.Script()
	vm, err := NewEngine(script, &tx, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}
	err = vm.Execute()
	if err != nil {
		fmt.Println(err)
	}
}

func TestOPIF(t *testing.T) {
	log.Init(0, 0, 0)
	pkScript := mustParseShortForm("TRUE OP_IF 2 3 OP_ADD OP_ELSE 1 1 OP_ADD OP_ENDIF")
	vm, err := NewEngine(pkScript, nil, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}
	err = vm.Execute()
	if err != nil {
		t.Errorf("failed to Execute script: %v", err)
	}
}

func TestHash256(t *testing.T) {
	log.Init(0, 0, 0)
	data := []byte{1, 3, 4, 5, 6, 7, 9, 9}
	builder := NewScriptBuilder()
	builder.addData(data)
	builder.AddOp(OP_HASH256)
	script, _ := builder.Script()
	vm, err := NewEngine(script, nil, 0, 0)
	if err != nil {
		t.Errorf("failed to create script: %v", err)
	}

	done := false
	for !done {
		done, err = vm.Step()
		if err != nil {
			t.Errorf("%v", err)
		}
		var dstr, astr string
		// if we're tracing, dump the stacks.
		if vm.dstack.Depth() != 0 {
			dstr = "Stack:\n" + vm.dstack.String()
		}
		if vm.astack.Depth() != 0 {
			astr = "AltStack:\n" + vm.astack.String()
		}
		str := dstr + astr
		log.Infof("%s", str)
	}
	if err != nil {
		fmt.Println(err)
	}
	stackData := vm.GetStack()
	if len(stackData) <= 0 {
		t.Errorf("opHash256 error")
	}
	stackHash := stackData[0]
	hash := common.Sha256D(data)
	if len(stackHash) != 32 {
		t.Errorf("opHash256 length error")
	}
	for i := 0; i < len(hash); i++ {
		if hash[i] != stackHash[i] {
			t.Errorf("opHash256 error")
		}
	}
}

// parse hex string into a []byte.
func parseHex(tok string) ([]byte, error) {
	if !strings.HasPrefix(tok, "0x") {
		return nil, errors.New("not a hex number")
	}
	return hex.DecodeString(tok[2:])
}

// mustParseShortForm parses the passed short form script and returns the
// resulting bytes.  It panics if an error occurs.  This is only used in the
// tests as a helper since the only way it can fail is if there is an error in
// the test source code.
func mustParseShortForm(script string) []byte {
	s, err := parseShortForm(script)
	if err != nil {
		panic("invalid short form script in test source: err " +
			err.Error() + ", script: " + script)
	}

	return s
}

// shortFormOps holds a map of opcode names to values for use in short form
// parsing.  It is declared here so it only needs to be created once.
var shortFormOps map[string]byte

// parseShortForm parses a string as as used in the Bitcoin Core reference tests
// into the script it came from.
//
// The format used for these tests is pretty simple if ad-hoc:
//   - Opcodes other than the push opcodes and unknown are present as
//     either OP_NAME or just NAME
//   - Plain numbers are made into push operations
//   - Numbers beginning with 0x are inserted into the []byte as-is (so
//     0x14 is OP_DATA_20)
//   - Single quoted strings are pushed as data
//   - Anything else is an error
func parseShortForm(script string) ([]byte, error) {
	// Only create the short form opcode map once.
	if shortFormOps == nil {
		ops := make(map[string]byte)
		for opcodeName, opcodeValue := range OpcodeByName {
			if strings.Contains(opcodeName, "OP_UNKNOWN") {
				continue
			}
			ops[opcodeName] = opcodeValue

			// The opcodes named OP_# can't have the OP_ prefix
			// stripped or they would conflict with the plain
			// numbers.  Also, since OP_FALSE and OP_TRUE are
			// aliases for the OP_0, and OP_1, respectively, they
			// have the same value, so detect those by name and
			// allow them.
			if (opcodeName == "OP_FALSE" || opcodeName == "OP_TRUE") ||
				(opcodeValue != OP_0 && (opcodeValue < OP_1 ||
					opcodeValue > OP_16)) {

				ops[strings.TrimPrefix(opcodeName, "OP_")] = opcodeValue
			}
		}
		shortFormOps = ops
	}

	// Split only does one separator so convert all \n and tab into  space.
	script = strings.Replace(script, "\n", " ", -1)
	script = strings.Replace(script, "\t", " ", -1)
	tokens := strings.Split(script, " ")
	builder := NewScriptBuilder()

	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
		// if parses as a plain number
		if num, err := strconv.ParseInt(tok, 10, 64); err == nil {
			builder.AddInt64(num)
			continue
		} else if bts, err := parseHex(tok); err == nil {
			// Concatenate the bytes manually since the test code
			// intentionally creates scripts that are too large and
			// would cause the builder to error otherwise.
			if builder.err == nil {
				builder.script = append(builder.script, bts...)
			}
		} else if len(tok) >= 2 && tok[0] == '\'' && tok[len(tok)-1] == '\'' {
			builder.AddFullData([]byte(tok[1:len(tok)-1]))
		} else if opcode, ok := shortFormOps[tok]; ok {
			builder.AddOp(opcode)
		} else {
			return nil, fmt.Errorf("bad token %q", tok)
		}

	}
	return builder.Script()
}
