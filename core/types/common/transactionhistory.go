package common

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

type VoteCategory byte

const (
	DPoS        VoteCategory = 0x01
	CRC         VoteCategory = 0x02
	Proposal    VoteCategory = 0x04
	Impeachment VoteCategory = 0x08
	DPoSV2      VoteCategory = 0x10
	Cancel      VoteCategory = 0x80
)

var TxTypeEnum = map[TxType]string{
	CoinBase:                "CoinBase",
	RegisterAsset:           "RegisterAsset",
	TransferAsset:           "TransferAsset",
	Record:                  "Record",
	Deploy:                  "Deploy",
	SideChainPow:            "SideChainPow",
	RechargeToSideChain:     "RechargeToSideChain",
	WithdrawFromSideChain:   "WithdrawFromSideChain",
	TransferCrossChainAsset: "TransferCrossChainAsset",
	RegisterProducer:        "RegisterProducer",
	CancelProducer:          "CancelProducer",
	UpdateProducer:          "UpdateProducer",
	ReturnDepositCoin:       "ReturnDepositCoin",
	ActivateProducer:        "ActivateProducer",

	IllegalProposalEvidence:  "IllegalProposalEvidence",
	IllegalVoteEvidence:      "IllegalVoteEvidence",
	IllegalBlockEvidence:     "IllegalBlockEvidence",
	IllegalSidechainEvidence: "IllegalSidechainEvidence",
	InactiveArbitrators:      "InactiveArbitrators",
	UpdateVersion:            "UpdateVersion",
	NextTurnDPOSInfo:         "NextTurnDPOSInfo",

	RegisterCR:          "RegisterCR",
	UnregisterCR:        "UnregisterCR",
	UpdateCR:            "UpdateCR",
	ReturnCRDepositCoin: "ReturnCRDepositCoin",

	CRCProposal:              "CRCProposal",
	CRCProposalReview:        "CRCProposalReview",
	CRCProposalTracking:      "CRCProposalTracking",
	CRCAppropriation:         "CRCAppropriation",
	CRCProposalWithdraw:      "CRCProposalWithdraw",
	CRCProposalRealWithdraw:  "CRCProposalRealWithdraw",
	CRAssetsRectify:          "CRAssetsRectify",
	CRCouncilMemberClaimNode: "CRCouncilMemberClaimNode",
}

type TransactionHistory struct {
	Address  common.Uint168
	Txid     common.Uint256
	Type     []byte
	Value    common.Fixed64
	Time     uint64
	Height   uint64
	Fee      common.Fixed64
	Inputs   []common.Uint168
	Outputs  []common.Uint168
	TxType   TxType
	VoteType VoteCategory
	Memo     []byte
	Status   uint64
}

type TransactionHistoryDisplay struct {
	Address  string       `json:"address"`
	Txid     string       `json:"txid"`
	Type     string       `json:"type"`
	Value    string       `json:"value"`
	Time     uint64       `json:"time"`
	Height   uint64       `json:"height"`
	Fee      string       `json:"fee"`
	Inputs   []string     `json:"inputs"`
	Outputs  []string     `json:"outputs"`
	TxType   TxType       `json:"txtype"`
	VoteType VoteCategory `json:"votecategory"`
	Memo     string       `json:"memo"`
	Status   string       `json:",omitempty"`
}

func (th *TransactionHistory) Serialize(w io.Writer) error {
	err := common.WriteVarBytes(w, th.Address.Bytes())
	if err != nil {
		return errors.New("[TransactionHistory], Address serialize failed.")
	}
	err = common.WriteVarBytes(w, th.Txid.Bytes())
	if err != nil {
		return errors.New("[TransactionHistory], Txid serialize failed.")
	}
	err = common.WriteVarBytes(w, th.Type)
	if err != nil {
		return errors.New("[TransactionHistory], Type serialize failed.")
	}
	err = th.Value.Serialize(w)
	if err != nil {
		return errors.New("[TransactionHistory], Amount serialize failed.")
	}
	err = common.WriteUint64(w, th.Time)
	if err != nil {
		return errors.New("[TransactionHistory], Time serialize failed.")
	}
	err = common.WriteUint64(w, th.Height)
	if err != nil {
		return errors.New("[TransactionHistory], Height serialize failed.")
	}
	err = th.Fee.Serialize(w)
	if err != nil {
		return errors.New("[TransactionHistory], Fee serialize failed.")
	}
	err = common.WriteVarUint(w, uint64(len(th.Inputs)))
	if err != nil {
		return errors.New("[TransactionHistory], Length of inputs serialize failed.")
	}
	for i := 0; i < len(th.Inputs); i++ {
		err = common.WriteVarBytes(w, th.Inputs[i].Bytes())
		if err != nil {
			return errors.New("[TransactionHistory], input:" + string(th.Inputs[i].Bytes()) + " serialize failed.")
		}
	}
	err = common.WriteVarUint(w, uint64(len(th.Outputs)))
	if err != nil {
		return errors.New("[TransactionHistory], Length of outputs serialize failed.")
	}
	for i := 0; i < len(th.Outputs); i++ {
		err = common.WriteVarBytes(w, th.Outputs[i].Bytes())
		if err != nil {
			return errors.New("[TransactionHistory], output:" + string(th.Outputs[i].Bytes()) + " serialize failed.")
		}
	}
	err = common.WriteVarBytes(w, []byte{byte(th.TxType)})
	if err != nil {
		return errors.New("[TransactionHistory], TxType serialize failed.")
	}
	err = common.WriteVarBytes(w, []byte{byte(th.VoteType)})
	if err != nil {
		return errors.New("[TransactionHistory], VoteCategory serialize failed.")
	}
	err = common.WriteVarBytes(w, th.Memo)
	if err != nil {
		return errors.New("[TransactionHistory], Memo serialize failed.")
	}
	err = common.WriteUint64(w, th.Status)
	if err != nil {
		return errors.New("[TransactionHistory], Status serialize failed.")
	}
	return nil
}

