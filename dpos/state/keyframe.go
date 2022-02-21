// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type ConsesusAlgorithm byte

const (
	DPOS ConsesusAlgorithm = 0x00
	POW  ConsesusAlgorithm = 0x01
)

// StateKeyFrame holds necessary state about State
type StateKeyFrame struct {
	NodeOwnerKeys            map[string]string // NodePublicKey as key, OwnerPublicKey as value
	PendingProducers         map[string]*Producer
	ActivityProducers        map[string]*Producer
	DposV2ActivityProducers  map[string]*Producer
	InactiveProducers        map[string]*Producer
	CanceledProducers        map[string]*Producer
	IllegalProducers         map[string]*Producer
	PendingCanceledProducers map[string]*Producer
	DposV2EffectedProducers  map[string]*Producer
	Votes                    map[string]struct{}

	// dpos 2.0
	DetailDPoSV1Votes  map[common.Uint256]payload.DetailedVoteInfo // key: hash of DetailedVoteInfo
	DposV2VoteRights   map[common.Uint168]common.Fixed64           // key: address value: amount
	DposVotes          map[common.Uint168]common.Fixed64           // key: address value: amount
	DposV2Votes        map[common.Uint168]common.Fixed64           // key: address value: amount
	CRVotes            map[common.Uint168]common.Fixed64           // key: address value: amount
	CRImpeachmentVotes map[common.Uint168]common.Fixed64           // key: address value: amount
	CRCProposalVotes   map[common.Uint168]common.Fixed64           // key: address value: amount

	DepositOutputs map[string]common.Fixed64
	//key is addr str value is dposReward
	DposV2RewardInfo         map[string]common.Fixed64
	DposV2RewardClaimingInfo map[string]common.Fixed64
	DposV2RewardClaimedInfo  map[string]common.Fixed64
	Nicknames                map[string]struct{}
	SpecialTxHashes          map[common.Uint256]struct{}
	PreBlockArbiters         map[string]struct{}
	ProducerDepositMap       map[common.Uint168]struct{}
	// dposV2Withdraw info
	WithdrawableTxInfo map[common.Uint256]common2.OutputInfo
	//votes  withdraw
	VotesWithdrawableTxInfo map[common.Uint256]common2.OutputInfo

	EmergencyInactiveArbiters map[string]struct{}
	LastRandomCandidateOwner  string
	VersionStartHeight        uint32
	VersionEndHeight          uint32
	LastRandomCandidateHeight uint32
	DPOSWorkHeight            uint32
	ConsensusAlgorithm        ConsesusAlgorithm
	LastBlockTimestamp        uint32
	NeedRevertToDPOSTX        bool
	NeedNextTurnDPOSInfo      bool
	NoProducers               bool
	NoClaimDPOSNode           bool
	//this height we receieved reverttopow tx and also it is pow work height
	RevertToPOWBlockHeight uint32
	//last irreversible height
	LastIrreversibleHeight uint32
	//record the height our consensus chang from pow into dpos.
	//when it is dpos and before RevertToPOWStartHeight  DPOSStartHeight is height - IrreversibleHeight
	DPOSStartHeight uint32
}

// RewardData defines variables to calculate reward of a round
type RewardData struct {
	OwnerVotesInRound map[common.Uint168]common.Fixed64
	TotalVotesInRound common.Fixed64
}

