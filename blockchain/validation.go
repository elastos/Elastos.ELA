// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"errors"

	"sort"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract"
	. "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/crypto"
)

func RunPrograms(data []byte, programHashes []common.Uint168, programs []*Program) error {
	if len(programHashes) != len(programs) {
		return errors.New("the number of data hashes is different with number of programs")
	}

	for i, program := range programs {
		programHash := programHashes[i]
		prefixType := contract.GetPrefixType(programHash)

		// TODO: this implementation will be deprecated
		if prefixType == contract.PrefixCrossChain {
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
		}

		codeHash := common.ToCodeHash(program.Code)
		ownerHash := programHash.ToCodeHash()

		if !ownerHash.IsEqual(*codeHash) {
			return errors.New("the data hashes is different with corresponding program code")
		}
		if prefixType == contract.PrefixStandard || prefixType == contract.PrefixDeposit {
			if contract.IsSchnorr(program.Code) {
				if ok, err := checkSchnorrSignatures(*program, common.Sha256D(data[:])); !ok {
					return errors.New("check schnorr signature failed:" + err.Error())
				}
			} else if contract.IsStandard(program.Code) {
				if err := CheckStandardSignature(*program, data); err != nil {
					return err
				}
			} else if contract.IsMultiSig(program.Code) {
				log.Info("mulitisign deposite")
				if err := crypto.CheckMultiSigSignatures(*program, data); err != nil {
					return err
				}
			}
		} else if prefixType == contract.PrefixMultiSig {
			if err := crypto.CheckMultiSigSignatures(*program, data); err != nil {
				return err
			}
		} else {
			return errors.New("unknown signature type")
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

	return crypto.VerifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
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
