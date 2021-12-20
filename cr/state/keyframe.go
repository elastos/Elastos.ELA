// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils"
)

// MemberState defines states during a CR member lifetime
type MemberState byte

const (
	// MemberElected indicates the CR member is Elected.
	MemberElected MemberState = iota

	// MemberImpeached indicates the CR member was impeached.
	MemberImpeached

	// MemberTerminated indicates the CR member was terminated because elected
	// CR members are not enough.
	MemberTerminated

	// MemberReturned indicates the CR member has deposit returned.
	MemberReturned

	// MemberInactive indicates the CR member was inactive because the dpos node
	// is inactive.
	MemberInactive

	// MemberIllegal indicates the CR member was illegal because the dpos node
	// is illegal.
	MemberIllegal
)

func (s *MemberState) String() string {
	switch *s {
	case MemberElected:
		return "Elected"
	case MemberImpeached:
		return "Impeached"
	case MemberReturned:
		return "Returned"
	case MemberTerminated:
		return "Terminated"
	case MemberInactive:
		return "Inactive"
	case MemberIllegal:
		return "Illegal"
	}

	return "Unknown"
}

type BudgetStatus uint8

const (
	// Unfinished indicates the proposal owner haven't started or uploaded the
	// status of the budget to CR Council.
	Unfinished BudgetStatus = iota

	// Withdrawable indicates the proposal owner have finished the milestone,
	// but haven't get the payment.
	Withdrawable

	// Withdrawn indicates the proposal owner have finished the milestone and
	// got the payment.
	Withdrawn

	// Rejected indicates the proposal owner have uploaded the status to CR
	// Council but the secretary-general rejected my request.
	Rejected

	// Closed indicates the proposal has been terminated and the milestone will
	// not be finished forever.
	Closed
)

func (s *BudgetStatus) Name() string {
	switch *s {
	case Unfinished:
		return "Unfinished"
	case Withdrawable:
		return "Withdrawable"
	case Withdrawn:
		return "Withdrawn"
	case Rejected:
		return "Rejected"
	case Closed:
		return "Closed"
	}
	return "Unknown"
}

// CRMember defines CR committee member related Info.
type CRMember struct {
	Info                   payload.CRInfo
	ImpeachmentVotes       common.Fixed64
	DepositHash            common.Uint168
	MemberState            MemberState
	DPOSPublicKey          []byte
	InactiveSince          uint32
	ActivateRequestHeight  uint32
	PenaltyBlockCount      uint32
	InactiveCount          uint32
	InactiveCountingHeight uint32
}

// StateKeyFrame holds necessary State about CR committee.
type KeyFrame struct {
	Members             map[common.Uint168]*CRMember
	HistoryMembers      map[uint64]map[common.Uint168]*CRMember
	PartProposalResults []payload.ProposalResult
	DetailCRVotes       map[common.Uint256]payload.DetailVoteInfo // key: hash of DetailVoteInfo

	LastCommitteeHeight      uint32
	LastVotingStartHeight    uint32
	InElectionPeriod         bool
	NeedAppropriation        bool
	NeedRecordProposalResult bool
	CRCFoundationBalance     common.Fixed64
	CRCCommitteeBalance      common.Fixed64
	CRCCommitteeUsedAmount   common.Fixed64
	CRCCurrentStageAmount    common.Fixed64
	DestroyedAmount          common.Fixed64
	CirculationAmount        common.Fixed64
	AppropriationAmount      common.Fixed64
	CommitteeUsedAmount      common.Fixed64

	CRAssetsAddressUTXOCount uint32
}

type DepositInfo struct {
	DepositAmount common.Fixed64
	Penalty       common.Fixed64
	TotalAmount   common.Fixed64
}

// StateKeyFrame holds necessary State about CR State.
type StateKeyFrame struct {
	CodeCIDMap           map[string]common.Uint168
	DepositHashCIDMap    map[common.Uint168]common.Uint168
	Candidates           map[common.Uint168]*Candidate
	HistoryCandidates    map[uint64]map[common.Uint168]*Candidate
	DepositInfo          map[common.Uint168]*DepositInfo
	CurrentSession       uint64
	Nicknames            map[string]struct{}
	Votes                map[string]struct{}
	DepositOutputs       map[string]common.Fixed64
	CRCFoundationOutputs map[string]common.Fixed64
	CRCCommitteeOutputs  map[string]common.Fixed64
}

// ProposalState defines necessary State about an CR proposals.
type ProposalState struct {
	Status             ProposalStatus
	Proposal           payload.CRCProposalInfo
	TxHash             common.Uint256
	TxPayloadVer       byte
	CRVotes            map[common.Uint168]payload.VoteResult
	VotersRejectAmount common.Fixed64
	RegisterHeight     uint32
	VoteStartHeight    uint32

	WithdrawnBudgets    map[uint8]common.Fixed64 // proposalWithdraw
	WithdrawableBudgets map[uint8]common.Fixed64 // proposalTracking
	BudgetsStatus       map[uint8]BudgetStatus
	FinalPaymentStatus  bool

	TrackingCount    uint8
	TerminatedHeight uint32
	ProposalOwner    []byte
	Recipient        common.Uint168
}

type ProposalHashSet map[common.Uint256]struct{}

func NewProposalHashSet() ProposalHashSet {
	return make(ProposalHashSet)
}

func (set *ProposalHashSet) Add(proposalHash common.Uint256) bool {
	_, found := (*set)[proposalHash]
	if found {
		return false //False if it existed already
	}
	(*set)[proposalHash] = struct{}{}
	return true
}

func (set *ProposalHashSet) Clear() {
	*set = NewProposalHashSet()
}

func (set *ProposalHashSet) Remove(proposalHash common.Uint256) {
	delete(*set, proposalHash)
}

func (set *ProposalHashSet) Contains(proposalHash common.Uint256) bool {
	if _, ok := (*set)[proposalHash]; !ok {
		return false
	}
	return true
}

func (set *ProposalHashSet) Len() int {
	return len(*set)
}