// SnapshotByHeight takes a SnapshotByHeight of current state and returns the copy.
func (s *StateKeyFrame) snapshot() *StateKeyFrame {
	state := StateKeyFrame{
		NodeOwnerKeys:            make(map[string]string),
		PendingProducers:         make(map[string]*Producer),
		ActivityProducers:        make(map[string]*Producer),
		DposV2ActivityProducers:  make(map[string]*Producer),
		InactiveProducers:        make(map[string]*Producer),
		CanceledProducers:        make(map[string]*Producer),
		IllegalProducers:         make(map[string]*Producer),
		PendingCanceledProducers: make(map[string]*Producer),
		DposV2EffectedProducers:  make(map[string]*Producer),
		Votes:                    make(map[string]struct{}),

		DposV2VoteRights:   make(map[common.Uint168]common.Fixed64),
		DposVotes:          make(map[common.Uint168]common.Fixed64),
		DposV2Votes:        make(map[common.Uint168]common.Fixed64),
		CRVotes:            make(map[common.Uint168]common.Fixed64),
		CRImpeachmentVotes: make(map[common.Uint168]common.Fixed64),
		CRCProposalVotes:   make(map[common.Uint168]common.Fixed64),

		DepositOutputs:           make(map[string]common.Fixed64),
		DposV2RewardInfo:         make(map[string]common.Fixed64),
		DposV2RewardClaimingInfo: make(map[string]common.Fixed64),
		DposV2RewardClaimedInfo:  make(map[string]common.Fixed64),
		WithdrawableTxInfo:       make(map[common.Uint256]common2.OutputInfo),
		VotesWithdrawableTxInfo:  make(map[common.Uint256]common2.OutputInfo),
		Nicknames:                make(map[string]struct{}),
		SpecialTxHashes:          make(map[common.Uint256]struct{}),
		PreBlockArbiters:         make(map[string]struct{}),
		ProducerDepositMap:       make(map[common.Uint168]struct{}),
	}
	state.NodeOwnerKeys = copyStringMap(s.NodeOwnerKeys)
	state.PendingProducers = copyProducerMap(s.PendingProducers)
	state.ActivityProducers = copyProducerMap(s.ActivityProducers)
	state.DposV2ActivityProducers = copyProducerMap(s.DposV2ActivityProducers)
	state.InactiveProducers = copyProducerMap(s.InactiveProducers)
	state.CanceledProducers = copyProducerMap(s.CanceledProducers)
	state.IllegalProducers = copyProducerMap(s.IllegalProducers)
	state.PendingCanceledProducers = copyProducerMap(s.PendingCanceledProducers)
	state.DposV2EffectedProducers = copyProducerMap(s.DposV2EffectedProducers)
	state.Votes = copyStringSet(s.Votes)

	state.DposV2VoteRights = copyProgramHashAmountSet(s.DposV2VoteRights)
	state.DposVotes = copyProgramHashAmountSet(s.DposVotes)
	state.DposV2Votes = copyProgramHashAmountSet(s.DposV2Votes)
	state.CRVotes = copyProgramHashAmountSet(s.CRVotes)
	state.CRImpeachmentVotes = copyProgramHashAmountSet(s.CRImpeachmentVotes)
	state.CRCProposalVotes = copyProgramHashAmountSet(s.CRCProposalVotes)

	state.DepositOutputs = copyFixed64Map(s.DepositOutputs)
	state.DposV2RewardInfo = copyFixed64Map(s.DposV2RewardInfo)
	state.DposV2RewardClaimingInfo = copyFixed64Map(s.DposV2RewardClaimingInfo)
	state.DposV2RewardClaimedInfo = copyFixed64Map(s.DposV2RewardClaimedInfo)
	state.WithdrawableTxInfo = copyWithdrawableTransactionsMap(s.WithdrawableTxInfo)
	state.VotesWithdrawableTxInfo = copyWithdrawableTransactionsMap(s.VotesWithdrawableTxInfo)

	state.Nicknames = copyStringSet(s.Nicknames)
	state.SpecialTxHashes = copyHashSet(s.SpecialTxHashes)
	state.PreBlockArbiters = copyStringSet(s.PreBlockArbiters)
	state.ProducerDepositMap = copyDIDSet(s.ProducerDepositMap)
	return &state
}

