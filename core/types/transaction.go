// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package types

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
)

const (
	InvalidTransactionSize = -1
)

type Transaction struct {
	Version        common2.TransactionVersion // New field added in TxVersion09
	TxType         common2.TxType
	PayloadVersion byte
	Payload        Payload
	Attributes     []*common2.Attribute
	Inputs         []*common2.Input
	Outputs        []*common2.Output
	LockTime       uint32
	Programs       []*pg.Program
	Fee            common.Fixed64
	FeePerKB       common.Fixed64

	txHash *common.Uint256
}

func (tx *Transaction) String() string {
	return fmt.Sprint("Transaction: {\n\t",
		"Hash: ", tx.hash().String(), "\n\t",
		"Version: ", tx.Version, "\n\t",
		"TxType: ", tx.TxType.Name(), "\n\t",
		"PayloadVersion: ", tx.PayloadVersion, "\n\t",
		"Payload: ", common.BytesToHexString(tx.Payload.Data(tx.PayloadVersion)), "\n\t",
		"Attributes: ", tx.Attributes, "\n\t",
		"Inputs: ", tx.Inputs, "\n\t",
		"Outputs: ", tx.Outputs, "\n\t",
		"LockTime: ", tx.LockTime, "\n\t",
		"Programs: ", tx.Programs, "\n\t",
		"}\n")
}

// Serialize the Transaction
func (tx *Transaction) Serialize(w io.Writer) error {
	if err := tx.SerializeUnsigned(w); err != nil {
		return errors.New("Transaction txSerializeUnsigned Serialize failed, " + err.Error())
	}
	//Serialize  Transaction's programs
	if err := common.WriteVarUint(w, uint64(len(tx.Programs))); err != nil {
		return errors.New("Transaction program count failed.")
	}
	for _, program := range tx.Programs {
		if err := program.Serialize(w); err != nil {
			return errors.New("Transaction Programs Serialize failed, " + err.Error())
		}
	}
	return nil
}

// Serialize the Transaction data without contracts
func (tx *Transaction) SerializeUnsigned(w io.Writer) error {
	// Version
	if tx.Version >= common2.TxVersion09 {
		if _, err := w.Write([]byte{byte(tx.Version)}); err != nil {
			return err
		}
	}
	// TxType
	if _, err := w.Write([]byte{byte(tx.TxType)}); err != nil {
		return err
	}
	// PayloadVersion
	if _, err := w.Write([]byte{tx.PayloadVersion}); err != nil {
		return err
	}
	// Payload
	if tx.Payload == nil {
		return errors.New("Transaction Payload is nil.")
	}
	if err := tx.Payload.Serialize(w, tx.PayloadVersion); err != nil {
		return err
	}

	//[]*txAttribute
	if err := common.WriteVarUint(w, uint64(len(tx.Attributes))); err != nil {
		return errors.New("Transaction item txAttribute length serialization failed.")
	}
	for _, attr := range tx.Attributes {
		if err := attr.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Inputs
	if err := common.WriteVarUint(w, uint64(len(tx.Inputs))); err != nil {
		return errors.New("Transaction item Inputs length serialization failed.")
	}
	for _, utxo := range tx.Inputs {
		if err := utxo.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Outputs
	if err := common.WriteVarUint(w, uint64(len(tx.Outputs))); err != nil {
		return errors.New("Transaction item Outputs length serialization failed.")
	}
	for _, output := range tx.Outputs {
		if err := output.Serialize(w, tx.Version); err != nil {
			return err
		}
	}

	return common.WriteUint32(w, tx.LockTime)
}

// Deserialize the Transaction
func (tx *Transaction) Deserialize(r io.Reader) error {
	// tx deserialize
	if err := tx.DeserializeUnsigned(r); err != nil {
		return errors.New("transaction Deserialize error: " + err.Error())
	}

	// tx program
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("transaction write program count error: " + err.Error())
	}
	for i := uint64(0); i < count; i++ {
		var program pg.Program
		if err := program.Deserialize(r); err != nil {
			return errors.New("transaction deserialize program error: " + err.Error())
		}
		tx.Programs = append(tx.Programs, &program)
	}
	return nil
}

func (tx *Transaction) DeserializeUnsigned(r io.Reader) error {
	flagByte, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}

	if common2.TransactionVersion(flagByte[0]) >= common2.TxVersion09 {
		tx.Version = common2.TransactionVersion(flagByte[0])
		txType, err := common.ReadBytes(r, 1)
		if err != nil {
			return err
		}
		tx.TxType = common2.TxType(txType[0])
	} else {
		tx.Version = common2.TxVersionDefault
		tx.TxType = common2.TxType(flagByte[0])
	}

	payloadVersion, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	tx.PayloadVersion = payloadVersion[0]

	tx.Payload, err = GetPayload(tx.TxType)
	if err != nil {
		return err
	}

	err = tx.Payload.Deserialize(r, tx.PayloadVersion)
	if err != nil {
		return errors.New("deserialize Payload failed: " + err.Error())
	}
	// attributes
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var attr common2.Attribute
		if err := attr.Deserialize(r); err != nil {
			return err
		}
		tx.Attributes = append(tx.Attributes, &attr)
	}
	// inputs
	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var input common2.Input
		if err := input.Deserialize(r); err != nil {
			return err
		}
		tx.Inputs = append(tx.Inputs, &input)
	}
	// outputs
	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var output common2.Output
		if err := output.Deserialize(r, tx.Version); err != nil {
			return err
		}
		tx.Outputs = append(tx.Outputs, &output)
	}

	tx.LockTime, err = common.ReadUint32(r)
	if err != nil {
		return err
	}

	return nil
}

