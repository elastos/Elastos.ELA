// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"crypto/sha256"
	"errors"
	"github.com/elastos/Elastos.ELA/vm"
	interfaces2 "github.com/elastos/Elastos.ELA/vm/interfaces"

	"sort"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	. "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/crypto"
)

var GetDataContainer = func(programHash *common.Uint168, tx interfaces.Transaction) interfaces2.IDataContainer {
	return tx
}

func RunPrograms(tx interfaces.Transaction, data []byte, hashes []common.Uint168, programs []*Program) error {
	if tx == nil {
		return errors.New("invalid data content nil transaction")
	}
	if len(hashes) != len(programs) {
		return errors.New("number of data hashes is different with number of programs")
	}

	for i := 0; i < len(programs); i++ {
		program := programs[i]
		programHash := hashes[i]

		codeHash := common.ToCodeHash(program.Code)

		prefixType := contract.GetPrefixType(programHash)


		switch prefixType {
		case contract.PrefixCrossChain:
			if contract.IsSchnorr(program.Code) {
				if ok, err := checkSchnorrSignatures(*program, common.Sha256D(data[:])); !ok {
					return errors.New("check schnorr signature failed:" + err.Error())
				}
			} else {
				if err := checkCrossChainSignatures(*program, data); err != nil {
					return err
				}
			}
			continue

		case contract.PrefixStandard, contract.PrefixDeposit:
			if !hashes[i].ToCodeHash().IsEqual(*codeHash) {
				return errors.New("data hash is different from corresponding program code")
			}

			if contract.IsSchnorr(program.Code) {
				if ok, err := checkSchnorrSignatures(*program, common.Sha256D(data[:])); !ok {
					return errors.New("check schnorr signature failed:" + err.Error())
				}
				continue
			}

		case contract.PrefixMultiSig:
			if !hashes[i].ToCodeHash().IsEqual(*codeHash) {
				return errors.New("data hash is different from corresponding program code")
			}
			//if err := CheckMultiSigSignatures(*program, data); err != nil {
			//	return err
			//}

		default:
			return errors.New("unknown signature type")
		}

		// check standard or multi signature
		// execute program on VM
		se := vm.NewExecutionEngine(GetDataContainer(&hashes[i], tx),
			new(vm.CryptoECDsa), vm.MAXSTEPS, nil, nil)
		se.LoadScript(programs[i].Code, false)
		se.LoadScript(programs[i].Parameter, true)
		se.Execute()

		if se.GetState() != vm.HALT {
			return errors.New("[VM] Finish State not equal to HALT")
		}

		if se.GetEvaluationStack().Count() != 1 {
			return errors.New("[VM] Execute Engine Stack Count Error")
		}

		success := se.GetExecuteResult()
		if !success {
			return errors.New("[VM] Check Sig FALSE")
		}
	}

	return nil
}

func GetTxProgramHashes(tx interfaces.Transaction, references map[*common2.Input]common2.Output) ([]common.Uint168, error) {
	if tx == nil {
		return nil, errors.New("[BaseTransaction],GetProgramHashes transaction is nil")
	}
	hashes := make([]common.Uint168, 0)
	uniqueHashes := make([]common.Uint168, 0)
	// add inputUTXO's transaction
	for _, output := range references {
		programHash := output.ProgramHash
		hashes = append(hashes, programHash)
	}
	for _, attribute := range tx.Attributes() {
		if attribute.Usage == common2.Script {
			dataHash, err := common.Uint168FromBytes(attribute.Data)
			if err != nil {
				return nil, errors.New("[BaseTransaction], GetProgramHashes err")
			}
			hashes = append(hashes, *dataHash)
		}
	}

	//remove duplicated hashes
	unique := make(map[common.Uint168]bool)
	for _, v := range hashes {
		unique[v] = true
	}
	for k := range unique {
		uniqueHashes = append(uniqueHashes, k)
	}
	return uniqueHashes, nil
}

func CheckStandardSignature(program Program, data []byte) error {
	if len(program.Parameter) != crypto.SignatureScriptLength {
		return errors.New("invalid signature length")
	}

	publicKey, err := crypto.DecodePoint(program.Code[1 : len(program.Code)-1])
	if err != nil {
		return err
	}

	return crypto.Verify(*publicKey, data, program.Parameter[1:])
}

func CheckMultiSigSignatures(program Program, data []byte) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1
	if m < 1 || m > n {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	return verifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
}

func checkSchnorrSignatures(program Program, data [32]byte) (bool, error) {
	publicKey := [33]byte{}
	copy(publicKey[:], program.Code[2:])

	signature := [64]byte{}
	copy(signature[:], program.Parameter[:64])

	return crypto.SchnorrVerify(publicKey, data, signature)
}

func checkCrossChainSignatures(program Program, data []byte) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1
	publicKeys, err := crypto.ParseCrossChainScript(code)
	if err != nil {
		return err
	}

	return verifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
}

func verifyMultisigSignatures(m, n int, publicKeys [][]byte, signatures, data []byte) error {
	if len(publicKeys) != n {
		return errors.New("invalid multi sign public key script count")
	}
	if len(signatures)%crypto.SignatureScriptLength != 0 {
		return errors.New("invalid multi sign signatures, length not match")
	}
	if len(signatures)/crypto.SignatureScriptLength < m {
		return errors.New("invalid signatures, not enough signatures")
	}
	if len(signatures)/crypto.SignatureScriptLength > n {
		return errors.New("invalid signatures, too many signatures")
	}

	var verified = make(map[common.Uint256]struct{})
	for i := 0; i < len(signatures); i += crypto.SignatureScriptLength {
		// Remove length byte
		sign := signatures[i : i+crypto.SignatureScriptLength][1:]
		// Match public key with signature
		for _, publicKey := range publicKeys {
			pubKey, err := crypto.DecodePoint(publicKey[1:])
			if err != nil {
				return err
			}
			err = crypto.Verify(*pubKey, data, sign)
			if err == nil {
				hash := sha256.Sum256(publicKey)
				if _, ok := verified[hash]; ok {
					return errors.New("duplicated signatures")
				}
				verified[hash] = struct{}{}
				break // back to public keys loop
			}
		}
	}
	// Check signatures count
	if len(verified) < m {
		return errors.New("matched signatures not enough")
	}

	return nil
}

func SortPrograms(programs []*Program) {
	sort.Sort(ByHash(programs))
}

type ByHash []*Program

func (p ByHash) Len() int      { return len(p) }
func (p ByHash) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByHash) Less(i, j int) bool {
	hashi := common.ToCodeHash(p[i].Code)
	hashj := common.ToCodeHash(p[j].Code)
	return hashi.Compare(*hashj) < 0
}