func (s *StateKeyFrame) Serialize(w io.Writer) (err error) {

	if err = s.SerializeStringMap(s.NodeOwnerKeys, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.PendingProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.ActivityProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.DposV2ActivityProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.InactiveProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.CanceledProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.IllegalProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.PendingCanceledProducers, w); err != nil {
		return
	}

	if err = s.SerializeProducerMap(s.DposV2EffectedProducers, w); err != nil {
		return
	}

	if err = s.SerializeStringSet(s.Votes, w); err != nil {
		return
	}

	if err = s.SerializeDetailVoteInfoMap(s.DetailDPoSV1Votes, w); err != nil {
		return err
	}

	if err = s.SerializeProgramHashAmountMap(s.DposV2VoteRights, w); err != nil {
		return
	}
	if err = s.SerializeProgramHashAmountMap(s.DposVotes, w); err != nil {
		return
	}
	if err = s.SerializeProgramHashAmountMap(s.DposV2Votes, w); err != nil {
		return
	}
	if err = s.SerializeProgramHashAmountMap(s.CRVotes, w); err != nil {
		return
	}
	if err = s.SerializeProgramHashAmountMap(s.CRImpeachmentVotes, w); err != nil {
		return
	}
	if err = s.SerializeProgramHashAmountMap(s.CRCProposalVotes, w); err != nil {
		return
	}

	if err = s.SerializeFixed64Map(s.DepositOutputs, w); err != nil {
		return
	}

	if err = s.SerializeFixed64Map(s.DposV2RewardInfo, w); err != nil {
		return
	}
	if err = s.SerializeFixed64Map(s.DposV2RewardClaimingInfo, w); err != nil {
		return
	}
	if err = s.SerializeFixed64Map(s.DposV2RewardClaimedInfo, w); err != nil {
		return
	}

	if err = s.serializeWithdrawableTransactionsMap(s.WithdrawableTxInfo, w); err != nil {
		return
	}
	if err = s.serializeWithdrawableTransactionsMap(s.VotesWithdrawableTxInfo, w); err != nil {
		return
	}
	if err = s.SerializeStringSet(s.Nicknames, w); err != nil {
		return
	}

	if err = s.SerializeHashSet(s.SpecialTxHashes, w); err != nil {
		return
	}

	if err = s.SerializeStringSet(s.PreBlockArbiters, w); err != nil {
		return
	}

	if err = s.SerializeDIDSet(s.ProducerDepositMap, w); err != nil {
		return
	}

	if err = s.SerializeStringSet(s.EmergencyInactiveArbiters, w); err != nil {
		return
	}

	if err = common.WriteVarString(w, s.LastRandomCandidateOwner); err != nil {
		return
	}

	if err = common.WriteElements(w, s.VersionStartHeight, s.VersionEndHeight,
		s.LastRandomCandidateHeight, s.DPOSWorkHeight, uint8(s.ConsensusAlgorithm),
		s.LastBlockTimestamp, s.NeedRevertToDPOSTX,
		s.NeedNextTurnDPOSInfo, s.NoProducers, s.NoClaimDPOSNode,
		s.RevertToPOWBlockHeight, s.LastIrreversibleHeight,
		s.DPOSStartHeight); err != nil {
		return err
	}

	return
}