func (tx *Transaction) GetSize() int {
	buf := new(bytes.Buffer)
	if err := tx.Serialize(buf); err != nil {
		return InvalidTransactionSize
	}
	return buf.Len()
}

func (tx *Transaction) hash() common.Uint256 {
	buf := new(bytes.Buffer)
	tx.SerializeUnsigned(buf)
	return common.Hash(buf.Bytes())
}

func (tx *Transaction) Hash() common.Uint256 {
	if tx.txHash == nil {
		txHash := tx.hash()
		tx.txHash = &txHash
	}
	return *tx.txHash
}

func (tx *Transaction) IsReturnSideChainDepositCoinTx() bool {
	return tx.TxType == common2.ReturnSideChainDepositCoin
}

func (tx *Transaction) ISCRCouncilMemberClaimNode() bool {
	return tx.TxType == common2.CRCouncilMemberClaimNode
}

func (tx *Transaction) IsCRAssetsRectifyTx() bool {
	return tx.TxType == common2.CRAssetsRectify
}

func (tx *Transaction) IsCRCAppropriationTx() bool {
	return tx.TxType == common2.CRCAppropriation
}

func (tx *Transaction) IsNextTurnDPOSInfoTx() bool {
	return tx.TxType == common2.NextTurnDPOSInfo
}

func (tx *Transaction) IsCustomIDResultTx() bool {
	return tx.TxType == common2.ProposalResult
}

func (tx *Transaction) IsCustomIDRelatedTx() bool {
	if tx.IsCRCProposalTx() {
		p, _ := tx.Payload.(*payload.CRCProposal)
		return p.ProposalType == payload.ReserveCustomID ||
			p.ProposalType == payload.ReceiveCustomID ||
			p.ProposalType == payload.ChangeCustomIDFee
	}
	if tx.IsCustomIDResultTx() {
		return true
	}
	return false
}

func (tx *Transaction) IsSideChainUpgradeTx() bool {
	if tx.IsCRCProposalTx() {
		p, _ := tx.Payload.(*payload.CRCProposal)
		return p.ProposalType > payload.MinUpgradeProposalType &&
			p.ProposalType <= payload.MaxUpgradeProposalType
	}
	return false
}

func (tx *Transaction) IsCRCProposalRealWithdrawTx() bool {
	return tx.TxType == common2.CRCProposalRealWithdraw
}

func (tx *Transaction) IsUpdateCRTx() bool {
	return tx.TxType == common2.UpdateCR
}

func (tx *Transaction) IsCRCProposalWithdrawTx() bool {
	return tx.TxType == common2.CRCProposalWithdraw
}

func (tx *Transaction) IsCRCProposalReviewTx() bool {
	return tx.TxType == common2.CRCProposalReview
}

func (tx *Transaction) IsCRCProposalTrackingTx() bool {
	return tx.TxType == common2.CRCProposalTracking
}

func (tx *Transaction) IsCRCProposalTx() bool {
	return tx.TxType == common2.CRCProposal
}

func (tx *Transaction) IsReturnCRDepositCoinTx() bool {
	return tx.TxType == common2.ReturnCRDepositCoin
}