func (set *ProposalHashSet) Equal(other ProposalHashSet) bool {
	if set.Len() != other.Len() {
		return false
	}
	for elem := range *set {
		if !other.Contains(elem) {
			return false
		}
	}
	return true
}

type ProposalsMap map[common.Uint256]*ProposalState
type ReviewDraftDataMap map[common.Uint256][]byte
type TrackingDraftDataMap map[common.Uint256][]byte

// ProposalKeyFrame holds all runtime State about CR proposals.
type ProposalKeyFrame struct {
	// key is did value is proposalhash set
	Proposals       ProposalsMap
	ProposalHashes  map[common.Uint168]ProposalHashSet
	ProposalSession map[uint64][]common.Uint256
	// proposalWithdraw Info
	WithdrawableTxInfo map[common.Uint256]common2.OutputInfo
	// publicKey of SecretaryGeneral
	SecretaryGeneralPublicKey string
	// reserved custom id list
	ReservedCustomIDLists []string
	// received custom id list
	PendingReceivedCustomIDMap map[string]struct{}
	ReceivedCustomIDLists      []string
	// registered side chain name
	RegisteredSideChainNames []string
	// magic numbers
	RegisteredMagicNumbers []uint32
	// genesis hashes
	RegisteredGenesisHashes []common.Uint256

	// store register Info with the approved Height
	RegisteredSideChainPayloadInfo map[uint32]map[common.Uint256]payload.SideChainInfo

	// reserve CustomID
	ReservedCustomID bool

	// detailed CRC proposal votes information
	DetailCRCProposalVotes map[common.Uint256]payload.DetailVoteInfo // key: hash of DetailVoteInfo
}

func NewProposalMap() ProposalsMap {
	return make(ProposalsMap)
}

func (c *CRMember) Serialize(w io.Writer) (err error) {
	if err = c.Info.SerializeUnsigned(w, payload.CRInfoDIDVersion); err != nil {
		return
	}

	if err = c.ImpeachmentVotes.Serialize(w); err != nil {
		return
	}

	if err = c.DepositHash.Serialize(w); err != nil {
		return
	}

	if err = common.WriteUint8(w, uint8(c.MemberState)); err != nil {
		return
	}

	if err = common.WriteVarBytes(w, c.DPOSPublicKey); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.InactiveSince); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.ActivateRequestHeight); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.PenaltyBlockCount); err != nil {
		return
	}
	if err = common.WriteUint32(w, c.InactiveCount); err != nil {
		return
	}
	return common.WriteUint32(w, c.InactiveCountingHeight)
}

func (c *CRMember) Deserialize(r io.Reader) (err error) {
	if err = c.Info.DeserializeUnsigned(r, payload.CRInfoDIDVersion); err != nil {
		return
	}

	if err = c.ImpeachmentVotes.Deserialize(r); err != nil {
		return
	}

	if err = c.DepositHash.Deserialize(r); err != nil {
		return
	}
	var state uint8
	if state, err = common.ReadUint8(r); err == nil {
		c.MemberState = MemberState(state)
	}

	c.DPOSPublicKey, err = common.ReadVarBytes(r, crypto.COMPRESSEDLEN, "public key")
	if err != nil {
		return
	}

	if c.InactiveSince, err = common.ReadUint32(r); err != nil {
		return
	}

	if c.ActivateRequestHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if c.PenaltyBlockCount, err = common.ReadUint32(r); err != nil {
		return
	}
	if c.InactiveCount, err = common.ReadUint32(r); err != nil {
		return
	}
	if c.InactiveCountingHeight, err = common.ReadUint32(r); err != nil {
		return
	}
	return
}

func (kf *KeyFrame) Serialize(w io.Writer) (err error) {
	if err = kf.serializeMembersMap(w, kf.Members); err != nil {
		return
	}

	if err = kf.serializeHistoryMembersMap(w, kf.HistoryMembers); err != nil {
		return
	}

	if err = kf.serializeProposalResultList(w, kf.PartProposalResults); err != nil {
		return
	}

	if err = serializeDetailVoteInfoMap(w, kf.DetailCRVotes); err != nil {
		return
	}

	return common.WriteElements(w, kf.LastCommitteeHeight,
		kf.LastVotingStartHeight, kf.InElectionPeriod, kf.NeedAppropriation,
		kf.NeedRecordProposalResult, kf.CRCFoundationBalance,
		kf.CRCCommitteeBalance, kf.CRCCommitteeUsedAmount, kf.CRCCurrentStageAmount,
		kf.DestroyedAmount, kf.CirculationAmount, kf.AppropriationAmount,
		kf.CommitteeUsedAmount, kf.CRAssetsAddressUTXOCount)
}

func (kf *KeyFrame) Deserialize(r io.Reader) (err error) {
	if kf.Members, err = kf.deserializeMembersMap(r); err != nil {
		return
	}

	if kf.HistoryMembers, err = kf.deserializeHistoryMembersMap(r); err != nil {
		return
	}

	if kf.PartProposalResults, err = kf.deserializeProposalResultList(r); err != nil {
		return
	}

	if kf.DetailCRVotes, err = deserializeDetailVoteInfoMap(r); err != nil {
		return
	}

	err = common.ReadElements(r, &kf.LastCommitteeHeight,
		&kf.LastVotingStartHeight, &kf.InElectionPeriod, &kf.NeedAppropriation,
		&kf.NeedRecordProposalResult, &kf.CRCFoundationBalance, &kf.CRCCommitteeBalance,
		&kf.CRCCommitteeUsedAmount, &kf.CRCCurrentStageAmount, &kf.DestroyedAmount, &kf.CirculationAmount,
		&kf.AppropriationAmount, &kf.CommitteeUsedAmount, &kf.CRAssetsAddressUTXOCount)

	return
}

