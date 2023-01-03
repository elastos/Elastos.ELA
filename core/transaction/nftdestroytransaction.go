// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type NFTDestroyTransactionFromSideChain struct {
	BaseTransaction
}

func (t *NFTDestroyTransactionFromSideChain) CheckTransactionInput() error {
	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *NFTDestroyTransactionFromSideChain) CheckTransactionOutput() error {
	if len(t.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}
	return nil
}

func (t *NFTDestroyTransactionFromSideChain) CheckAttributeProgram() error {
	if len(t.Programs()) != 1 || len(t.Attributes()) != 1 {
		return errors.New("zero cost tx should have one programs and one attributes")
	}
	return nil
}

func (t *NFTDestroyTransactionFromSideChain) CheckTransactionPayload() error {
	_, ok := t.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return errors.New("Invalid NFTDestroyFromSideChain payload type")
	}

	return nil
}

func (t *NFTDestroyTransactionFromSideChain) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config
	if blockHeight < chainParams.DPoSConfiguration.NFTStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before NFTStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *NFTDestroyTransactionFromSideChain) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *NFTDestroyTransactionFromSideChain) SpecialContextCheck() (elaerr.ELAError, bool) {
	nftDestroyPayload, ok := t.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	_, err := common.Uint168FromAddress(nftDestroyPayload.GenesisBlockAddress)
	// check genesis block when sidechain registered in the future
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New(" invalid GenesisBlockAddress")), true
	}

	state := t.parameters.BlockChain.GetState()

	canDestroyIDs := state.CanNFTDestroy(nftDestroyPayload.IDs)
	if len(canDestroyIDs) != len(nftDestroyPayload.IDs) {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New(" NFT can not destroy")), true
	}

	err = t.checkNFTDestroyTransactionFromSideChain()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	return nil, true
}

func (t *NFTDestroyTransactionFromSideChain) checkNFTDestroyTransactionFromSideChain() error {
	buf := new(bytes.Buffer)
	t.SerializeUnsigned(buf)
	height := t.parameters.BlockHeight
	for _, p := range t.Programs() {
		publicKeys, m, n, err := crypto.ParseCrossChainScriptV1(p.Code)
		if err != nil {
			return err
		}
		var arbiters []*state.ArbiterInfo
		var minCount uint32
		if height >= t.parameters.Config.DPoSConfiguration.DPOSNodeCrossChainHeight {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetArbitrators()
			minCount = uint32(t.parameters.Config.DPoSConfiguration.NormalArbitratorsCount) + 1
		} else {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetCRCArbiters()
			minCount = t.parameters.Config.CRConfiguration.CRAgreementCount
		}
		var arbitersCount int
		for _, c := range arbiters {
			if !c.IsNormal {
				continue
			}
			arbitersCount++
		}
		if n != arbitersCount {
			return errors.New("invalid arbiters total count in code")
		}
		if m < int(minCount) {
			return errors.New("invalid arbiters sign count in code")
		}
		if err := checkCrossChainArbitrators(publicKeys); err != nil {
			return err
		}
		if err := checkCrossChainSignatures(*p, buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func checkCrossChainSignatures(program program.Program, data []byte) error {
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