func (s *StateKeyFrame) Deserialize(r io.Reader) (err error) {
	if s.NodeOwnerKeys, err = s.DeserializeStringMap(r); err != nil {
		return
	}

	if s.PendingProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.ActivityProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.DposV2ActivityProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.InactiveProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.CanceledProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.IllegalProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.PendingCanceledProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.DposV2EffectedProducers, err = s.DeserializeProducerMap(r); err != nil {
		return
	}

	if s.Votes, err = s.DeserializeStringSet(r); err != nil {
		return
	}

	if s.DetailDPoSV1Votes, err = s.DeserializeDetailVoteInfoMap(r); err != nil {
		return
	}

	if s.DposV2VoteRights, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}
	if s.DposVotes, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}
	if s.DposV2Votes, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}
	if s.CRVotes, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}
	if s.CRImpeachmentVotes, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}
	if s.CRCProposalVotes, err = s.DeserializeProgramHashAmountMap(r); err != nil {
		return
	}

	if s.DepositOutputs, err = s.DeserializeFixed64Map(r); err != nil {
		return
	}

	if s.DposV2RewardInfo, err = s.DeserializeFixed64Map(r); err != nil {
		return
	}

	if s.DposV2RewardClaimingInfo, err = s.DeserializeFixed64Map(r); err != nil {
		return
	}

	if s.DposV2RewardClaimedInfo, err = s.DeserializeFixed64Map(r); err != nil {
		return
	}

	if s.WithdrawableTxInfo, err = s.deserializeWithdrawableTransactionsMap(r); err != nil {
		return
	}
	if s.VotesWithdrawableTxInfo, err = s.deserializeWithdrawableTransactionsMap(r); err != nil {
		return
	}
	if s.Nicknames, err = s.DeserializeStringSet(r); err != nil {
		return
	}

	if s.SpecialTxHashes, err = s.DeserializeHashSet(r); err != nil {
		return
	}

	if s.PreBlockArbiters, err = s.DeserializeStringSet(r); err != nil {
		return
	}

	if s.ProducerDepositMap, err = s.DeserializeDIDSet(r); err != nil {
		return
	}

	if s.EmergencyInactiveArbiters, err = s.DeserializeStringSet(r); err != nil {
		return
	}

	if s.LastRandomCandidateOwner, err = common.ReadVarString(r); err != nil {
		return
	}

	var consensusAlgorithm uint8
	if err = common.ReadElements(r, &s.VersionStartHeight, &s.VersionEndHeight,
		&s.LastRandomCandidateHeight, &s.DPOSWorkHeight, &consensusAlgorithm,
		&s.LastBlockTimestamp, &s.NeedRevertToDPOSTX,
		&s.NeedNextTurnDPOSInfo, &s.NoProducers, &s.NoClaimDPOSNode,
		&s.RevertToPOWBlockHeight, &s.LastIrreversibleHeight,
		&s.DPOSStartHeight); err != nil {
		return err
	}

	s.ConsensusAlgorithm = ConsesusAlgorithm(consensusAlgorithm)

	return
}

func (s *StateKeyFrame) SerializeHashSet(vmap map[common.Uint256]struct{},
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k := range vmap {
		if err = k.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) DeserializeHashSet(
	r io.Reader) (vmap map[common.Uint256]struct{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint256]struct{})
	for i := uint64(0); i < count; i++ {
		k := common.Uint256{}
		if err = k.Deserialize(r); err != nil {
			return
		}
		vmap[k] = struct{}{}
	}
	return
}

func (s *StateKeyFrame) SerializeFixed64Map(vmap map[string]common.Fixed64,
	w io.Writer) (err error) {
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

func (p *StateKeyFrame) serializeWithdrawableTransactionsMap(
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

func (p *StateKeyFrame) deserializeWithdrawableTransactionsMap(r io.Reader) (
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

func (s *StateKeyFrame) DeserializeFixed64Map(
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

func (s *StateKeyFrame) SerializeStringSet(vmap map[string]struct{},
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k := range vmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) SerializeDetailVoteInfoMap(vmap map[common.Uint256]payload.DetailedVoteInfo,
	w io.Writer) (err error) {
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

func (s *StateKeyFrame) DeserializeDetailVoteInfoMap(
	r io.Reader) (vmap map[common.Uint256]payload.DetailedVoteInfo, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint256]payload.DetailedVoteInfo)
	for i := uint64(0); i < count; i++ {
		var k common.Uint256
		if err = k.Deserialize(r); err != nil {
			return
		}
		var v payload.DetailedVoteInfo
		if err = v.Deserialize(r); err != nil {
			return
		}
		vmap[k] = v
	}
	return
}

func (s *StateKeyFrame) SerializeStringHeightMap(vmap map[string]uint32,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k, v := range vmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
		if err = common.WriteUint32(w, v); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) SerializeProgramHashAmountMap(vmap map[common.Uint168]common.Fixed64,
	w io.Writer) (err error) {
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

func (s *StateKeyFrame) DeserializeStringSet(
	r io.Reader) (vmap map[string]struct{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[string]struct{})
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		vmap[k] = struct{}{}
	}
	return
}

func (s *StateKeyFrame) DeserializeStringHeightMap(
	r io.Reader) (vmap map[string]uint32, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[string]uint32)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		var v uint32
		if v, err = common.ReadUint32(r); err != nil {
			return
		}
		vmap[k] = v
	}
	return
}

func (s *StateKeyFrame) DeserializeProgramHashAmountMap(
	r io.Reader) (vmap map[common.Uint168]common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint168]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
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

func (s *StateKeyFrame) SerializeDIDSet(vmap map[common.Uint168]struct{},
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k := range vmap {
		if err = k.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) DeserializeDIDSet(
	r io.Reader) (vmap map[common.Uint168]struct{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint168]struct{})
	for i := uint64(0); i < count; i++ {
		k := common.Uint168{}
		if err = k.Deserialize(r); err != nil {
			return
		}
		vmap[k] = struct{}{}
	}
	return
}

func (s *StateKeyFrame) SerializeStringMap(smap map[string]string,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(smap))); err != nil {
		return
	}
	for k, v := range smap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}

		if err = common.WriteVarString(w, v); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) DeserializeStringMap(
	r io.Reader) (smap map[string]string, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	smap = make(map[string]string)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		var v string
		if v, err = common.ReadVarString(r); err != nil {
			return
		}
		smap[k] = v
	}
	return
}

