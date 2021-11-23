// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"math/big"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type WithdrawFromSideChainTransaction struct {
	BaseTransaction
}

func (t *WithdrawFromSideChainTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.CheckTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *WithdrawFromSideChainTransaction)  CheckTransactionOutput(params *TransactionParameters) error {
	txn := params.Transaction
	blockHeight := params.BlockHeight
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if len(txn.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	specialOutputCount := 0
	for _, output := range txn.Outputs() {
		if output.AssetID != config.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}

		if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
			return err
		}

		if txn.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := checkWithdrawFromSideChainOutputPayload(output); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkWithdrawFromSideChainOutputPayload(output *common2.Output) error {
	switch output.Type {
	case common2.OTNone:
	case common2.OTWithdrawFromSideChain:
	default:
		return errors.New("transaction type dose not match the output payload type")
	}

	return output.Payload.Validate()
}

func (t *WithdrawFromSideChainTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch pld := t.Payload().(type) {
	case *payload.WithdrawFromSideChain:
		existingHashs := make(map[common.Uint256]struct{})
		for _, hash := range pld.SideChainTransactionHashes {
			if _, exist := existingHashs[hash]; exist {
				return errors.New("Duplicate sidechain tx detected in a transaction")
			}
			existingHashs[hash] = struct{}{}
		}
	}

	return errors.New("invalid payload type")
}

func (t *WithdrawFromSideChainTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return false
}

func (t *WithdrawFromSideChainTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	var err error
	if t.PayloadVersion() == payload.WithdrawFromSideChainVersion {
		err = t.checkWithdrawFromSideChainTransactionV0(params)
	} else if t.PayloadVersion() == payload.WithdrawFromSideChainVersionV1 {
		err = t.checkWithdrawFromSideChainTransactionV1(params)
	} else if t.PayloadVersion() == payload.WithdrawFromSideChainVersionV2 {
		err = t.checkWithdrawFromSideChainTransactionV2(params)
	}

	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}

func (t *WithdrawFromSideChainTransaction) checkWithdrawFromSideChainTransactionV0(params *TransactionParameters) error {
	witPayload, ok := t.Payload().(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("Invalid withdraw from side chain payload type")
	}
	for _, hash := range witPayload.SideChainTransactionHashes {
		if exist := blockchain.DefaultLedger.Store.IsSidechainTxHashDuplicate(hash); exist {
			return errors.New("Duplicate side chain transaction hash in paylod")
		}
	}

	for _, output := range t.references {
		if bytes.Compare(output.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
			return errors.New("Invalid transaction inputs address, without \"X\" at beginning")
		}
	}

	height := params.BlockHeight
	for _, p := range t.Programs() {
		publicKeys, m, n, err := crypto.ParseCrossChainScriptV1(p.Code)
		if err != nil {
			return err
		}

		if height >= params.Config.CRClaimDPOSNodeStartHeight {
			var arbiters []*state.ArbiterInfo
			var minCount uint32
			if height >= params.Config.DPOSNodeCrossChainHeight {
				arbiters = blockchain.DefaultLedger.Arbitrators.GetArbitrators()
				minCount = uint32(params.Config.GeneralArbiters) + 1
			} else {
				arbiters = blockchain.DefaultLedger.Arbitrators.GetCRCArbiters()
				minCount = params.Config.CRAgreementCount
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
		} else {
			if m < 1 || m > n || n != int(blockchain.DefaultLedger.Arbitrators.GetCrossChainArbitersCount()) ||
				m <= int(blockchain.DefaultLedger.Arbitrators.GetCrossChainArbitersMajorityCount()) {
				return errors.New("invalid multi sign script code")
			}
		}
		if err := checkCrossChainArbitrators(publicKeys); err != nil {
			return err
		}
	}

	return nil
}

func checkCrossChainArbitrators(publicKeys [][]byte) error {
	arbiters := blockchain.DefaultLedger.Arbitrators.GetCrossChainArbiters()

	arbitratorsMap := make(map[string]interface{})
	var count int
	for _, arbitrator := range arbiters {
		if !arbitrator.IsNormal {
			continue
		}
		count++

		found := false
		for _, pk := range publicKeys {
			if bytes.Equal(arbitrator.NodePublicKey, pk[1:]) {
				found = true
				break
			}
		}

		if !found {
			return errors.New("invalid cross chain arbitrators")
		}

		arbitratorsMap[common.BytesToHexString(arbitrator.NodePublicKey)] = nil
	}

	if count != len(publicKeys) || count != len(arbitratorsMap) {
		return errors.New("invalid arbitrator count")
	}

	return nil
}

func (t *WithdrawFromSideChainTransaction) checkWithdrawFromSideChainTransactionV1(params *TransactionParameters) error {
	for _, output := range t.Outputs() {
		if output.Type != common2.OTWithdrawFromSideChain {
			continue
		}
		witPayload, ok := output.Payload.(*outputpayload.Withdraw)
		if !ok {
			return errors.New("Invalid withdraw from side chain output payload type")
		}
		if exist := blockchain.DefaultLedger.Store.IsSidechainTxHashDuplicate(witPayload.SideChainTransactionHash); exist {
			return errors.New("Duplicate side chain transaction hash in output paylod")
		}
	}

	for _, output := range t.references {
		if bytes.Compare(output.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
			return errors.New("Invalid transaction inputs address, without \"X\" at beginning")
		}
	}

	height := params.BlockHeight
	for _, p := range t.Programs() {
		publicKeys, m, n, err := crypto.ParseCrossChainScriptV1(p.Code)
		if err != nil {
			return err
		}
		var arbiters []*state.ArbiterInfo
		var minCount uint32
		if height >= params.Config.DPOSNodeCrossChainHeight {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetArbitrators()
			minCount = uint32(params.Config.GeneralArbiters) + 1
		} else {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetCRCArbiters()
			minCount = params.Config.CRAgreementCount
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
	}

	return nil
}

func (t *WithdrawFromSideChainTransaction) checkWithdrawFromSideChainTransactionV2(params *TransactionParameters, ) error {
	pld, ok := t.Payload().(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("Invalid withdraw from side chain payload type")
	}

	if len(pld.Signers) < (int(params.Config.CRMemberCount)*2/3 + 1) {
		return errors.New("Signers number must be bigger than 2/3+1 CRMemberCount")
	}

	for _, output := range t.references {
		if bytes.Compare(output.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
			return errors.New("Invalid transaction inputs address, without \"X\" at beginning")
		}
	}

	err := checkSchnorrWithdrawFromSidechain(t, pld)
	if err != nil {
		return err
	}
	return nil
}

func checkSchnorrWithdrawFromSidechain(txn interfaces.Transaction, pld *payload.WithdrawFromSideChain) error {
	var pxArr []*big.Int
	var pyArr []*big.Int
	for _, index := range pld.Signers {
		arbiters := blockchain.DefaultLedger.Arbitrators.GetCrossChainArbiters()
		px, py := crypto.Unmarshal(crypto.Curve, arbiters[index].NodePublicKey)
		pxArr = append(pxArr, px)
		pyArr = append(pyArr, py)
	}
	Px, Py := crypto.Curve.Add(pxArr[0], pyArr[0], pxArr[1], pyArr[1])
	for i := 2; i < len(pxArr); i++ {
		Px, Py = crypto.Curve.Add(Px, Py, pxArr[i], pyArr[i])
	}
	var sumPublicKey []byte
	copy(sumPublicKey, crypto.Marshal(crypto.Curve, Px, Py))
	publicKey, err := crypto.DecodePoint(sumPublicKey)
	if err != nil {
		return errors.New("Invalid schnorr public key")
	}
	redeemScript, err := contract.CreateSchnorrMultiSigRedeemScript(publicKey)
	if err != nil {
		return errors.New("CreateSchnorrMultiSigRedeemScript error")
	}
	for _, program := range txn.Programs() {
		if contract.IsSchnorr(program.Code) {
			if hex.EncodeToString(program.Code) != hex.EncodeToString(redeemScript) {
				return errors.New("WithdrawFromSideChain invalid , signers can not match")
			}
		} else {
			return errors.New("Invalid schnorr program code")
		}
	}
	return nil
}