func (tx *Transaction) IsUnregisterCRTx() bool {
	return tx.TxType == common2.UnregisterCR
}

func (tx *Transaction) IsRegisterCRTx() bool {
	return tx.TxType == common2.RegisterCR
}

func (tx *Transaction) IsIllegalTypeTx() bool {
	return tx.IsIllegalProposalTx() || tx.IsIllegalVoteTx() || tx.IsIllegalBlockTx() || tx.IsSidechainIllegalDataTx()
}

//special tx is this kind of tx who have no input and output
func (tx *Transaction) IsSpecialTx() bool {
	if tx.IsIllegalTypeTx() || tx.IsInactiveArbitrators() || tx.IsNextTurnDPOSInfoTx() {
		return true
	}
	return false
}

func (tx *Transaction) GetSpecialTxHash() (common.Uint256, error) {
	switch tx.TxType {
	case common2.IllegalProposalEvidence, common2.IllegalVoteEvidence,
		common2.IllegalBlockEvidence, common2.IllegalSidechainEvidence, common2.InactiveArbitrators:
		illegalData, ok := tx.Payload.(payload.DPOSIllegalData)
		if !ok {
			return common.Uint256{}, errors.New("special tx payload cast failed")
		}
		return illegalData.Hash(), nil
	case common2.NextTurnDPOSInfo:
		payloadData, ok := tx.Payload.(*payload.NextTurnDPOSInfo)
		if !ok {
			return common.Uint256{}, errors.New("NextTurnDPOSInfo tx payload cast failed")
		}
		return payloadData.Hash(), nil
	}
	return common.Uint256{}, errors.New("wrong TxType not special tx")
}

func (tx *Transaction) IsIllegalProposalTx() bool {
	return tx.TxType == common2.IllegalProposalEvidence
}

func (tx *Transaction) IsIllegalVoteTx() bool {
	return tx.TxType == common2.IllegalVoteEvidence
}

func (tx *Transaction) IsIllegalBlockTx() bool {
	return tx.TxType == common2.IllegalBlockEvidence
}

func (tx *Transaction) IsSidechainIllegalDataTx() bool {
	return tx.TxType == common2.IllegalSidechainEvidence
}

func (tx *Transaction) IsInactiveArbitrators() bool {
	return tx.TxType == common2.InactiveArbitrators
}

func (tx *Transaction) IsRevertToPOW() bool {
	return tx.TxType == common2.RevertToPOW
}

func (tx *Transaction) IsRevertToDPOS() bool {
	return tx.TxType == common2.RevertToDPOS
}

func (tx *Transaction) IsUpdateVersion() bool {
	return tx.TxType == common2.UpdateVersion
}

func (tx *Transaction) IsProducerRelatedTx() bool {
	return tx.TxType == common2.RegisterProducer || tx.TxType == common2.UpdateProducer ||
		tx.TxType == common2.ActivateProducer || tx.TxType == common2.CancelProducer
}

func (tx *Transaction) IsUpdateProducerTx() bool {
	return tx.TxType == common2.UpdateProducer
}

func (tx *Transaction) IsReturnDepositCoin() bool {
	return tx.TxType == common2.ReturnDepositCoin
}

func (tx *Transaction) IsCancelProducerTx() bool {
	return tx.TxType == common2.CancelProducer
}

func (tx *Transaction) IsActivateProducerTx() bool {
	return tx.TxType == common2.ActivateProducer
}

func (tx *Transaction) IsRegisterProducerTx() bool {
	return tx.TxType == common2.RegisterProducer
}

func (tx *Transaction) IsSideChainPowTx() bool {
	return tx.TxType == common2.SideChainPow
}

func (tx *Transaction) IsNewSideChainPowTx() bool {
	if !tx.IsSideChainPowTx() || len(tx.Inputs) != 0 {
		return false
	}

	return true
}

func (tx *Transaction) IsTransferCrossChainAssetTx() bool {
	return tx.TxType == common2.TransferCrossChainAsset
}

func (tx *Transaction) IsWithdrawFromSideChainTx() bool {
	return tx.TxType == common2.WithdrawFromSideChain
}

func (tx *Transaction) IsRechargeToSideChainTx() bool {
	return tx.TxType == common2.RechargeToSideChain
}

func (tx *Transaction) IsCoinBaseTx() bool {
	return tx.TxType == common2.CoinBase
}