func (kf *KeyFrame) serializeMembersMap(w io.Writer,
	mmap map[common.Uint168]*CRMember) (err error) {
	if err = common.WriteVarUint(w, uint64(len(mmap))); err != nil {
		return
	}
	for k, v := range mmap {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (kf *KeyFrame) serializeHistoryMembersMap(w io.Writer,
	hmap map[uint64]map[common.Uint168]*CRMember) (err error) {
	if err = common.WriteVarUint(w, uint64(len(hmap))); err != nil {
		return
	}
	for k, v := range hmap {
		if err = common.WriteVarUint(w, k); err != nil {
			return
		}

		if err = kf.serializeMembersMap(w, v); err != nil {
			return
		}
	}

	return
}

func (kf *KeyFrame) serializeAmountList(w io.Writer,
	amounts []common.Fixed64) (err error) {
	if err = common.WriteVarUint(w, uint64(len(amounts))); err != nil {
		return
	}
	for _, a := range amounts {
		if err = a.Serialize(w); err != nil {
			return err
		}
	}
	return
}

func (kf *KeyFrame) serializeProposalResultList(w io.Writer,
	results []payload.ProposalResult) (err error) {
	if err = common.WriteVarUint(w, uint64(len(results))); err != nil {
		return
	}
	for _, a := range results {
		if err = a.Serialize(w, payload.CustomIDResultVersion); err != nil {
			return err
		}
	}
	return
}
func serializeDetailVoteInfoMap(w io.Writer, vmap map[common.Uint256]payload.DetailVoteInfo) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k, v := range vmap {
		if err = k.Serialize(w); err != nil {
			return
		}
		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func deserializeDetailVoteInfoMap(
	r io.Reader) (vmap map[common.Uint256]payload.DetailVoteInfo, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint256]payload.DetailVoteInfo)
	for i := uint64(0); i < count; i++ {
		var k common.Uint256
		if err = k.Deserialize(r); err != nil {
			return
		}
		var v payload.DetailVoteInfo
		if err = v.Deserialize(r); err != nil {
			return
		}
		vmap[k] = v
	}
	return
}

func (kf *KeyFrame) deserializeProposalResultList(
	r io.Reader) (results []payload.ProposalResult, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	results = make([]payload.ProposalResult, 0)
	for i := uint64(0); i < count; i++ {
		var amount payload.ProposalResult
		if err = amount.Deserialize(r, payload.CustomIDResultVersion); err != nil {
			return
		}
		results = append(results, amount)
	}
	return
}

func (kf *KeyFrame) deserializeAmountList(
	r io.Reader) (amounts []common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	amounts = make([]common.Fixed64, 0)
	for i := uint64(0); i < count; i++ {
		var amount common.Fixed64
		if err = amount.Deserialize(r); err != nil {
			return
		}
		amounts = append(amounts, amount)
	}
	return
}

func (kf *KeyFrame) deserializeMembersMap(
	r io.Reader) (mmap map[common.Uint168]*CRMember, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	mmap = make(map[common.Uint168]*CRMember)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}
		candidate := &CRMember{}
		if err = candidate.Deserialize(r); err != nil {
			return
		}
		mmap[k] = candidate
	}
	return
}

func (kf *KeyFrame) deserializeHistoryMembersMap(
	r io.Reader) (hmap map[uint64]map[common.Uint168]*CRMember, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	hmap = make(map[uint64]map[common.Uint168]*CRMember)
	for i := uint64(0); i < count; i++ {
		var k uint64
		k, err = common.ReadVarUint(r, 0)
		if err != nil {
			return
		}
		var cmap map[common.Uint168]*CRMember
		cmap, err = kf.deserializeMembersMap(r)
		if err != nil {
			return
		}
		hmap[k] = cmap
	}
	return
}

func (kf *KeyFrame) Snapshot() *KeyFrame {
	frame := NewKeyFrame()
	frame.LastCommitteeHeight = kf.LastCommitteeHeight
	frame.LastVotingStartHeight = kf.LastVotingStartHeight
	frame.InElectionPeriod = kf.InElectionPeriod
	frame.NeedAppropriation = kf.NeedAppropriation
	frame.NeedRecordProposalResult = kf.NeedRecordProposalResult

	frame.CRCFoundationBalance = kf.CRCFoundationBalance
	frame.CRCCommitteeBalance = kf.CRCCommitteeBalance
	frame.CRCCommitteeUsedAmount = kf.CRCCommitteeUsedAmount
	frame.CRCCurrentStageAmount = kf.CRCCurrentStageAmount

	frame.DestroyedAmount = kf.DestroyedAmount
	frame.CirculationAmount = kf.CirculationAmount
	frame.AppropriationAmount = kf.AppropriationAmount
	frame.CommitteeUsedAmount = kf.CommitteeUsedAmount
	frame.Members = copyMembersMap(kf.Members)
	frame.HistoryMembers = copyHistoryMembersMap(kf.HistoryMembers)
	frame.CRAssetsAddressUTXOCount = kf.CRAssetsAddressUTXOCount
	return frame
}

func NewKeyFrame() *KeyFrame {
	return &KeyFrame{
		Members:             make(map[common.Uint168]*CRMember, 0),
		HistoryMembers:      make(map[uint64]map[common.Uint168]*CRMember, 0),
		LastCommitteeHeight: 0,
	}
}

func (d *DepositInfo) Serialize(w io.Writer) (err error) {
	if err = d.DepositAmount.Serialize(w); err != nil {
		return
	}

	if err = d.Penalty.Serialize(w); err != nil {
		return
	}

	if err = d.TotalAmount.Serialize(w); err != nil {
		return
	}

	return
}

func (d *DepositInfo) Deserialize(r io.Reader) (err error) {
	if err = d.DepositAmount.Deserialize(r); err != nil {
		return
	}

	if err = d.Penalty.Deserialize(r); err != nil {
		return
	}

	if err = d.TotalAmount.Deserialize(r); err != nil {
		return
	}

	return
}