func (s *StateKeyFrame) SerializeProducerMap(pmap map[string]*Producer,
	w io.Writer) (err error) {
	if err = common.WriteVarUint(w, uint64(len(pmap))); err != nil {
		return
	}
	for k, v := range pmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (s *StateKeyFrame) DeserializeProducerMap(
	r io.Reader) (pmap map[string]*Producer, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	pmap = make(map[string]*Producer)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		producer := &Producer{}
		if err = producer.Deserialize(r); err != nil {
			return
		}
		pmap[k] = producer
	}
	return
}

func NewStateKeyFrame() *StateKeyFrame {
	info := make(map[string]common.Fixed64)
	return &StateKeyFrame{
		NodeOwnerKeys:             make(map[string]string),
		PendingProducers:          make(map[string]*Producer),
		ActivityProducers:         make(map[string]*Producer),
		DposV2ActivityProducers:   make(map[string]*Producer),
		InactiveProducers:         make(map[string]*Producer),
		CanceledProducers:         make(map[string]*Producer),
		IllegalProducers:          make(map[string]*Producer),
		PendingCanceledProducers:  make(map[string]*Producer),
		DposV2EffectedProducers:   make(map[string]*Producer),
		Votes:                     make(map[string]struct{}),
		DposV2VoteRights:          make(map[common.Uint168]common.Fixed64),
		DposVotes:                 make(map[common.Uint168]common.Fixed64),
		DposV2Votes:               make(map[common.Uint168]common.Fixed64),
		CRVotes:                   make(map[common.Uint168]common.Fixed64),
		CRImpeachmentVotes:        make(map[common.Uint168]common.Fixed64),
		CRCProposalVotes:          make(map[common.Uint168]common.Fixed64),
		DepositOutputs:            make(map[string]common.Fixed64),
		DposV2RewardInfo:          info,
		DposV2RewardClaimingInfo:  make(map[string]common.Fixed64),
		DposV2RewardClaimedInfo:   make(map[string]common.Fixed64),
		WithdrawableTxInfo:        make(map[common.Uint256]common2.OutputInfo),
		VotesWithdrawableTxInfo:   make(map[common.Uint256]common2.OutputInfo),
		Nicknames:                 make(map[string]struct{}),
		SpecialTxHashes:           make(map[common.Uint256]struct{}),
		PreBlockArbiters:          make(map[string]struct{}),
		EmergencyInactiveArbiters: make(map[string]struct{}),
		ProducerDepositMap:        make(map[common.Uint168]struct{}),
		VersionStartHeight:        0,
		VersionEndHeight:          0,
	}
}