func (th *TransactionHistory) Deserialize(r io.Reader) (*TransactionHistoryDisplay, error) {
	var err error
	txhd := new(TransactionHistoryDisplay)
	buf, err := common.ReadVarBytes(r, 1024, "address")
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Address deserialize failed.")
	}
	th.Address.Deserialize(bytes.NewBuffer(buf))
	txhd.Address, _ = th.Address.ToAddress()

	buf, err = common.ReadVarBytes(r, 1024, "txid")
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Txid deserialize failed.")
	}
	th.Txid.Deserialize(bytes.NewBuffer(buf))
	txhd.Txid, _ = common.ReverseHexString(th.Txid.String())

	th.Type, err = common.ReadVarBytes(r, 1024, "transfer type")
	txhd.Type = string(th.Type)
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Type deserialize failed.")
	}
	err = th.Value.Deserialize(r)
	txhd.Value = th.Value.String()
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Amount deserialize failed.")
	}
	th.Time, err = common.ReadUint64(r)
	txhd.Time = th.Time
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Time deserialize failed.")
	}
	th.Height, err = common.ReadUint64(r)
	txhd.Height = th.Height
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Height deserialize failed.")
	}
	err = th.Fee.Deserialize(r)
	txhd.Fee = th.Fee.String()
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Fee deserialize failed.")
	}
	n, err := common.ReadVarUint(r, 0)
	if err != nil {
		return txhd, errors.New("[TransactionHistory], length of inputs deserialize failed.")
	}
	for i := uint64(0); i < n; i++ {
		programHash := common.Uint168{}
		buf, err = common.ReadVarBytes(r, 1024, "address")
		if err != nil {
			return txhd, errors.New("[TransactionHistory], input deserialize failed.")
		}
		programHash.Deserialize(bytes.NewBuffer(buf))
		th.Inputs = append(th.Inputs, programHash)
		addr, _ := programHash.ToAddress()
		txhd.Inputs = append(txhd.Inputs, addr)
	}
	n, err = common.ReadVarUint(r, 0)
	if err != nil {
		return txhd, errors.New("[TransactionHistory], length of outputs deserialize failed.")
	}
	for i := uint64(0); i < n; i++ {
		programHash := common.Uint168{}
		buf, err = common.ReadVarBytes(r, 1024, "address")
		if err != nil {
			return txhd, errors.New("[TransactionHistory], output deserialize failed.")
		}
		programHash.Deserialize(bytes.NewBuffer(buf))
		th.Outputs = append(th.Outputs, programHash)
		addr, _ := programHash.ToAddress()
		txhd.Outputs = append(txhd.Outputs, addr)
	}
	content, err := common.ReadVarBytes(r, 1, "TxType")
	if err != nil {
		return txhd, errors.New("[TransactionHistory], TxType serialize failed.")
	}
	th.TxType = TxType(content[0])
	txhd.TxType = th.TxType
	content, err = common.ReadVarBytes(r, 1, "VoteCategory")
	if err != nil {
		return txhd, errors.New("[TransactionHistory], VoteCategory serialize failed.")
	}
	th.VoteType = VoteCategory(content[0])
	txhd.VoteType = th.VoteType
	th.Memo, err = common.ReadVarBytes(r, common.MaxVarStringLength, "memo")
	txhd.Memo = string(th.Memo)
	if err != nil {
		return txhd, errors.New("[TransactionHistory], Memo serialize failed.")
	}

	th.Status, err = common.ReadUint64(r)
	if err != nil {
		txhd.Status = "confirmed"
		return txhd, nil
	}
	var status = int64(th.Status)
	if status == 0 {
		txhd.Status = "confirmed"
	} else {
		txhd.Status = "pending"
	}
	return txhd, nil
}

func (th TransactionHistory) String() string {
	return fmt.Sprintf("addr: %s,txid: %s,value: %d,height: %d", th.Address, th.Txid, th.Value, th.Height)
}

// TransactionHistorySorter implements sort.Interface for []TransactionHistory based on
// the Height field.
type TransactionHistorySorter []TransactionHistoryDisplay

func (a TransactionHistorySorter) Len() int      { return len(a) }
func (a TransactionHistorySorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TransactionHistorySorter) Less(i, j int) bool {
	return a[i].Height < a[j].Height
}

type TransactionHistorySorterDesc []TransactionHistoryDisplay

func (a TransactionHistorySorterDesc) Len() int      { return len(a) }
func (a TransactionHistorySorterDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TransactionHistorySorterDesc) Less(i, j int) bool {
	if a[i].Height == 0 {
		return true
	}
	if a[j].Height == 0 {
		return false
	}
	return a[i].Height > a[j].Height
}

func (a TransactionHistorySorter) Filter(skip, limit uint32) TransactionHistorySorter {
	rst := TransactionHistorySorter{}
	for i, v := range a {
		if uint32(i) < skip {
			continue
		}
		rst = append(rst, v)
		if uint32(len(rst)) == limit {
			break
		}
	}
	return rst
}

func (a TransactionHistorySorterDesc) Filter(skip, limit uint32) TransactionHistorySorterDesc {
	rst := TransactionHistorySorterDesc{}
	for i, v := range a {
		if uint32(i) < skip {
			continue
		}
		rst = append(rst, v)
		if uint32(len(rst)) == limit {
			break
		}
	}
	return rst
}