func (kf *StateKeyFrame) Serialize(w io.Writer) (err error) {
	if err = kf.serializeCodeAddressMap(w, kf.CodeCIDMap); err != nil {
		return
	}

	if err = kf.serializeDepositCIDMap(w, kf.DepositHashCIDMap); err != nil {
		return
	}

	if err = kf.serializeCandidateMap(w, kf.Candidates); err != nil {
		return
	}

	if err = kf.serializeHistoryCandidateMap(w, kf.HistoryCandidates); err != nil {
		return
	}

	if err = kf.serializeDepositInfoMap(w, kf.DepositInfo); err != nil {
		return
	}

	if err = common.WriteVarUint(w, kf.CurrentSession); err != nil {
		return
	}

	if err = utils.SerializeStringSet(w, kf.Nicknames); err != nil {
		return
	}

	if err = utils.SerializeStringSet(w, kf.Votes); err != nil {
		return
	}

	if err = kf.SerializeFixed64Map(w, kf.DepositOutputs); err != nil {
		return
	}

	if err = kf.SerializeFixed64Map(w, kf.CRCFoundationOutputs); err != nil {
		return
	}

	return kf.SerializeFixed64Map(w, kf.CRCCommitteeOutputs)
}

func (kf *StateKeyFrame) Deserialize(r io.Reader) (err error) {
	if kf.CodeCIDMap, err = kf.deserializeCodeAddressMap(r); err != nil {
		return
	}

	if kf.DepositHashCIDMap, err = kf.deserializeDepositCIDMap(r); err != nil {
		return
	}

	if kf.Candidates, err = kf.deserializeCandidateMap(r); err != nil {
		return
	}

	if kf.HistoryCandidates, err = kf.deserializeHistoryCandidateMap(r); err != nil {
		return
	}

	if kf.DepositInfo, err = kf.deserializeDepositInfoMap(r); err != nil {
		return
	}

	if kf.CurrentSession, err = common.ReadVarUint(r, 0); err != nil {
		return
	}

	if kf.Nicknames, err = utils.DeserializeStringSet(r); err != nil {
		return
	}

	if kf.Votes, err = utils.DeserializeStringSet(r); err != nil {
		return
	}

	if kf.DepositOutputs, err = kf.DeserializeFixed64Map(r); err != nil {
		return
	}

	if kf.CRCFoundationOutputs, err = kf.DeserializeFixed64Map(r); err != nil {
		return
	}

	if kf.CRCCommitteeOutputs, err = kf.DeserializeFixed64Map(r); err != nil {
		return
	}
	return
}

func (kf *StateKeyFrame) serializeCodeAddressMap(w io.Writer,
	cmap map[string]common.Uint168) (err error) {
	if err = common.WriteVarUint(w, uint64(len(cmap))); err != nil {
		return
	}
	for k, v := range cmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (kf *StateKeyFrame) deserializeCodeAddressMap(r io.Reader) (
	cmap map[string]common.Uint168, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	cmap = make(map[string]common.Uint168)

	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		var v common.Uint168
		if err = v.Deserialize(r); err != nil {
			return
		}
		cmap[k] = v
	}
	return
}

func (kf *StateKeyFrame) serializeDepositCIDMap(w io.Writer,
	cmap map[common.Uint168]common.Uint168) (err error) {
	if err = common.WriteVarUint(w, uint64(len(cmap))); err != nil {
		return
	}
	for k, v := range cmap {
		if err = k.Serialize(w); err != nil {
			return
		}
		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (kf *StateKeyFrame) deserializeDepositCIDMap(r io.Reader) (
	cmap map[common.Uint168]common.Uint168, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	cmap = make(map[common.Uint168]common.Uint168)

	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}
		var v common.Uint168
		if err = v.Deserialize(r); err != nil {
			return
		}
		cmap[k] = v
	}
	return
}

func (kf *StateKeyFrame) serializeCandidateMap(w io.Writer,
	cmap map[common.Uint168]*Candidate) (err error) {
	if err = common.WriteVarUint(w, uint64(len(cmap))); err != nil {
		return
	}
	for k, v := range cmap {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (kf *StateKeyFrame) serializeHistoryCandidateMap(w io.Writer,
	hmap map[uint64]map[common.Uint168]*Candidate) (err error) {
	if err = common.WriteVarUint(w, uint64(len(hmap))); err != nil {
		return
	}
	for k, v := range hmap {
		if err = common.WriteVarUint(w, k); err != nil {
			return
		}

		if err = kf.serializeCandidateMap(w, v); err != nil {
			return
		}
	}

	return
}

func (kf *StateKeyFrame) serializeDepositInfoMap(w io.Writer,
	vmap map[common.Uint168]*DepositInfo) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k, v := range vmap {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}

	return
}

func (kf *StateKeyFrame) deserializeCandidateMap(
	r io.Reader) (cmap map[common.Uint168]*Candidate, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	cmap = make(map[common.Uint168]*Candidate)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}
		candidate := &Candidate{}
		if err = candidate.Deserialize(r); err != nil {
			return
		}
		cmap[k] = candidate
	}
	return
}

func (kf *StateKeyFrame) deserializeDepositInfoMap(
	r io.Reader) (vmap map[common.Uint168]*DepositInfo, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint168]*DepositInfo)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}
		v := &DepositInfo{}
		if err = v.Deserialize(r); err != nil {
			return
		}
		vmap[k] = v
	}
	return
}

func (kf *StateKeyFrame) deserializeHistoryCandidateMap(
	r io.Reader) (hmap map[uint64]map[common.Uint168]*Candidate, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	hmap = make(map[uint64]map[common.Uint168]*Candidate)
	for i := uint64(0); i < count; i++ {
		var k uint64
		k, err = common.ReadVarUint(r, 0)
		if err != nil {
			return
		}
		var cmap map[common.Uint168]*Candidate
		cmap, err = kf.deserializeCandidateMap(r)
		if err != nil {
			return
		}
		hmap[k] = cmap
	}
	return
}

func (kf *StateKeyFrame) SerializeFixed64Map(w io.Writer,
	vmap map[string]common.Fixed64) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k, v := range vmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (kf *StateKeyFrame) DeserializeFixed64Map(
	r io.Reader) (vmap map[string]common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[string]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		var v common.Fixed64
		if err = v.Deserialize(r); err != nil {
			return
		}
		vmap[k] = v
	}
	return
}