func (d *RewardData) Serialize(w io.Writer) error {
	if err := common.WriteUint64(w, uint64(d.TotalVotesInRound)); err != nil {
		return err
	}

	if err := common.WriteVarUint(w,
		uint64(len(d.OwnerVotesInRound))); err != nil {
		return err
	}
	for k, v := range d.OwnerVotesInRound {
		if err := k.Serialize(w); err != nil {
			return err
		}
		if err := common.WriteUint64(w, uint64(v)); err != nil {
			return err
		}
	}
	return nil
}

func (d *RewardData) Deserialize(r io.Reader) (err error) {
	var votes uint64
	if votes, err = common.ReadUint64(r); err != nil {
		return
	}
	d.TotalVotesInRound = common.Fixed64(votes)

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	d.OwnerVotesInRound = make(map[common.Uint168]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		k := common.Uint168{}
		if err = k.Deserialize(r); err != nil {
			return
		}
		var v uint64
		if v, err = common.ReadUint64(r); err != nil {
			return
		}
		d.OwnerVotesInRound[k] = common.Fixed64(v)
	}
	return
}

func NewRewardData() *RewardData {
	return &RewardData{
		OwnerVotesInRound: make(map[common.Uint168]common.Fixed64),
		TotalVotesInRound: 0,
	}
}

// copyProducerMap copy the src map's key, value pairs into dst map.
func copyProducerMap(src map[string]*Producer) (dst map[string]*Producer) {
	dst = map[string]*Producer{}
	for k, v := range src {
		p := *v
		dst[k] = &p
	}
	return
}

func copyStringMap(src map[string]string) (dst map[string]string) {
	dst = map[string]string{}
	for k, v := range src {
		p := v
		dst[k] = p
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

func copyWithdrawableTransactionsMap(src map[common.Uint256]common2.OutputInfo) (dst map[common.Uint256]common2.OutputInfo) {
	dst = map[common.Uint256]common2.OutputInfo{}
	for k, v := range src {
		dst[k] = common2.OutputInfo{
			Recipient: v.Recipient,
			Amount:    v.Amount,
		}
	}
	return
}

func copyStringSet(src map[string]struct{}) (dst map[string]struct{}) {
	dst = map[string]struct{}{}
	for k := range src {
		dst[k] = struct{}{}
	}
	return
}

func copyStringHeightMap(src map[string]uint32) (dst map[string]uint32) {
	dst = make(map[string]uint32)
	for k, v := range src {
		h := v
		dst[k] = h
	}
	return
}

func copyHashSet(src map[common.Uint256]struct{}) (
	dst map[common.Uint256]struct{}) {
	dst = map[common.Uint256]struct{}{}
	for k := range src {
		dst[k] = struct{}{}
	}
	return
}

func copyDIDSet(src map[common.Uint168]struct{}) (
	dst map[common.Uint168]struct{}) {
	dst = map[common.Uint168]struct{}{}
	for k := range src {
		dst[k] = struct{}{}
	}
	return
}

func copyProgramHashAmountSet(src map[common.Uint168]common.Fixed64) (
	dst map[common.Uint168]common.Fixed64) {
	dst = map[common.Uint168]common.Fixed64{}
	for k, v := range src {
		a := v
		dst[k] = a
	}
	return
}

func copyByteList(src []ArbiterMember) (dst []ArbiterMember) {
	for _, v := range src {
		member := v.Clone()
		dst = append(dst, member)
	}
	return
}

func copyReward(src *RewardData) (dst *RewardData) {
	dst = &RewardData{
		OwnerVotesInRound: make(map[common.Uint168]common.Fixed64),
	}
	dst.TotalVotesInRound = src.TotalVotesInRound

	for k, v := range src.OwnerVotesInRound {
		dst.OwnerVotesInRound[k] = v
	}
	return
}

func copyCRCArbitersMap(src map[common.Uint168]ArbiterMember) (dst map[common.Uint168]ArbiterMember) {
	dst = make(map[common.Uint168]ArbiterMember)
	for k, v := range src {
		member := v.Clone()
		dst[k] = member
	}

	return dst
}