// SerializeSizeStripped returns the number of bytes it would take to serialize
// the block, excluding any witness data (if any).
func (tx *Transaction) SerializeSizeStripped() int {
	// todo add cache for size according to btcd
	return tx.GetSize()
}

// Payload define the func for loading the payload data
// base on payload type which have different structure
type Payload interface {
	// Get payload data
	Data(version byte) []byte

	Serialize(w io.Writer, version byte) error

	Deserialize(r io.Reader, version byte) error

	payload.PayloadChecker
}

func GetPayload(txType common2.TxType) (Payload, error) {
	var p Payload
	switch txType {
	case common2.CoinBase:
		p = new(payload.CoinBase)
	case common2.RegisterAsset:
		p = new(payload.RegisterAsset)
	case common2.TransferAsset:
		p = new(payload.TransferAsset)
	case common2.Record:
		p = new(payload.Record)
	case common2.SideChainPow:
		p = new(payload.SideChainPow)
	case common2.WithdrawFromSideChain:
		p = new(payload.WithdrawFromSideChain)
	case common2.TransferCrossChainAsset:
		p = new(payload.TransferCrossChainAsset)
	case common2.RegisterProducer:
		p = new(payload.ProducerInfo)
	case common2.CancelProducer:
		p = new(payload.ProcessProducer)
	case common2.UpdateProducer:
		p = new(payload.ProducerInfo)
	case common2.ReturnDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common2.ActivateProducer:
		p = new(payload.ActivateProducer)
	case common2.IllegalProposalEvidence:
		p = new(payload.DPOSIllegalProposals)
	case common2.IllegalVoteEvidence:
		p = new(payload.DPOSIllegalVotes)
	case common2.IllegalBlockEvidence:
		p = new(payload.DPOSIllegalBlocks)
	case common2.IllegalSidechainEvidence:
		p = new(payload.SidechainIllegalData)
	case common2.InactiveArbitrators:
		p = new(payload.InactiveArbitrators)
	case common2.RevertToDPOS:
		p = new(payload.RevertToDPOS)
	case common2.UpdateVersion:
		p = new(payload.UpdateVersion)
	case common2.RegisterCR:
		p = new(payload.CRInfo)
	case common2.UpdateCR:
		p = new(payload.CRInfo)
	case common2.UnregisterCR:
		p = new(payload.UnregisterCR)
	case common2.ReturnCRDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common2.CRCProposal:
		p = new(payload.CRCProposal)
	case common2.CRCProposalReview:
		p = new(payload.CRCProposalReview)
	case common2.CRCProposalWithdraw:
		p = new(payload.CRCProposalWithdraw)
	case common2.CRCProposalTracking:
		p = new(payload.CRCProposalTracking)
	case common2.CRCAppropriation:
		p = new(payload.CRCAppropriation)
	case common2.CRAssetsRectify:
		p = new(payload.CRAssetsRectify)
	case common2.CRCProposalRealWithdraw:
		p = new(payload.CRCProposalRealWithdraw)
	case common2.CRCouncilMemberClaimNode:
		p = new(payload.CRCouncilMemberClaimNode)
	case common2.NextTurnDPOSInfo:
		p = new(payload.NextTurnDPOSInfo)
	case common2.RevertToPOW:
		p = new(payload.RevertToPOW)
	case common2.ProposalResult:
		p = new(payload.RecordProposalResult)
	case common2.ReturnSideChainDepositCoin:
		p = new(payload.ReturnSideChainDepositCoin)
	default:
		return nil, errors.New("[Transaction], invalid transaction type.")
	}
	return p, nil
}

func (tx *Transaction) IsSmallTransfer(min common.Fixed64) bool {
	var totalCrossAmt common.Fixed64
	if tx.PayloadVersion == payload.TransferCrossChainVersion {
		payloadObj, ok := tx.Payload.(*payload.TransferCrossChainAsset)
		if !ok {
			return false
		}
		for i := 0; i < len(payloadObj.CrossChainAddresses); i++ {
			if bytes.Compare(tx.Outputs[payloadObj.OutputIndexes[i]].ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += tx.Outputs[payloadObj.OutputIndexes[i]].Value
			}
		}
	} else {
		for _, o := range tx.Outputs {
			if bytes.Compare(o.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += o.Value
			}
		}
	}

	return totalCrossAmt <= min
}