// Snapshot will create a new StateKeyFrame object and deep copy all related data.
func (kf *StateKeyFrame) Snapshot() *StateKeyFrame {
	state := NewStateKeyFrame()
	state.CodeCIDMap = copyCodeAddressMap(kf.CodeCIDMap)
	state.DepositHashCIDMap = copyHashIDMap(kf.DepositHashCIDMap)
	state.Candidates = copyCandidateMap(kf.Candidates)
	state.HistoryCandidates = copyHistoryCandidateMap(kf.HistoryCandidates)
	state.DepositInfo = copyDepositInfoMap(kf.DepositInfo)
	state.CurrentSession = kf.CurrentSession
	state.Nicknames = utils.CopyStringSet(kf.Nicknames)
	state.Votes = utils.CopyStringSet(kf.Votes)
	state.DepositOutputs = copyFixed64Map(kf.DepositOutputs)
	state.CRCFoundationOutputs = copyFixed64Map(kf.CRCFoundationOutputs)
	state.CRCCommitteeOutputs = copyFixed64Map(kf.CRCCommitteeOutputs)

	return state
}

func (p *ProposalState) Serialize(w io.Writer) (err error) {
	if err = p.Proposal.Serialize(w, payload.CRCProposalVersion); err != nil {
		return
	}

	if err = common.WriteUint8(w, uint8(p.Status)); err != nil {
		return
	}

	if err = common.WriteUint8(w, p.TxPayloadVer); err != nil {
		return
	}

	if err = common.WriteUint32(w, p.RegisterHeight); err != nil {
		return
	}

	if err = common.WriteUint32(w, p.VoteStartHeight); err != nil {
		return
	}

	if err = common.WriteUint64(w, uint64(p.VotersRejectAmount)); err != nil {
		return
	}

	if err = common.WriteVarUint(w, uint64(len(p.CRVotes))); err != nil {
		return
	}

	for k, v := range p.CRVotes {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = common.WriteUint8(w, uint8(v)); err != nil {
			return
		}
	}
	if err = p.serializeBudgets(p.WithdrawnBudgets, w); err != nil {
		return
	}
	if err = p.serializeBudgets(p.WithdrawableBudgets, w); err != nil {
		return
	}
	if err = p.serializeBudgetsStatus(p.BudgetsStatus, w); err != nil {
		return
	}
	if err := common.WriteElement(w, p.FinalPaymentStatus); err != nil {
		return err
	}
	if err = common.WriteUint8(w, p.TrackingCount); err != nil {
		return
	}
	if err = common.WriteUint32(w, p.TerminatedHeight); err != nil {
		return
	}
	if err := common.WriteVarBytes(w, p.ProposalOwner); err != nil {
		return err
	}
	if err := p.Recipient.Serialize(w); err != nil {
		return err
	}

	return p.TxHash.Serialize(w)
}

func (p *ProposalState) Deserialize(r io.Reader) (err error) {
	if err = p.Proposal.Deserialize(r, payload.CRCProposalVersion); err != nil {
		return
	}

	var status uint8
	if status, err = common.ReadUint8(r); err != nil {
		return
	}
	p.Status = ProposalStatus(status)

	var payloadVersion uint8
	if payloadVersion, err = common.ReadUint8(r); err != nil {
		return
	}
	p.TxPayloadVer = payloadVersion

	if p.RegisterHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if p.VoteStartHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	var amount uint64
	if amount, err = common.ReadUint64(r); err != nil {
		return
	}
	p.VotersRejectAmount = common.Fixed64(amount)

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}

	p.CRVotes = make(map[common.Uint168]payload.VoteResult, count)
	for i := uint64(0); i < count; i++ {
		var key common.Uint168
		if err = key.Deserialize(r); err != nil {
			return
		}

		var value uint8
		if value, err = common.ReadUint8(r); err != nil {
			return
		}
		p.CRVotes[key] = payload.VoteResult(value)
	}

	if p.WithdrawnBudgets, err = p.deserializeBudgets(r); err != nil {
		return
	}
	if p.WithdrawableBudgets, err = p.deserializeBudgets(r); err != nil {
		return
	}
	if p.BudgetsStatus, err = p.deserializeBudgetsStatus(r); err != nil {
		return
	}
	if err = common.ReadElement(r, &p.FinalPaymentStatus); err != nil {
		return err
	}
	if p.TrackingCount, err = common.ReadUint8(r); err != nil {
		return
	}
	if p.TerminatedHeight, err = common.ReadUint32(r); err != nil {
		return
	}
	if p.ProposalOwner, err = common.ReadVarBytes(r, crypto.NegativeBigLength,
		"proposal owner"); err != nil {
		return err
	}
	if err = p.Recipient.Deserialize(r); err != nil {
		return err
	}

	return p.TxHash.Deserialize(r)
}

func (p *ProposalState) serializeBudgets(withdrawableBudgets map[uint8]common.Fixed64,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(withdrawableBudgets))); err != nil {
		return
	}
	for k, v := range withdrawableBudgets {
		if err = common.WriteElement(w, k); err != nil {
			return
		}
		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (p *ProposalState) serializeBudgetsStatus(budgetsStatus map[uint8]BudgetStatus,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(budgetsStatus))); err != nil {
		return
	}
	for k, v := range budgetsStatus {
		if err = common.WriteElements(w, k, uint8(v)); err != nil {
			return
		}
	}
	return
}

func (p *ProposalState) deserializeBudgets(r io.Reader) (
	withdrawableBudgets map[uint8]common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	withdrawableBudgets = make(map[uint8]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		var stage uint8
		if err = common.ReadElement(r, &stage); err != nil {
			return
		}
		var amount common.Fixed64
		if err = amount.Deserialize(r); err != nil {
			return
		}
		withdrawableBudgets[stage] = amount
	}
	return
}

func (p *ProposalState) deserializeBudgetsStatus(r io.Reader) (
	budgetsStatus map[uint8]BudgetStatus, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	budgetsStatus = make(map[uint8]BudgetStatus)
	for i := uint64(0); i < count; i++ {
		var stage uint8
		var status uint8
		if err = common.ReadElements(r, &stage, &status); err != nil {
			return
		}
		budgetsStatus[stage] = BudgetStatus(status)
	}
	return
}

