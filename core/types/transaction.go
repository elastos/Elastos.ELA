package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	. "github.com/elastos/Elastos.ELA/core/types/payload"
)

const (
	InvalidTransactionSize = -1
)

// TxType represents different transaction types with different payload format.
// The TxType range is 0x00 - 0x08. When it is greater than 0x08 it will be
// interpreted as a TransactionVersion.
type TxType byte

const (
	CoinBase                TxType = 0x00
	RegisterAsset           TxType = 0x01
	TransferAsset           TxType = 0x02
	Record                  TxType = 0x03
	Deploy                  TxType = 0x04
	SideChainPow            TxType = 0x05
	RechargeToSideChain     TxType = 0x06
	WithdrawFromSideChain   TxType = 0x07
	TransferCrossChainAsset TxType = 0x08

	RegisterProducer TxType = 0x09
	CancelProducer   TxType = 0x0a
	UpdateProducer   TxType = 0x0b
	ReturnDepositCoin TxType = 0x0c

	IllegalProposalEvidence TxType = 0x0d
	IllegalVoteEvidence     TxType = 0x0e
	IllegalBlockEvidence    TxType = 0x0f
)

func (self TxType) Name() string {
	switch self {
	case CoinBase:
		return "CoinBase"
	case RegisterAsset:
		return "RegisterAsset"
	case TransferAsset:
		return "TransferAsset"
	case Record:
		return "Record"
	case Deploy:
		return "Deploy"
	case SideChainPow:
		return "SideChainPow"
	case RechargeToSideChain:
		return "RechargeToSideChain"
	case WithdrawFromSideChain:
		return "WithdrawFromSideChain"
	case TransferCrossChainAsset:
		return "TransferCrossChainAsset"
	case RegisterProducer:
		return "RegisterProducer"
	case CancelProducer:
		return "CancelProducer"
	case UpdateProducer:
		return "UpdateProducer"
	case ReturnDepositCoin:
		return "ReturnDepositCoin"
	case IllegalProposalEvidence:
		return "IllegalProposalEvidence"
	case IllegalVoteEvidence:
		return "IllegalVoteEvidence"
	case IllegalBlockEvidence:
		return "IllegalBlockEvidence"
	default:
		return "Unknown"
	}
}

type TransactionVersion byte

const (
	TxVersionDefault TransactionVersion = 0x00
	TxVersion09      TransactionVersion = 0x09
)

type Transaction struct {
	Version        TransactionVersion // New field added in TxVersionC0
	TxType         TxType
	PayloadVersion byte
	Payload        Payload
	Attributes     []*Attribute
	Inputs         []*Input
	Outputs        []*Output
	LockTime       uint32
	Programs       []*pg.Program
	Fee            common.Fixed64
	FeePerKB       common.Fixed64

	hash *common.Uint256
}

func (tx *Transaction) String() string {
	hash := tx.Hash()
	return fmt.Sprint("Transaction: {\n\t",
		"Hash: ", hash.String(), "\n\t",
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
	if tx.Version >= TxVersion09 {
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

	if TransactionVersion(flagByte[0]) >= TxVersion09 {
		tx.Version = TransactionVersion(flagByte[0])
		txType, err := common.ReadBytes(r, 1)
		if err != nil {
			return err
		}
		tx.TxType = TxType(txType[0])
	} else {
		tx.Version = TxVersionDefault
		tx.TxType = TxType(flagByte[0])
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
		return errors.New("deserialize Payload failed")
	}
	// attributes
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var attr Attribute
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
		var input Input
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
		var output Output
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

func (tx *Transaction) Hash() common.Uint256 {
	if tx.hash == nil {
		buf := new(bytes.Buffer)
		tx.SerializeUnsigned(buf)
		hash := common.Uint256(common.Sha256D(buf.Bytes()))
		tx.hash = &hash
	}
	return *tx.hash
}

func (tx *Transaction) IsIllegalProposalTx() bool {
	return tx.TxType == IllegalProposalEvidence
}

func (tx *Transaction) IsIllegalVoteTx() bool {
	return tx.TxType == IllegalVoteEvidence
}

func (tx *Transaction) IsIllegalBlockTx() bool {
	return tx.TxType == IllegalBlockEvidence
}

func (tx *Transaction) IsUpdateProducerTx() bool {
	return tx.TxType == UpdateProducer
}

func (tx *Transaction) IsReturnDepositCoin() bool {
	return tx.TxType == ReturnDepositCoin
}

func (tx *Transaction) IsCancelProducerTx() bool {
	return tx.TxType == CancelProducer
}

func (tx *Transaction) IsRegisterProducerTx() bool {
	return tx.TxType == RegisterProducer
}

func (tx *Transaction) IsSideChainPowTx() bool {
	return tx.TxType == SideChainPow
}

func (tx *Transaction) IsTransferCrossChainAssetTx() bool {
	return tx.TxType == TransferCrossChainAsset
}

func (tx *Transaction) IsWithdrawFromSideChainTx() bool {
	return tx.TxType == WithdrawFromSideChain
}

func (tx *Transaction) IsRechargeToSideChainTx() bool {
	return tx.TxType == RechargeToSideChain
}

func (tx *Transaction) IsCoinBaseTx() bool {
	return tx.TxType == CoinBase
}

func NewTrimmedTx(hash common.Uint256) *Transaction {
	tx := new(Transaction)
	tx.hash, _ = common.Uint256FromBytes(hash[:])
	return tx
}

// Payload define the func for loading the payload data
// base on payload type which have different structure
type Payload interface {
	// Get payload data
	Data(version byte) []byte

	Serialize(w io.Writer, version byte) error

	Deserialize(r io.Reader, version byte) error
}

func GetPayload(txType TxType) (Payload, error) {
	var p Payload
	switch txType {
	case CoinBase:
		p = new(PayloadCoinBase)
	case RegisterAsset:
		p = new(PayloadRegisterAsset)
	case TransferAsset:
		p = new(PayloadTransferAsset)
	case Record:
		p = new(PayloadRecord)
	case SideChainPow:
		p = new(PayloadSideChainPow)
	case WithdrawFromSideChain:
		p = new(PayloadWithdrawFromSideChain)
	case TransferCrossChainAsset:
		p = new(PayloadTransferCrossChainAsset)
	case RegisterProducer:
		p = new(PayloadRegisterProducer)
	case CancelProducer:
		p = new(PayloadCancelProducer)
	case UpdateProducer:
		p = new(PayloadUpdateProducer)
	case ReturnDepositCoin:
		p = new(PayloadReturnDepositCoin)
	case IllegalProposalEvidence:
		p = new(PayloadIllegalProposal)
	case IllegalVoteEvidence:
		p = new(PayloadIllegalVote)
	case IllegalBlockEvidence:
		p = new(PayloadIllegalBlock)
	default:
		return nil, errors.New("[Transaction], invalid transaction type.")
	}
	return p, nil
}