func (p *ProposalKeyFrame) Serialize(w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(p.Proposals))); err != nil {
		return
	}

	for k, v := range p.Proposals {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	if err = p.serializeProposalHashsMap(p.ProposalHashes, w); err != nil {
		return
	}
	if err = p.serializeProposalSessionMap(p.ProposalSession, w); err != nil {
		return
	}
	if err = p.serializeWithdrawableTransactionsMap(p.WithdrawableTxInfo, w); err != nil {
		return
	}
	if err = common.WriteVarString(w, p.SecretaryGeneralPublicKey); err != nil {
		return
	}

	if err = common.WriteVarUint(w, uint64(len(p.ReservedCustomIDLists))); err != nil {
		return
	}
	for _, name := range p.ReservedCustomIDLists {
		err = common.WriteVarString(w, name)
		if err != nil {
			return
		}
	}

	if err = p.serializeMapStringNULL(w, p.PendingReceivedCustomIDMap); err != nil {
		return
	}

	if err = common.WriteVarUint(w, uint64(len(p.ReceivedCustomIDLists))); err != nil {
		return
	}
	for _, name := range p.ReceivedCustomIDLists {
		err = common.WriteVarString(w, name)
		if err != nil {
			return
		}
	}

	if err = common.WriteVarUint(w, uint64(len(p.RegisteredSideChainNames))); err != nil {
		return
	}
	for _, name := range p.RegisteredSideChainNames {
		err = common.WriteVarString(w, name)
		if err != nil {
			return
		}
	}

	if err = common.WriteVarUint(w, uint64(len(p.RegisteredMagicNumbers))); err != nil {
		return
	}
	for _, magic := range p.RegisteredMagicNumbers {
		err = common.WriteUint32(w, magic)
		if err != nil {
			return
		}
	}

	if err = common.WriteVarUint(w, uint64(len(p.RegisteredGenesisHashes))); err != nil {
		return
	}
	for _, hash := range p.RegisteredGenesisHashes {
		err = hash.Serialize(w)
		if err != nil {
			return
		}
	}

	if err = p.serializeRegisterSideChainData(w, p.RegisteredSideChainPayloadInfo); err != nil {
		return
	}

	////ReservedCustomID
	if err = common.WriteElements(w, p.ReservedCustomID); err != nil {
		return
	}

	if err = serializeDetailVoteInfoMap(w, p.DetailCRCProposalVotes); err != nil {
		return err
	}

	return
}

func (p *ProposalKeyFrame) serializeDraftDataMap(draftData map[common.Uint256][]byte,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(draftData))); err != nil {
		return
	}
	for hash, data := range draftData {
		if err = hash.Serialize(w); err != nil {
			return
		}
		if err = common.WriteVarBytes(w, data); err != nil {
			return
		}
	}
	return
}

func (p *ProposalKeyFrame) serializeMapStringNULL(w io.Writer, data map[string]struct{}) (err error) {
	if err = common.WriteVarUint(w, uint64(len(data))); err != nil {
		return
	}
	for k, _ := range data {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
	}
	return
}

func (p *ProposalKeyFrame) deserializeMapStringNULL(r io.Reader) (
	result map[string]struct{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	result = make(map[string]struct{})
	for i := uint64(0); i < count; i++ {

		var str string
		str, err = common.ReadVarString(r)
		if err != nil {
			return
		}
		result[str] = struct{}{}
	}
	return
}

func (p *ProposalKeyFrame) serializeRegisterSideChainData(w io.Writer,
	data map[uint32]map[common.Uint256]payload.SideChainInfo) (err error) {
	if err = common.WriteVarUint(w, uint64(len(data))); err != nil {
		return
	}
	for k1, v1 := range data {
		// write key
		if err = common.WriteUint32(w, k1); err != nil {
			return
		}

		// write value
		if err = common.WriteVarUint(w, uint64(len(v1))); err != nil {
			return
		}
		for k2, v2 := range v1 {
			if err = k2.Serialize(w); err != nil {
				return
			}
			if err = v2.Serialize(w); err != nil {
				return
			}
		}
	}
	return
}

func (p *ProposalKeyFrame) deserializeRegisterSideChainData(r io.Reader) (
	result map[uint32]map[common.Uint256]payload.SideChainInfo, err error) {
	var count1 uint64
	if count1, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	result = make(map[uint32]map[common.Uint256]payload.SideChainInfo)
	for i := uint64(0); i < count1; i++ {
		var height uint32
		height, err = common.ReadUint32(r)
		if err != nil {
			return
		}

		var count2 uint64
		if count2, err = common.ReadVarUint(r, 0); err != nil {
			return
		}
		data := make(map[common.Uint256]payload.SideChainInfo)
		for i := uint64(0); i < count2; i++ {
			var hash common.Uint256
			err = hash.Deserialize(r)
			if err != nil {
				return
			}

			var sideChainInfo payload.SideChainInfo
			err = sideChainInfo.Deserialize(r)
			if err != nil {
				return
			}

			data[hash] = sideChainInfo
		}

		result[height] = data
	}
	return
}

func (p *ProposalKeyFrame) serializeProposalHashsMap(proposalHashMap map[common.Uint168]ProposalHashSet,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(proposalHashMap))); err != nil {
		return
	}
	for k, ProposalHashSet := range proposalHashMap {
		if err = k.Serialize(w); err != nil {
			return
		}
		if err := common.WriteVarUint(w,
			uint64(len(ProposalHashSet))); err != nil {
			return err
		}
		for proposalHash, _ := range ProposalHashSet {
			if err := proposalHash.Serialize(w); err != nil {
				return err
			}
		}
	}
	return
}

func (p *ProposalKeyFrame) serializeProposalSessionMap(
	proposalSessionMap map[uint64][]common.Uint256, w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(proposalSessionMap))); err != nil {
		return
	}
	for k, v := range proposalSessionMap {
		if err = common.WriteUint64(w, k); err != nil {
			return
		}
		if err := common.WriteVarUint(w,
			uint64(len(v))); err != nil {
			return err
		}
		for _, proposalHash := range v {
			if err := proposalHash.Serialize(w); err != nil {
				return err
			}
		}
	}
	return
}

func (p *ProposalKeyFrame) serializeWithdrawableTransactionsMap(
	proposalWithdrableTx map[common.Uint256]common2.OutputInfo, w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(proposalWithdrableTx))); err != nil {
		return
	}
	for k, v := range proposalWithdrableTx {
		if err = k.Serialize(w); err != nil {
			return
		}
		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (p *ProposalKeyFrame) Deserialize(r io.Reader) (err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}

	p.Proposals = make(map[common.Uint256]*ProposalState, count)
	for i := uint64(0); i < count; i++ {
		var k common.Uint256
		if err = k.Deserialize(r); err != nil {
			return
		}

		var v ProposalState
		if err = v.Deserialize(r); err != nil {
			return
		}
		p.Proposals[k] = &v
	}
	if p.ProposalHashes, err = p.deserializeProposalHashsMap(r); err != nil {
		return
	}
	if p.ProposalSession, err = p.deserializeProposalSessionMap(r); err != nil {
		return
	}
	if p.WithdrawableTxInfo, err = p.deserializeWithdrawableTransactionsMap(r); err != nil {
		return
	}
	if p.SecretaryGeneralPublicKey, err = common.ReadVarString(r); err != nil {
		return
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	p.ReservedCustomIDLists = make([]string, 0)
	for i := uint64(0); i < count; i++ {
		var name string
		name, err = common.ReadVarString(r)
		p.ReservedCustomIDLists = append(p.ReservedCustomIDLists, name)
		if err != nil {
			return
		}
	}

	if p.PendingReceivedCustomIDMap, err = p.deserializeMapStringNULL(r); err != nil {
		return
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	p.ReceivedCustomIDLists = make([]string, 0)
	for i := uint64(0); i < count; i++ {
		var name string
		name, err = common.ReadVarString(r)
		p.ReceivedCustomIDLists = append(p.ReceivedCustomIDLists, name)
		if err != nil {
			return
		}
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	p.RegisteredSideChainNames = make([]string, 0)
	for i := uint64(0); i < count; i++ {
		var name string
		name, err = common.ReadVarString(r)
		p.RegisteredSideChainNames = append(p.RegisteredSideChainNames, name)
		if err != nil {
			return
		}
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	p.RegisteredMagicNumbers = make([]uint32, 0)
	for i := uint64(0); i < count; i++ {
		var magic uint32
		magic, err = common.ReadUint32(r)
		p.RegisteredMagicNumbers = append(p.RegisteredMagicNumbers, magic)
		if err != nil {
			return
		}
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	p.RegisteredGenesisHashes = make([]common.Uint256, 0)
	for i := uint64(0); i < count; i++ {
		var h common.Uint256
		err = h.Deserialize(r)
		if err != nil {
			return err
		}
		p.RegisteredGenesisHashes = append(p.RegisteredGenesisHashes, h)
	}

	if p.RegisteredSideChainPayloadInfo, err = p.deserializeRegisterSideChainData(r); err != nil {
		return
	}

	if err = common.ReadElements(r, &p.ReservedCustomID); err != nil {
		return
	}

	if p.DetailCRCProposalVotes, err = deserializeDetailVoteInfoMap(r); err != nil {
		return
	}

	return
}

func (p *ProposalKeyFrame) deserializeDraftDataMap(r io.Reader) (
	draftDataMap map[common.Uint256][]byte, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	draftDataMap = make(map[common.Uint256][]byte)
	for i := uint64(0); i < count; i++ {

		var hash common.Uint256
		if err = hash.Deserialize(r); err != nil {
			return
		}
		var data []byte
		data, err = common.ReadVarBytes(r, payload.MaxPayloadDataSize, "draft data")
		if err != nil {
			return
		}

		draftDataMap[hash] = data
	}
	return
}

func (p *ProposalKeyFrame) deserializeProposalHashsMap(r io.Reader) (
	proposalHashMap map[common.Uint168]ProposalHashSet, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	proposalHashMap = make(map[common.Uint168]ProposalHashSet)
	for i := uint64(0); i < count; i++ {

		var did common.Uint168
		if err = did.Deserialize(r); err != nil {
			return
		}
		var lenProposalHashSet uint64
		proposalHashSet := NewProposalHashSet()

		if lenProposalHashSet, err = common.ReadVarUint(r, 0); err != nil {
			return
		}
		for i := uint64(0); i < lenProposalHashSet; i++ {
			hash := &common.Uint256{}
			if err = hash.Deserialize(r); err != nil {
				return
			}
			proposalHashSet.Add(*hash)
		}

		proposalHashMap[did] = proposalHashSet
	}
	return
}

func (p *ProposalKeyFrame) deserializeProposalSessionMap(r io.Reader) (
	proposalSessionMap map[uint64][]common.Uint256, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	proposalSessionMap = make(map[uint64][]common.Uint256)
	for i := uint64(0); i < count; i++ {
		var session uint64
		if session, err = common.ReadUint64(r); err != nil {
			return
		}

		var lenHashes uint64
		if lenHashes, err = common.ReadVarUint(r, 0); err != nil {
			return
		}
		hashes := make([]common.Uint256, 0, lenHashes)
		for i := uint64(0); i < lenHashes; i++ {
			hash := &common.Uint256{}
			if err = hash.Deserialize(r); err != nil {
				return
			}
			hashes = append(hashes, *hash)
		}

		proposalSessionMap[session] = hashes
	}
	return
}

func (p *ProposalKeyFrame) deserializeWithdrawableTransactionsMap(r io.Reader) (
	withdrawableTxsMap map[common.Uint256]common2.OutputInfo, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	withdrawableTxsMap = make(map[common.Uint256]common2.OutputInfo)
	for i := uint64(0); i < count; i++ {
		var hash common.Uint256
		if err = hash.Deserialize(r); err != nil {
			return
		}
		var withdrawInfo common2.OutputInfo
		if err = withdrawInfo.Deserialize(r); err != nil {
			return
		}
	}
	return
}

// Snapshot will create a new ProposalKeyFrame object and deep copy all related data.
func (p *ProposalKeyFrame) Snapshot() *ProposalKeyFrame {
	buf := new(bytes.Buffer)
	p.Serialize(buf)

	state := NewProposalKeyFrame()
	state.Deserialize(buf)
	return state
}

func NewProposalKeyFrame() *ProposalKeyFrame {
	genesisHash, _ := common.Uint256FromHexString("2ce99b16ab5ad0e2027709ad61520fa07017ee639b49154bdee3bec8fadb0d2c")
	return &ProposalKeyFrame{
		Proposals:                      make(map[common.Uint256]*ProposalState),
		ProposalHashes:                 make(map[common.Uint168]ProposalHashSet),
		ProposalSession:                make(map[uint64][]common.Uint256),
		WithdrawableTxInfo:             make(map[common.Uint256]common2.OutputInfo),
		PendingReceivedCustomIDMap:     make(map[string]struct{}),
		RegisteredSideChainPayloadInfo: make(map[uint32]map[common.Uint256]payload.SideChainInfo),
		RegisteredSideChainNames:       []string{"ID"},
		RegisteredMagicNumbers:         []uint32{2018201},
		RegisteredGenesisHashes:        []common.Uint256{*genesisHash},
	}
}

func NewStateKeyFrame() *StateKeyFrame {
	return &StateKeyFrame{
		CodeCIDMap:           make(map[string]common.Uint168),
		DepositHashCIDMap:    make(map[common.Uint168]common.Uint168),
		Candidates:           make(map[common.Uint168]*Candidate),
		HistoryCandidates:    make(map[uint64]map[common.Uint168]*Candidate),
		DepositInfo:          make(map[common.Uint168]*DepositInfo),
		CurrentSession:       0,
		Nicknames:            make(map[string]struct{}),
		Votes:                make(map[string]struct{}),
		DepositOutputs:       make(map[string]common.Fixed64),
		CRCFoundationOutputs: make(map[string]common.Fixed64),
		CRCCommitteeOutputs:  make(map[string]common.Fixed64),
	}
}

// copyCandidateMap copy the CR map's key and value, and return the dst map.
func copyCandidateMap(src map[common.Uint168]*Candidate) (
	dst map[common.Uint168]*Candidate) {
	dst = map[common.Uint168]*Candidate{}
	for k, v := range src {
		p := *v
		dst[k] = &p
	}
	return
}

// copyHistoryCandidateMap copy the CR History map's key and value, and return
// the dst map.
func copyHistoryCandidateMap(src map[uint64]map[common.Uint168]*Candidate) (
	dst map[uint64]map[common.Uint168]*Candidate) {
	dst = map[uint64]map[common.Uint168]*Candidate{}
	for k, v := range src {
		dst[k] = copyCandidateMap(v)
	}
	return
}

// copyHashIDMap copy the map's key and value, and return the dst map.
func copyHashIDMap(src map[common.Uint168]common.Uint168) (
	dst map[common.Uint168]common.Uint168) {
	dst = map[common.Uint168]common.Uint168{}
	for k, v := range src {
		dst[k] = v
	}
	return
}

// copyDepositInfoMap copy the map's key and value, and return the dst map.
func copyDepositInfoMap(src map[common.Uint168]*DepositInfo) (
	dst map[common.Uint168]*DepositInfo) {
	dst = map[common.Uint168]*DepositInfo{}
	for k, v := range src {
		d := *v
		dst[k] = &d
	}
	return
}

// copyDIDAmountMap copy the map's key and value, and return the dst map.
func copyDIDAmountMap(src map[common.Uint168]common.Fixed64) (
	dst map[common.Uint168]common.Fixed64) {
	dst = map[common.Uint168]common.Fixed64{}
	for k, v := range src {
		dst[k] = v
	}
	return
}

// copyCodeAddressMap copy the map's key and value, and return the dst map.
func copyCodeAddressMap(src map[string]common.Uint168) (
	dst map[string]common.Uint168) {
	dst = map[string]common.Uint168{}
	for k, v := range src {
		dst[k] = v
	}
	return
}

func copyFixed64Map(src map[string]common.Fixed64) (dst map[string]common.Fixed64) {
	dst = map[string]common.Fixed64{}
	for k, v := range src {
		p := v
		dst[k] = p
	}
	return
}

func getCRMembers(src map[common.Uint168]*CRMember) []*CRMember {
	dst := make([]*CRMember, 0, len(src))
	for _, v := range src {
		dst = append(dst, v)
	}
	return dst
}

func getCRMembersCopy(src map[common.Uint168]*CRMember) []*CRMember {
	dst := make([]*CRMember, 0, len(src))
	for _, v := range src {
		m := *v
		dst = append(dst, &m)
	}
	return dst
}

func getHistoryMembers(src map[uint64]map[common.Uint168]*CRMember) []*CRMember {
	dst := make([]*CRMember, 0, len(src))
	for _, v := range src {
		for _, m := range v {
			m := *m
			dst = append(dst, &m)
		}
	}
	return dst
}

func getElectedCRMembers(src map[common.Uint168]*CRMember) []*CRMember {
	dst := make([]*CRMember, 0)
	for _, v := range src {
		if v.MemberState == MemberElected {
			m := *v
			dst = append(dst, &m)
		}
	}
	return dst
}

func getImpeachableCRMembers(src map[common.Uint168]*CRMember) []*CRMember {
	dst := make([]*CRMember, 0)
	for _, v := range src {
		if v.MemberState == MemberElected ||
			v.MemberState == MemberInactive || v.MemberState == MemberImpeached {
			m := *v
			dst = append(dst, &m)
		}
	}
	return dst
}

// copyMembersMap copy the CR members map's key and value, and return the dst map.
func copyMembersMap(src map[common.Uint168]*CRMember) (
	dst map[common.Uint168]*CRMember) {
	dst = map[common.Uint168]*CRMember{}
	for k, v := range src {
		p := *v
		dst[k] = &p
	}
	return
}

// copyHistoryMembersMap copy the CR members map's key and value, and return
// the dst map.
func copyHistoryMembersMap(src map[uint64]map[common.Uint168]*CRMember) (
	dst map[uint64]map[common.Uint168]*CRMember) {
	dst = map[uint64]map[common.Uint168]*CRMember{}
	for k, v := range src {
		dst[k] = copyMembersMap(v)
	}
	return
}
