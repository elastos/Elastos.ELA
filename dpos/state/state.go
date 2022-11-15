// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	msg2 "github.com/elastos/Elastos.ELA/dpos/p2p/msg"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/vm"
)

// ProducerState represents the state of a producer.
type ProducerState byte

const (
	// Pending indicates the producer is just registered and didn't get 6
	// confirmations yet.
	Pending ProducerState = iota

	// Active indicates the producer is registered and confirmed by more than
	// 6 blocks.
	Active

	// Inactive indicates the producer has been inactivated for a period which shall
	// be punished and will be activated later.
	Inactive

	// Canceled indicates the producer was canceled.
	Canceled

	// Illegal indicates the producer was found to break the consensus.
	Illegal

	// Returned indicates the producer has canceled and deposit returned.
	Returned
)

// CacheVotesSize indicate the size to cache votes information.
const CacheVotesSize = 6

// IrreversibleHeight defines the max height that the chain be reorganized
const IrreversibleHeight = 6

// producerStateStrings is a array of producer states back to their constant
// names for pretty printing.
var producerStateStrings = []string{"Pending", "Active", "Inactive",
	"Canceled", "Illegal", "Returned"}

func (ps ProducerState) String() string {
	if int(ps) < len(producerStateStrings) {
		return producerStateStrings[ps]
	}
	return fmt.Sprintf("ProducerState-%d", ps)
}

// ProducerIdentity represents the identity of a producer.
type ProducerIdentity byte

const (
	// DPoSV1 indicates the DPoS node is DPoS 1.0 node
	DPoSV1 ProducerIdentity = iota

	// DPoSV2 indicates the DPoS node is DPoS 2.0 node
	DPoSV2

	// DPoSV2 indicates the DPoS node is DPoS 1.0 & DPoS 2.0 node
	DPoSV1V2
)

// producerIdentityStrings is a array of producer identity back to their constant
// names for pretty printing.
var producerIdentityStrings = []string{"DPoSV1", "DPoSV2", "DPoSV1V2"}

func (pi ProducerIdentity) String() string {
	if int(pi) < len(producerIdentityStrings) {
		return producerIdentityStrings[pi]
	}

	return fmt.Sprintf("ProducerIdentity-%d", pi)
}

// Producer holds a producer's info.  It provides read only methods to access
// producer's info.
type Producer struct {
	info                  payload.ProducerInfo
	state                 ProducerState
	identity              ProducerIdentity
	registerHeight        uint32
	cancelHeight          uint32
	inactiveSince         uint32
	activateRequestHeight uint32
	illegalHeight         uint32
	penalty               common.Fixed64
	votes                 common.Fixed64
	dposV2Votes           common.Fixed64

	// the detail information of DPoSV2 votes
	//Uint168 key is voter's sVoteAddr
	//Uint256 key is DetailedVoteInfo's hash
	detailedDPoSV2Votes map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo

	depositAmount                common.Fixed64
	totalAmount                  common.Fixed64
	depositHash                  common.Uint168
	selected                     bool
	randomCandidateInactiveCount uint32
	inactiveCountingHeight       uint32
	lastUpdateInactiveHeight     uint32
	inactiveCount                uint32
	inactiveCountV2              uint32
	workedInRound                bool
}

// Info returns a copy of the origin registered producer info.
func (p *Producer) Info() payload.ProducerInfo {
	return p.info
}

// State returns the producer's state, can be pending, active or canceled.
func (p *Producer) State() ProducerState {
	return p.state
}

// State returns the producer's identity, can be DPoSV1, DPoSV2 or DPoSV1V2.
func (p *Producer) Identity() ProducerIdentity {
	return p.identity
}

// RegisterHeight returns the height when the producer was registered.
func (p *Producer) RegisterHeight() uint32 {
	return p.registerHeight
}

// CancelHeight returns the height when the producer was canceled.
func (p *Producer) CancelHeight() uint32 {
	return p.cancelHeight
}

// Votes returns the votes of the producer.
func (p *Producer) Votes() common.Fixed64 {
	return p.votes
}

// UsedDposV2Votes returns the votes of the dposV2.
func (p *Producer) DposV2Votes() common.Fixed64 {
	return p.dposV2Votes
}

func (p *Producer) NodePublicKey() []byte {
	return p.info.NodePublicKey
}

func (p *Producer) OwnerPublicKey() []byte {
	return p.info.OwnerPublicKey
}

func (p *Producer) Penalty() common.Fixed64 {
	return p.penalty
}

func (p *Producer) InactiveSince() uint32 {
	return p.inactiveSince
}

func (p *Producer) IllegalHeight() uint32 {
	return p.illegalHeight
}

func (p *Producer) ActivateRequestHeight() uint32 {
	return p.activateRequestHeight
}

func (p *Producer) DepositAmount() common.Fixed64 {
	return p.depositAmount
}

func (p *Producer) TotalAmount() common.Fixed64 {
	return p.totalAmount
}

func (p *Producer) AvailableAmount() common.Fixed64 {
	return p.totalAmount - p.depositAmount - p.penalty
}

func (p *Producer) Selected() bool {
	return p.selected
}

func (p *Producer) GetDetailedDPoSV2Votes(stakeAddress common.Uint168,
	referKey common.Uint256) (pl payload.DetailedVoteInfo, err error) {
	votes, ok := p.detailedDPoSV2Votes[stakeAddress]
	if !ok {
		err = errors.New("stake address not found in producer")
	}
	vote, ok := votes[referKey]
	if !ok {
		err = errors.New("referKey not found in producer")
	}
	pl = vote

	return
}

func (p *Producer) GetAllDetailedDPoSV2Votes() map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo {
	return p.detailedDPoSV2Votes
}

func (p *Producer) GetTotalDPoSV2VoteRights() float64 {
	var result float64
	for _, sVoteDetail := range p.detailedDPoSV2Votes {
		var totalN float64
		for _, votes := range sVoteDetail {
			weightF := math.Log10(float64(votes.Info[0].LockTime-votes.BlockHeight) / 7200 * 10)
			N := common.Fixed64(float64(votes.Info[0].Votes) * weightF)
			totalN += float64(N)
		}
		result += totalN
	}

	return result
}

func (p *Producer) SetInfo(i payload.ProducerInfo) {
	p.info = i
}

func (p *Producer) SetState(s ProducerState) {
	p.state = s
}

func (p *Producer) SetRegisterHeight(h uint32) {
	p.registerHeight = h
}

func (p *Producer) SetCancelHeight(h uint32) {
	p.cancelHeight = h
}

func (p *Producer) SetInactiveSince(h uint32) {
	p.inactiveSince = h
}

func (p *Producer) SetActivateRequestHeight(h uint32) {
	p.activateRequestHeight = h
}

func (p *Producer) SetIllegalHeight(h uint32) {
	p.illegalHeight = h
}

func (p *Producer) SetPenalty(v common.Fixed64) {
	p.penalty = v
}

func (p *Producer) SetVotes(v common.Fixed64) {
	p.votes = v
}

func (p *Producer) SetTotalAmount(v common.Fixed64) {
	p.totalAmount = v
}

func (p *Producer) SetDposV2Votes(v common.Fixed64) {
	p.dposV2Votes = v
}

func (p *Producer) SetSelected(s bool) {
	p.selected = s
}

func (p *Producer) Serialize(w io.Writer) error {
	if err := p.info.Serialize(w, payload.ProducerInfoDposV2Version); err != nil {
		return err
	}

	if err := common.WriteUint8(w, uint8(p.state)); err != nil {
		return err
	}

	if err := common.WriteUint8(w, uint8(p.identity)); err != nil {
		return err
	}

	if err := common.WriteUint32(w, p.registerHeight); err != nil {
		return err
	}

	if err := common.WriteUint32(w, p.cancelHeight); err != nil {
		return err
	}

	if err := common.WriteUint32(w, p.inactiveSince); err != nil {
		return err
	}

	if err := common.WriteUint32(w, p.activateRequestHeight); err != nil {
		return err
	}

	if err := common.WriteUint32(w, p.illegalHeight); err != nil {
		return err
	}

	if err := p.penalty.Serialize(w); err != nil {
		return err
	}

	if err := p.votes.Serialize(w); err != nil {
		return err
	}

	if err := p.dposV2Votes.Serialize(w); err != nil {
		return err
	}
	if err := SerializeDetailVoteInfoMap(p.detailedDPoSV2Votes, w); err != nil {
		return err
	}

	if err := p.depositAmount.Serialize(w); err != nil {
		return err
	}

	if err := p.totalAmount.Serialize(w); err != nil {
		return err
	}

	if err := p.depositHash.Serialize(w); err != nil {
		return err
	}

	return common.WriteElements(w, p.selected, p.randomCandidateInactiveCount,
		p.inactiveCountingHeight, p.lastUpdateInactiveHeight, p.inactiveCount, p.inactiveCountV2, p.workedInRound)
}

func SerializeDetailVoteInfoMap(
	vmap map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo,
	w io.Writer) (err error) {

	if err := common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return err
	}
	for k, v := range vmap {
		if err := k.Serialize(w); err != nil {
			return err
		}
		if err := common.WriteVarUint(w, uint64(len(v))); err != nil {
			return err
		}
		for k2, v2 := range v {
			if err := k2.Serialize(w); err != nil {
				return err
			}
			if err := v2.Serialize(w); err != nil {
				return err
			}
		}
	}

	return
}

func (p *Producer) Deserialize(r io.Reader) (err error) {
	if err = p.info.Deserialize(r, payload.ProducerInfoDposV2Version); err != nil {
		return
	}

	var state uint8
	if state, err = common.ReadUint8(r); err != nil {
		return
	}
	p.state = ProducerState(state)

	var identity uint8
	if identity, err = common.ReadUint8(r); err != nil {
		return
	}
	p.identity = ProducerIdentity(identity)

	if p.registerHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if p.cancelHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if p.inactiveSince, err = common.ReadUint32(r); err != nil {
		return
	}

	if p.activateRequestHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if p.illegalHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if err = p.penalty.Deserialize(r); err != nil {
		return
	}

	if err = p.votes.Deserialize(r); err != nil {
		return
	}

	if err = p.dposV2Votes.Deserialize(r); err != nil {
		return
	}

	voteInfoMap, err := DeserializeDetailVoteInfoMap(r)
	if err != nil {
		return err
	}
	p.detailedDPoSV2Votes = voteInfoMap

	if err = p.depositAmount.Deserialize(r); err != nil {
		return
	}

	if err = p.totalAmount.Deserialize(r); err != nil {
		return
	}

	if err = p.depositHash.Deserialize(r); err != nil {
		return
	}

	return common.ReadElements(r, &p.selected, &p.randomCandidateInactiveCount,
		&p.inactiveCountingHeight, &p.lastUpdateInactiveHeight, &p.inactiveCount, &p.inactiveCountV2, &p.workedInRound)
}

func DeserializeDetailVoteInfoMap(
	r io.Reader) (vmap map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}

		var count2 uint64
		if count2, err = common.ReadVarUint(r, 0); err != nil {
			return
		}
		vmap2 := make(map[common.Uint256]payload.DetailedVoteInfo)
		for j := uint64(0); j < count2; j++ {
			var k2 common.Uint256
			if err = k2.Deserialize(r); err != nil {
				return
			}

			var v2 payload.DetailedVoteInfo
			if err = v2.Deserialize(r); err != nil {
				return
			}

			vmap2[k2] = v2
		}

		vmap[k] = vmap2
	}
	return
}

const (
	// maxHistoryCapacity indicates the maximum capacity of change History.
	maxHistoryCapacity = 720

	// ActivateDuration is about how long we should activate from pending or
	// inactive state
	ActivateDuration = 6
)

// State is a memory database storing DPOS producers state, like pending
// producers active producers and their votes.
type State struct {
	*StateKeyFrame

	// GetArbiters defines methods about get current arbiters
	GetArbiters                   func() []*ArbiterInfo
	getCurrentCRMembers           func() []*state.CRMember
	getNextCRMembers              func() []*state.CRMember
	getCRMember                   func(key string) *state.CRMember
	updateCRInactivePenalty       func(cid common.Uint168, height uint32)
	revertUpdateCRInactivePenalty func(cid common.Uint168, height uint32)
	isInElectionPeriod            func() bool
	GetProducerDepositAmount      func(programHash common.Uint168) (
		common.Fixed64, error)
	GetTxReference func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error)
	tryUpdateCRMemberInactivity func(did common.Uint168, needReset bool, height uint32)
	tryRevertCRMemberInactivity func(did common.Uint168, oriState state.MemberState,
		oriInactiveCountingHeight uint32, height uint32)
	tryUpdateCRMemberIllegal func(did common.Uint168, height uint32, illegalPenalty common.Fixed64)
	tryRevertCRMemberIllegal func(did common.Uint168, oriState state.MemberState, height uint32, illegalPenalty common.Fixed64)

	ChainParams *config.Configuration
	mtx         sync.RWMutex
	History     *utils.History

	getHeight                           func() uint32
	isCurrent                           func() bool
	broadcast                           func(msg p2p.Message)
	appendToTxpool                      func(transaction interfaces.Transaction) elaerr.ELAError
	createDposV2RealWithdrawTransaction func(withdrawTransactionHashes []common.Uint256,
		outputs []*common2.OutputInfo) (interfaces.Transaction, error)
	createVotesRealWithdrawTransaction func(withdrawTransactionHashes []common.Uint256,
		outputs []*common2.OutputInfo) (interfaces.Transaction, error)
}

func (s *State) DPoSV2Started() bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.dposV2Started()
}

func (s *State) dposV2Started() bool {
	return s.DPoSV2ActiveHeight != math.MaxUint32
}

func (s *State) isDposV2Active() bool {
	log.Errorf("isDposV2Active len(a.DposV2EffectedProducers) %d  GeneralArbiters %d", len(s.DposV2EffectedProducers),
		s.ChainParams.DPoSConfiguration.NormalArbitratorsCount)
	return len(s.DposV2EffectedProducers) >= s.ChainParams.DPoSConfiguration.NormalArbitratorsCount*3/2
}

func (s *State) GetRealWithdrawTransactions() map[common.Uint256]common2.OutputInfo {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.StateKeyFrame.WithdrawableTxInfo
}

func (s *State) GetVotesWithdrawableTxInfo() map[common.Uint256]common2.OutputInfo {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.StateKeyFrame.VotesWithdrawableTxInfo
}

// getProducerKey returns the producer's owner public key string, whether the
// given public key is the producer's node public key or owner public key.
func (s *State) getProducerKey(publicKey []byte) string {
	key := hex.EncodeToString(publicKey)

	// If the given public key is node public key, get the producer's owner
	// public key.
	if owner, ok := s.NodeOwnerKeys[key]; ok {
		return owner
	}

	if owner, ok := s.LastCRNodeOwnerKeys[key]; ok {
		return owner
	}

	if owner, ok := s.CurrentCRNodeOwnerKeys[key]; ok {
		return owner
	}

	if owner, ok := s.NextCRNodeOwnerKeys[key]; ok {
		return owner
	}

	return key
}

// getProducer returns a producer with the producer's node public key or it's
// owner public key, if no matches return nil.
func (s *State) getProducer(publicKey []byte) *Producer {
	key := s.getProducerKey(publicKey)
	return s.getProducerByOwnerPublicKey(key)
}

// getDPoSV2Producer returns a DPoSV2 producer with the producer's node public
// key or it's owner public key, if no matches return nil.
func (s *State) getDPoSV2Producer(publicKey []byte) *Producer {
	key := s.getProducerKey(publicKey)
	produer := s.getProducerByOwnerPublicKey(key)
	if produer != nil && produer.info.StakeUntil == 0 {
		return nil
	}

	return produer
}

// getProducer returns a producer with the producer's owner public key,
// if no matches return nil.
func (s *State) getProducerByOwnerPublicKey(key string) *Producer {
	if producer, ok := s.ActivityProducers[key]; ok {
		return producer
	}
	if producer, ok := s.CanceledProducers[key]; ok {
		return producer
	}
	if producer, ok := s.IllegalProducers[key]; ok {
		return producer
	}
	if producer, ok := s.PendingProducers[key]; ok {
		return producer
	}
	if producer, ok := s.InactiveProducers[key]; ok {
		return producer
	}
	return nil
}

// updateProducerInfo updates the producer's info with value compare, any change
// will be updated.
func (s *State) updateProducerInfo(origin *payload.ProducerInfo, update *payload.ProducerInfo) {
	producer := s.getProducer(origin.OwnerPublicKey)

	// compare and update node nickname.
	if origin.NickName != update.NickName {
		delete(s.Nicknames, origin.NickName)
		s.Nicknames[update.NickName] = struct{}{}
	}

	// compare and update node public key, we only query pending and active node
	// because canceled and illegal node can not be updated.
	if !bytes.Equal(origin.NodePublicKey, update.NodePublicKey) {
		oldKey := hex.EncodeToString(origin.NodePublicKey)
		newKey := hex.EncodeToString(update.NodePublicKey)
		delete(s.NodeOwnerKeys, oldKey)
		s.NodeOwnerKeys[newKey] = hex.EncodeToString(origin.OwnerPublicKey)
	}

	producer.info = *update
}

func (s *State) ExistProducerByDepositHash(programHash common.Uint168) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	_, ok := s.ProducerDepositMap[programHash]
	return ok
}

// GetProducer returns a producer with the producer's node public key or it's
// owner public key including canceled and illegal producers.  If no matches
// return nil.
func (s *State) GetProducer(publicKey []byte) *Producer {
	s.mtx.RLock()
	producer := s.getProducer(publicKey)
	s.mtx.RUnlock()
	return producer
}

// GetProducers returns all producers including pending and active producers (no
// canceled and illegal producers).
func (s *State) GetProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.PendingProducers)+
		len(s.ActivityProducers))
	for _, producer := range s.PendingProducers {
		producers = append(producers, producer)
	}
	for _, producer := range s.ActivityProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetProducers returns all producers including pending and active producers (no
// canceled and illegal producers).
func (s *State) GetDposV2Producers() []*Producer {
	s.mtx.RLock()
	producers := s.getDposV2Producers()
	s.mtx.RUnlock()
	return producers
}

func (s *State) getDposV2Producers() []*Producer {
	dposV2Producers := make([]*Producer, 0)
	allProducer := s.getAllProducers()
	for _, producer := range allProducer {
		if producer.info.StakeUntil != 0 {
			dposV2Producers = append(dposV2Producers, producer)
		}
	}
	return dposV2Producers
}

func (s *State) GetAllProducersPublicKey() []string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	var nodePublicKeys []string
	for nodePK, _ := range s.NodeOwnerKeys {
		nodePublicKeys = append(nodePublicKeys, nodePK)
	}
	for nodePK, _ := range s.CurrentCRNodeOwnerKeys {
		nodePublicKeys = append(nodePublicKeys, nodePK)
	}
	for nodePK, _ := range s.NextCRNodeOwnerKeys {
		nodePublicKeys = append(nodePublicKeys, nodePK)
	}
	for _, nodePK := range s.ChainParams.DPoSConfiguration.CRCArbiters {
		nodePublicKeys = append(nodePublicKeys, nodePK)
	}
	return nodePublicKeys
}

// GetAllProducers returns all producers including pending, active, canceled, illegal and inactive producers.
func (s *State) GetAllProducers() []Producer {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.getAllProducersByCopy()
}

func (s *State) GetDetailedDPoSV2Votes(stakeProgramHash *common.Uint168) []payload.DetailedVoteInfo {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	var result []payload.DetailedVoteInfo
	ps := s.getAllProducers()
	for _, p := range ps {
		if len(p.detailedDPoSV2Votes[*stakeProgramHash]) == 0 {
			continue
		}
		for _, detailedVoteInfo := range p.detailedDPoSV2Votes[*stakeProgramHash] {
			result = append(result, detailedVoteInfo)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ReferKey().Compare(result[j].ReferKey()) >= 0
	})
	return result
}

func (s *State) getAllProducersByCopy() []Producer {
	producers := make([]Producer, 0, len(s.PendingProducers)+
		len(s.ActivityProducers))
	for _, producer := range s.PendingProducers {
		producers = append(producers, *producer)
	}
	for _, producer := range s.ActivityProducers {
		producers = append(producers, *producer)
	}
	for _, producer := range s.InactiveProducers {
		producers = append(producers, *producer)
	}
	for _, producer := range s.CanceledProducers {
		producers = append(producers, *producer)
	}
	for _, producer := range s.IllegalProducers {
		producers = append(producers, *producer)
	}
	return producers
}

func (s *State) getAllProducers() []*Producer {
	producers := make([]*Producer, 0, len(s.PendingProducers)+
		len(s.ActivityProducers))
	for _, producer := range s.PendingProducers {
		producers = append(producers, producer)
	}
	for _, producer := range s.ActivityProducers {
		producers = append(producers, producer)
	}
	for _, producer := range s.InactiveProducers {
		producers = append(producers, producer)
	}
	for _, producer := range s.CanceledProducers {
		producers = append(producers, producer)
	}
	for _, producer := range s.IllegalProducers {
		producers = append(producers, producer)
	}
	return producers
}

func (s *State) getAllNodePublicKey() map[string]struct{} {

	nodePublicKeyMap := make(map[string]struct{})

	for _, producer := range s.PendingProducers {
		strNodePublicKey := common.BytesToHexString(producer.info.NodePublicKey)
		nodePublicKeyMap[strNodePublicKey] = struct{}{}
	}
	for _, producer := range s.ActivityProducers {
		strNodePublicKey := common.BytesToHexString(producer.info.NodePublicKey)
		nodePublicKeyMap[strNodePublicKey] = struct{}{}
	}
	for _, producer := range s.InactiveProducers {
		strNodePublicKey := common.BytesToHexString(producer.info.NodePublicKey)
		nodePublicKeyMap[strNodePublicKey] = struct{}{}
	}
	for _, producer := range s.CanceledProducers {
		strNodePublicKey := common.BytesToHexString(producer.info.NodePublicKey)
		nodePublicKeyMap[strNodePublicKey] = struct{}{}
	}
	for _, producer := range s.IllegalProducers {
		strNodePublicKey := common.BytesToHexString(producer.info.NodePublicKey)
		nodePublicKeyMap[strNodePublicKey] = struct{}{}
	}
	return nodePublicKeyMap
}

// GetPendingProducers returns all producers that in pending state.
func (s *State) GetPendingProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.PendingProducers))
	for _, producer := range s.PendingProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetActiveProducers returns all producers that in active state.
func (s *State) GetActiveProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.ActivityProducers))
	for _, producer := range s.ActivityProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetActiveProducers returns all producers that in active state.
func (s *State) GetActiveV1Producers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.ActivityProducers))
	for _, producer := range s.ActivityProducers {
		if producer.identity != DPoSV1 && producer.identity != DPoSV1V2 {
			continue
		}
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetActivityV2Producers returns all DPoS V2 producers that in active state.
func (s *State) GetActivityV2Producers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.ActivityProducers))
	for _, producer := range s.ActivityProducers {
		if producer.identity != DPoSV2 && producer.identity != DPoSV1V2 {
			continue
		}
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetVotedProducers returns all producers that in active state with votes.
func (s *State) GetVotedProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.ActivityProducers))
	for _, producer := range s.ActivityProducers {
		// limit arbiters can only be producers who have votes
		if producer.Votes() > 0 {
			producers = append(producers, producer)
		}
	}
	s.mtx.RUnlock()
	return producers
}

// GetDposV2ActiveProducers returns all producers that in active state with votes.
func (s *State) GetDposV2ActiveProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.ActivityProducers))
	for _, producer := range s.ActivityProducers {
		// limit arbiters can only be producers who have effective dposV2 votes
		voteRights := producer.GetTotalDPoSV2VoteRights()
		if voteRights > float64(s.ChainParams.DPoSV2EffectiveVotes) {
			producers = append(producers, producer)
		}
	}
	s.mtx.RUnlock()
	return producers
}

// GetCanceledProducers returns all producers that in cancel state.
func (s *State) GetCanceledProducers() []*Producer {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.getCanceledProducers()
}

// getCanceledProducers returns all producers that in cancel state.
func (s *State) getCanceledProducers() []*Producer {
	producers := make([]*Producer, 0, len(s.CanceledProducers))
	for _, producer := range s.CanceledProducers {
		if producer.state == Canceled {
			producers = append(producers, producer)
		}
	}
	return producers
}

// GetPendingCanceledProducers returns all producers that in pending canceled state.
func (s *State) GetPendingCanceledProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.PendingCanceledProducers))
	for _, producer := range s.PendingCanceledProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetReturnedDepositProducers returns producers that in returned deposit state.
func (s *State) GetReturnedDepositProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.CanceledProducers))
	for _, producer := range s.CanceledProducers {
		if producer.state == Returned {
			producers = append(producers, producer)
		}
	}
	s.mtx.RUnlock()
	return producers
}

// GetIllegalProducers returns all illegal producers.
func (s *State) GetIllegalProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.IllegalProducers))
	for _, producer := range s.IllegalProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// GetIllegalProducers returns all inactive producers.
func (s *State) GetInactiveProducers() []*Producer {
	s.mtx.RLock()
	producers := make([]*Producer, 0, len(s.InactiveProducers))
	for _, producer := range s.InactiveProducers {
		producers = append(producers, producer)
	}
	s.mtx.RUnlock()
	return producers
}

// IsPendingProducer returns if a producer is in pending list according to the
// public key.
func (s *State) IsPendingProducer(publicKey []byte) bool {
	s.mtx.RLock()
	_, ok := s.PendingProducers[s.getProducerKey(publicKey)]
	s.mtx.RUnlock()
	return ok
}

func (s *State) GetConsensusAlgorithm() ConsesusAlgorithm {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.ConsensusAlgorithm
}

// IsActiveProducer returns if a producer is in activate list according to the
// public key.
func (s *State) IsActiveProducer(publicKey []byte) bool {
	s.mtx.RLock()
	_, ok := s.ActivityProducers[s.getProducerKey(publicKey)]
	s.mtx.RUnlock()
	return ok
}

// IsInactiveProducer returns if a producer is in inactivate list according to
// the public key.
func (s *State) IsInactiveProducer(publicKey []byte) bool {
	s.mtx.RLock()
	ok := s.isInactiveProducer(publicKey)
	s.mtx.RUnlock()
	return ok
}

func (s *State) isInactiveProducer(publicKey []byte) bool {
	_, ok := s.InactiveProducers[s.getProducerKey(publicKey)]
	return ok
}

// IsCanceledProducer returns if a producer is in canceled list according to the
// public key.
func (s *State) IsCanceledProducer(publicKey []byte) bool {
	s.mtx.RLock()
	_, ok := s.CanceledProducers[s.getProducerKey(publicKey)]
	s.mtx.RUnlock()
	return ok
}

// IsIllegalProducer returns if a producer is in illegal list according to the
// public key.
func (s *State) IsIllegalProducer(publicKey []byte) bool {
	s.mtx.RLock()
	_, ok := s.IllegalProducers[s.getProducerKey(publicKey)]
	s.mtx.RUnlock()
	return ok
}

// IsAbleToRecoverFromInactiveMode returns if most of the emergency arbiters have activated
// and able to work again
func (s *State) IsAbleToRecoverFromInactiveMode() bool {
	activatedNum := 0

	s.mtx.RLock()
	totalNum := len(s.EmergencyInactiveArbiters)
	for k := range s.EmergencyInactiveArbiters {
		if _, ok := s.InactiveProducers[k]; !ok {
			activatedNum++
		}
	}
	s.mtx.RUnlock()

	return totalNum == 0 || float64(activatedNum)/float64(totalNum) >
		MajoritySignRatioNumerator/MajoritySignRatioDenominator
}

// IsAbleToRecoverFromInactiveMode returns if there are enough active arbiters
func (s *State) IsAbleToRecoverFromUnderstaffedState() bool {
	s.mtx.RLock()
	result := len(s.ActivityProducers) >= s.ChainParams.DPoSConfiguration.NormalArbitratorsCount
	s.mtx.RUnlock()
	return result
}

// LeaveEmergency will reset EmergencyInactiveArbiters variable
func (s *State) LeaveEmergency(history *utils.History, height uint32) {
	s.mtx.Lock()
	oriArbiters := s.EmergencyInactiveArbiters
	history.Append(height, func() {
		s.EmergencyInactiveArbiters = map[string]struct{}{}
	}, func() {
		s.EmergencyInactiveArbiters = oriArbiters
	})
	s.mtx.Unlock()
}

// NicknameExists returns if a nickname is exists.
func (s *State) NicknameExists(nickname string) bool {
	s.mtx.RLock()
	_, ok := s.Nicknames[nickname]
	s.mtx.RUnlock()
	return ok
}

// ProducerExists returns if a producer is exists by it's node public key or
// owner public key.
func (s *State) ProducerExists(publicKey []byte) bool {
	s.mtx.RLock()
	producer := s.getProducer(publicKey)
	s.mtx.RUnlock()
	return producer != nil
}

// ProducerExists returns if a producer is exists by it's owner public key.
func (s *State) ProducerOwnerPublicKeyExists(publicKey []byte) bool {
	s.mtx.RLock()
	key := hex.EncodeToString(publicKey)
	producer := s.getProducerByOwnerPublicKey(key)
	s.mtx.RUnlock()
	return producer != nil
}

func (s *State) GetProducerByOwnerPublicKey(publicKey []byte) *Producer {
	s.mtx.RLock()
	key := hex.EncodeToString(publicKey)
	producer := s.getProducerByOwnerPublicKey(key)
	s.mtx.RUnlock()
	return producer
}

// ProducerOrCRNodePublicKeyExists returns if a producer is exists by it's node public key.
func (s *State) ProducerOrCRNodePublicKeyExists(publicKey []byte) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	key := hex.EncodeToString(publicKey)
	if _, ok := s.NodeOwnerKeys[key]; ok {
		return ok
	}
	if _, ok := s.CurrentCRNodeOwnerKeys[key]; ok {
		return ok
	}
	_, ok := s.NextCRNodeOwnerKeys[key]

	return ok
}

// NextCRNodePublicKeyExists returns if a CR producer is exists by it's node public key.
func (s *State) ProducerAndCurrentCRNodePublicKeyExists(publicKey []byte) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	key := hex.EncodeToString(publicKey)
	if _, ok := s.NodeOwnerKeys[key]; ok {
		return ok
	}
	_, ok := s.CurrentCRNodeOwnerKeys[key]

	return ok
}

// ProducerOrCRNodePublicKeyExists returns if a producer is exists by it's node public key.
func (s *State) ProducerAndNextCRNodePublicKeyExists(publicKey []byte) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	key := hex.EncodeToString(publicKey)
	if _, ok := s.NodeOwnerKeys[key]; ok {
		return ok
	}
	_, ok := s.NextCRNodeOwnerKeys[key]

	return ok
}

// SpecialTxExists returns if a special tx (typically means illegal and
// inactive tx) is exists by it's hash
func (s *State) SpecialTxExists(tx interfaces.Transaction) bool {
	illegalData, ok := tx.Payload().(payload.DPOSIllegalData)
	if !ok {
		log.Error("special tx payload cast failed, tx:", common.ToReversedString(tx.Hash()))
		return false
	}

	hash := illegalData.Hash()
	s.mtx.RLock()
	_, ok = s.SpecialTxHashes[hash]
	s.mtx.RUnlock()
	return ok
}

// IsDPOSTransaction returns if a transaction will change the producers and
// votes state.
func (s *State) IsDPOSTransaction(tx interfaces.Transaction) bool {
	switch tx.TxType() {
	// Transactions will changes the producers state.
	case common2.RegisterProducer, common2.UpdateProducer, common2.CancelProducer,
		common2.ActivateProducer, common2.IllegalProposalEvidence,
		common2.IllegalVoteEvidence, common2.IllegalBlockEvidence,
		common2.IllegalSidechainEvidence, common2.InactiveArbitrators,
		common2.ReturnDepositCoin:
		return true

	// Transactions will change the producer votes state.
	case common2.TransferAsset:
		if tx.Version() >= common2.TxVersion09 {
			// Votes to producers.
			for _, output := range tx.Outputs() {
				if output.Type != common2.OTVote {
					continue
				}
				p, _ := output.Payload.(*outputpayload.VoteOutput)
				if p.Version == outputpayload.VoteProducerVersion {
					return true
				} else {
					for _, content := range p.Contents {
						if content.VoteType == outputpayload.Delegate {
							return true
						}
					}
				}
			}
		}
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()
	// Cancel votes.
	for _, input := range tx.Inputs() {
		_, ok := s.Votes[input.ReferKey()]
		if ok {
			return true
		}
	}

	return false
}

// ProcessBlock takes a block and it's confirm to update producers state and
// votes accordingly.
func (s *State) ProcessBlock(block *types.Block, confirm *payload.Confirm, dutyIndex int) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	//s.tryInitProducerAssetAmounts(block.Height)
	s.processTransactions(block.Transactions, block.Height)
	s.ProcessVoteStatisticsBlock(block)
	s.updateProducersDepositCoin(block.Height)
	s.recordLastBlockTime(block)
	s.tryRevertToPOWByStateOfCRMember(block.Height)
	s.tryUpdateLastIrreversibleHeight(block.Height)

	if confirm != nil {
		if block.Height > s.DPoSV2ActiveHeight {
			s.countArbitratorsInactivityV3(block.Height, confirm, dutyIndex)
		} else if block.Height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
			s.countArbitratorsInactivityV2(block.Height, confirm)
		} else if block.Height >= s.ChainParams.CRConfiguration.CRClaimDPOSNodeStartHeight {
			s.countArbitratorsInactivityV1(block.Height, confirm)
		} else {
			s.countArbitratorsInactivityV0(block.Height, confirm)
		}
	}

	if block.Height >= s.ChainParams.DPoSV2StartHeight &&
		len(s.WithdrawableTxInfo) != 0 {
		s.createDposV2ClaimRewardRealWithdrawTransaction(block.Height)
	}

	if block.Height >= s.ChainParams.DPoSV2StartHeight &&
		len(s.VotesWithdrawableTxInfo) != 0 {
		s.createRealWithdrawTransaction(block.Height)
	}

	// todo remove me
	if block.Height > s.ChainParams.DPoSV2StartHeight {
		msg2.SetPayloadVersion(msg2.DPoSV2Version)
	}

	// Commit changes here if no errors found.
	s.History.Commit(block.Height)
}

func (s *State) createRealWithdrawTransaction(height uint32) {
	if s.createVotesRealWithdrawTransaction != nil && height == s.getHeight() {
		retVoteswithdrawTxHashes := make([]common.Uint256, 0)
		ouputs := make([]*common2.OutputInfo, 0)
		for k, v := range s.VotesWithdrawableTxInfo {
			retVoteswithdrawTxHashes = append(retVoteswithdrawTxHashes, k)
			outputInfo := v
			ouputs = append(ouputs, &outputInfo)
		}
		tx, err := s.createVotesRealWithdrawTransaction(retVoteswithdrawTxHashes, ouputs)
		if err != nil {
			log.Error("create real return votes tx failed:", err.Error())
			return
		}

		log.Info("create real return votes transaction:", tx.Hash())
		if s.isCurrent != nil && s.broadcast != nil && s.
			appendToTxpool != nil {
			go func() {
				if s.isCurrent() {
					if err := s.appendToTxpool(tx); err == nil {
						s.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create real return votes tx "+
							"append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return
}

func (s *State) createDposV2ClaimRewardRealWithdrawTransaction(height uint32) {
	if s.createDposV2RealWithdrawTransaction != nil && height == s.getHeight() {
		withdrawTransactionHahses := make([]common.Uint256, 0)
		ouputs := make([]*common2.OutputInfo, 0)
		for k, v := range s.WithdrawableTxInfo {
			withdrawTransactionHahses = append(withdrawTransactionHahses, k)
			outputInfo := v
			ouputs = append(ouputs, &outputInfo)
		}
		tx, err := s.createDposV2RealWithdrawTransaction(withdrawTransactionHahses, ouputs)
		if err != nil {
			log.Error("create dposv2 real withdraw tx failed:", err.Error())
			return
		}

		log.Info("create dposv2 real withdraw transaction:", tx.Hash())
		if s.isCurrent != nil && s.broadcast != nil && s.
			appendToTxpool != nil {
			go func() {
				if s.isCurrent() {
					if err := s.appendToTxpool(tx); err == nil {
						s.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create dposv2 real withdraw transaction "+
							"append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return
}

type StateFuncsConfig struct {
	GetHeight                           func() uint32
	CreateDposV2RealWithdrawTransaction func(withdrawTransactionHashes []common.Uint256,
		outpus []*common2.OutputInfo) (interfaces.Transaction, error)
	CreateVotesRealWithdrawTransaction func(withdrawTransactionHashes []common.Uint256,
		outpus []*common2.OutputInfo) (interfaces.Transaction, error)
	IsCurrent      func() bool
	Broadcast      func(msg p2p.Message)
	AppendToTxpool func(transaction interfaces.Transaction) elaerr.ELAError
}

func (c *State) RegisterFuncitons(cfg *StateFuncsConfig) {
	c.createDposV2RealWithdrawTransaction = cfg.CreateDposV2RealWithdrawTransaction
	c.createVotesRealWithdrawTransaction = cfg.CreateVotesRealWithdrawTransaction
	c.isCurrent = cfg.IsCurrent
	c.broadcast = cfg.Broadcast
	c.appendToTxpool = cfg.AppendToTxpool
	c.getHeight = cfg.GetHeight
}

func (s *State) tryRevertToPOWByStateOfCRMember(height uint32) {
	if !s.isInElectionPeriod() || s.NoClaimDPOSNode ||
		s.ConsensusAlgorithm == POW {
		return
	}
	for _, m := range s.getCurrentCRMembers() {
		if m.MemberState == state.MemberElected {
			return
		}
	}
	s.History.Append(height, func() {
		s.NoClaimDPOSNode = true
	}, func() {
		s.NoClaimDPOSNode = false
	})
	log.Info("[tryRevertToPOWByStateOfCRMember] found that no CR member"+
		" claimed DPoS node at height:", height)
}

// record timestamp of last block
func (s *State) recordLastBlockTime(block *types.Block) {
	oriLastBlockTime := s.LastBlockTimestamp
	s.History.Append(block.Height, func() {
		s.LastBlockTimestamp = block.Timestamp
	}, func() {
		s.LastBlockTimestamp = oriLastBlockTime
	})
}

// update producers deposit coin
func (s *State) updateProducersDepositCoin(height uint32) {
	updateDepositCoin := func(producer *Producer) {
		oriDepositAmount := producer.depositAmount
		s.History.Append(height, func() {
			producer.depositAmount -= state.MinDepositAmount
		}, func() {
			producer.depositAmount = oriDepositAmount
		})
	}
	updateDPoSV1V2DepositCoin := func(producer *Producer) {
		oriDepositAmount := producer.depositAmount
		s.History.Append(height, func() {
			//when we are in v2 satge, v1v2producer depositAmount need -3000
			producer.depositAmount -= (state.MinDepositAmount - state.MinDPoSV2DepositAmount)
		}, func() {
			producer.depositAmount = oriDepositAmount
		})
	}
	updateDepositCoinAndState := func(producer *Producer) {
		oriState := producer.state
		key := hex.EncodeToString(producer.OwnerPublicKey())
		oriDepositAmount := producer.depositAmount
		s.History.Append(height, func() {
			producer.depositAmount -= state.MinDepositAmount
			producer.state = Canceled
			s.CanceledProducers[key] = producer
			switch oriState {
			case Pending:
				delete(s.PendingProducers, key)
				s.PendingCanceledProducers[key] = producer
			case Active:
				delete(s.ActivityProducers, key)
			case Inactive:
				delete(s.InactiveProducers, key)
			case Illegal:
				delete(s.IllegalProducers, key)
			}
			delete(s.Nicknames, producer.info.NickName)
		}, func() {
			producer.depositAmount = oriDepositAmount
			producer.state = oriState
			delete(s.CanceledProducers, key)
			switch oriState {
			case Pending:
				s.PendingProducers[key] = producer
				delete(s.PendingCanceledProducers, key)
			case Active:
				s.ActivityProducers[key] = producer
			case Inactive:
				s.InactiveProducers[key] = producer
			case Illegal:
				s.IllegalProducers[key] = producer

			}
			s.Nicknames[producer.info.NickName] = struct{}{}

		})
	}

	canceledProducers := s.getCanceledProducers()
	for _, producer := range canceledProducers {
		if height-producer.CancelHeight() == s.ChainParams.CRConfiguration.DepositLockupBlocks {
			updateDepositCoin(producer)
		}
	}

	//when we are in DPoSV2ActiveHeight update dpos 1.0 producer DepositCoin and State
	if height == s.DPoSV2ActiveHeight {
		ps := s.getAllProducers()
		for _, producer := range ps {
			if producer.identity == DPoSV1 && producer.state != Returned && producer.state != Canceled {
				updateDepositCoinAndState(producer)
			} else if producer.identity == DPoSV1V2 && producer.state != Returned && producer.state != Canceled {
				updateDPoSV1V2DepositCoin(producer)
			}

		}
	}

}

// ProcessVoteStatisticsBlock deal with block with vote statistics error.
func (s *State) ProcessVoteStatisticsBlock(block *types.Block) {
	if block.Height == s.ChainParams.VoteStatisticsHeight {
		s.processTransactions(block.Transactions, block.Height)
	}
}

// processTransactions takes the transactions and the height when they have been
// packed into a block.  Then loop through the transactions to update producers
// state and votes according to transactions content.
func (s *State) processTransactions(txs []interfaces.Transaction, height uint32) {

	for _, tx := range txs {
		s.processTransaction(tx, height)
	}

	// Check if any pending producers has got 6 confirms, set them to activate.
	activateProducerFromPending := func(key string, producer *Producer) {
		s.History.Append(height, func() {
			producer.state = Active
			s.ActivityProducers[key] = producer
			delete(s.PendingProducers, key)
		}, func() {
			producer.state = Pending
			s.PendingProducers[key] = producer
			delete(s.ActivityProducers, key)
		})
	}

	// Check if any pending inactive producers has got 6 confirms,
	// then set them to activate.
	activateProducerFromInactive := func(key string, producer *Producer) {
		s.History.Append(height, func() {
			producer.state = Active
			s.ActivityProducers[key] = producer
			delete(s.InactiveProducers, key)
		}, func() {
			producer.state = Inactive
			s.InactiveProducers[key] = producer
			delete(s.ActivityProducers, key)
		})
	}

	cancelDposV2AndDposV1V2Producer := func(key string, producer *Producer) {
		oriState := producer.state
		oriDepositAmount := producer.depositAmount
		s.History.Append(height, func() {
			producer.state = Canceled
			producer.depositAmount -= state.MinDPoSV2DepositAmount
			s.CanceledProducers[key] = producer
			switch oriState {
			case Active:
				delete(s.ActivityProducers, key)
			case Inactive:
				delete(s.InactiveProducers, key)
			case Illegal:
				delete(s.IllegalProducers, key)
			}
			delete(s.Nicknames, producer.info.NickName)
		}, func() {
			producer.state = oriState
			producer.depositAmount = oriDepositAmount
			switch oriState {
			case Active:
				s.ActivityProducers[key] = producer
			case Inactive:
				s.InactiveProducers[key] = producer
			case Illegal:
				s.IllegalProducers[key] = producer
			}
			delete(s.CanceledProducers, key)
			s.Nicknames[producer.info.NickName] = struct{}{}
		})
	}

	cleanExpiredDposV2Votes := func(key common.Uint256, stakeAddress common.Uint168, detailVoteInfo payload.DetailedVoteInfo, producer *Producer) {

		for _, i := range detailVoteInfo.Info {
			info := i
			s.History.Append(height, func() {
				s.UsedDposV2Votes[stakeAddress] -= info.Votes
			}, func() {
				s.UsedDposV2Votes[stakeAddress] += info.Votes
			})

			voteRights := producer.GetTotalDPoSV2VoteRights()

			s.History.Append(height, func() {
				delete(producer.detailedDPoSV2Votes[stakeAddress], key)
				producer.dposV2Votes -= info.Votes
				if voteRights < float64(s.ChainParams.DPoSV2EffectiveVotes) {
					delete(s.DposV2EffectedProducers, hex.EncodeToString(producer.OwnerPublicKey()))
				}
			}, func() {
				if producer.detailedDPoSV2Votes == nil {
					producer.detailedDPoSV2Votes = make(map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo)
				}
				if _, ok := producer.detailedDPoSV2Votes[stakeAddress]; !ok {
					producer.detailedDPoSV2Votes[stakeAddress] = make(map[common.Uint256]payload.DetailedVoteInfo)
				}
				producer.detailedDPoSV2Votes[stakeAddress][key] = detailVoteInfo
				producer.dposV2Votes += info.Votes
				voteRights := producer.GetTotalDPoSV2VoteRights()
				if voteRights >= float64(s.ChainParams.DPoSV2EffectiveVotes) {
					s.DposV2EffectedProducers[hex.EncodeToString(producer.OwnerPublicKey())] = producer
				}
			})
		}
	}

	// Check if any pending illegal producers has got 6 confirms,
	// then set them to activate.
	activateProducerFromIllegal := func(key string, producer *Producer) {
		s.History.Append(height, func() {
			producer.state = Active
			s.ActivityProducers[key] = producer
			delete(s.IllegalProducers, key)
		}, func() {
			producer.state = Illegal
			s.IllegalProducers[key] = producer
			delete(s.ActivityProducers, key)
		})
	}

	if len(s.PendingProducers) > 0 {
		for key, producer := range s.PendingProducers {
			if height-producer.registerHeight+1 >= ActivateDuration {
				activateProducerFromPending(key, producer)
			}
		}
	}
	if len(s.InactiveProducers) > 0 {
		for key, producer := range s.InactiveProducers {
			if height > producer.activateRequestHeight &&
				height-producer.activateRequestHeight+1 >= ActivateDuration {
				activateProducerFromInactive(key, producer)
			}
		}
	}

	if ps := s.getDposV2Producers(); len(ps) > 0 {
		for _, p := range ps {
			cp := p
			if cp.info.StakeUntil < height {
				key := hex.EncodeToString(cp.info.OwnerPublicKey)
				if cp.state != Returned && cp.state != Canceled &&
					(cp.identity == DPoSV2 || (cp.identity == DPoSV1V2 && height > s.DPoSV2ActiveHeight)) {
					cancelDposV2AndDposV1V2Producer(key, cp)
				}
			}
			for stake, detail := range p.detailedDPoSV2Votes {
				for refer, info := range detail {
					ci := info
					crefer := refer
					cstake := stake
					if info.Info[0].LockTime < height {
						cleanExpiredDposV2Votes(crefer, cstake, ci, cp)
					}
				}
			}
		}
	}

	if height >= s.ChainParams.EnableActivateIllegalHeight &&
		len(s.IllegalProducers) > 0 {
		for key, producer := range s.IllegalProducers {
			if height > producer.activateRequestHeight &&
				height-producer.activateRequestHeight+1 >= ActivateDuration {
				activateProducerFromIllegal(key, producer)
			}
		}
	}

	// Check if any pending producers has got 6 confirms, set them to activate.
	revertToDPOS := func() {
		s.History.Append(height, func() {
			s.ConsensusAlgorithm = DPOS
		}, func() {
			s.ConsensusAlgorithm = POW
		})
	}
	if s.DPOSWorkHeight != 0 {
		if height >= s.DPOSWorkHeight && s.ConsensusAlgorithm == POW {
			revertToDPOS()
		}
	}

}

// processTransaction take a transaction and the height it has been packed into
// a block, then update producers state and votes according to the transaction
// content.
func (s *State) processTransaction(tx interfaces.Transaction, height uint32) {
	switch tx.TxType() {
	case common2.RegisterProducer:
		s.registerProducer(tx, height)

	case common2.UpdateProducer:
		s.updateProducer(tx.Payload().(*payload.ProducerInfo), height)

	case common2.CancelProducer:
		s.cancelProducer(tx.Payload().(*payload.ProcessProducer), height)

	case common2.ActivateProducer:
		s.activateProducer(tx.Payload().(*payload.ActivateProducer), height)

	case common2.TransferAsset:
		s.processVotes(tx, height)

	case common2.ExchangeVotes:
		s.processStake(tx, height)

	case common2.Voting:
		s.processVoting(tx, height)

	case common2.IllegalProposalEvidence, common2.IllegalVoteEvidence,
		common2.IllegalBlockEvidence, common2.IllegalSidechainEvidence:
		s.processIllegalEvidence(tx.Payload(), height)

		payloadHash, err := tx.GetSpecialTxHash()
		if err != nil {
			log.Error(err.Error())
			return
		}
		s.recordSpecialTx(payloadHash, height)

	case common2.InactiveArbitrators:
		s.processEmergencyInactiveArbitrators(
			tx.Payload().(*payload.InactiveArbitrators), height)
		payloadHash, err := tx.GetSpecialTxHash()
		if err != nil {
			log.Error(err.Error())
			return
		}
		s.recordSpecialTx(payloadHash, height)

	case common2.ReturnDepositCoin:
		s.returnDeposit(tx, height)

	case common2.UpdateVersion:
		s.updateVersion(tx, height)

	case common2.NextTurnDPOSInfo:
		s.processNextTurnDPOSInfo(tx, height)

	case common2.CRCouncilMemberClaimNode:
		s.processCRCouncilMemberClaimNode(tx, height)

	case common2.RevertToPOW:
		s.processRevertToPOW(tx, height)

	case common2.RevertToDPOS:
		s.processRevertToDPOS(tx.Payload().(*payload.RevertToDPOS), height)

	case common2.DposV2ClaimReward:
		s.processDposV2ClaimReward(tx, height)

	case common2.ReturnVotes:
		s.processReturnVotes(tx, height)

	case common2.DposV2ClaimRewardRealWithdraw:
		s.processDposV2ClaimRewardRealWithdraw(tx, height)

	case common2.VotesRealWithdraw:
		s.processRetVotesRewardRealWithdraw(tx, height)

	case common2.CreateNFT:
		s.processCreateNFT(tx, height)

	}

	if tx.TxType() != common2.RegisterProducer {
		s.processDeposit(tx, height)
	}
	s.processCancelVotes(tx, height)
}

// registerProducer handles the register producer transaction.
func (s *State) registerProducer(tx interfaces.Transaction, height uint32) {
	info := tx.Payload().(*payload.ProducerInfo)
	nickname := info.NickName
	nodeKey := hex.EncodeToString(info.NodePublicKey)
	ownerKey := hex.EncodeToString(info.OwnerPublicKey)
	// ignore error here because this converting process has been ensured in
	// the context check already
	programHash, _ := contract.PublicKeyToDepositProgramHash(info.
		OwnerPublicKey)

	amount := common.Fixed64(0)
	depositOutputs := make(map[string]common.Fixed64)
	for i, output := range tx.Outputs() {
		if output.ProgramHash.IsEqual(*programHash) {
			amount += output.Value
			op := common2.NewOutPoint(tx.Hash(), uint16(i))
			depositOutputs[op.ReferKey()] = output.Value
		}
	}

	depositAmount := common.Fixed64(0)
	var identity ProducerIdentity
	if info.StakeUntil != 0 {
		depositAmount = state.MinDPoSV2DepositAmount
		identity = DPoSV2
	} else {
		depositAmount = state.MinDepositAmount
		identity = DPoSV1
	}

	producer := Producer{
		info:                         *info,
		identity:                     identity,
		registerHeight:               height,
		votes:                        0,
		dposV2Votes:                  0,
		inactiveSince:                0,
		inactiveCount:                0,
		inactiveCountV2:              0,
		randomCandidateInactiveCount: 0,
		penalty:                      common.Fixed64(0),
		activateRequestHeight:        math.MaxUint32,
		depositAmount:                depositAmount,
		totalAmount:                  amount,
		depositHash:                  *programHash,
	}

	s.History.Append(height, func() {
		s.Nicknames[nickname] = struct{}{}
		s.NodeOwnerKeys[nodeKey] = ownerKey
		s.PendingProducers[ownerKey] = &producer
		s.ProducerDepositMap[*programHash] = struct{}{}
		for k, v := range depositOutputs {
			s.DepositOutputs[k] = v
		}
	}, func() {
		delete(s.Nicknames, nickname)
		delete(s.NodeOwnerKeys, nodeKey)
		delete(s.PendingProducers, ownerKey)
		delete(s.ProducerDepositMap, *programHash)
		for k := range depositOutputs {
			delete(s.DepositOutputs, k)
		}
	})
}

// updateProducer handles the update producer transaction.
func (s *State) updateProducer(info *payload.ProducerInfo, height uint32) {
	producer := s.getProducer(info.OwnerPublicKey)
	originProducerIdentity := producer.identity
	producerInfo := producer.info
	s.History.Append(height, func() {
		s.updateProducerInfo(&producerInfo, info)

		// update identity
		if info.StakeUntil != 0 {
			switch producer.identity {
			case DPoSV1:
				producer.identity = DPoSV1V2
			case DPoSV2, DPoSV1V2:
				// do nothing
			}
		}

	}, func() {
		s.updateProducerInfo(info, &producerInfo)

		// rollback identity
		producer.identity = originProducerIdentity
	})
}

// cancelProducer handles the cancel producer transaction.
func (s *State) cancelProducer(payload *payload.ProcessProducer, height uint32) {
	key := hex.EncodeToString(payload.OwnerPublicKey)
	producer := s.getProducer(payload.OwnerPublicKey)
	oriState := producer.state
	s.History.Append(height, func() {
		producer.state = Canceled
		producer.cancelHeight = height
		s.CanceledProducers[key] = producer
		switch oriState {
		case Pending:
			delete(s.PendingProducers, key)
			s.PendingCanceledProducers[key] = producer
		case Active:
			delete(s.ActivityProducers, key)
		case Inactive:
			delete(s.InactiveProducers, key)
		}
		delete(s.Nicknames, producer.info.NickName)
	}, func() {
		producer.cancelHeight = 0
		delete(s.CanceledProducers, key)
		producer.state = oriState
		switch oriState {
		case Pending:
			s.PendingProducers[key] = producer
			delete(s.PendingCanceledProducers, key)
		case Active:
			s.ActivityProducers[key] = producer
		case Inactive:
			s.InactiveProducers[key] = producer
		}
		s.Nicknames[producer.info.NickName] = struct{}{}
	})
}

// activateProducer handles the activate producer transaction.
func (s *State) activateProducer(p *payload.ActivateProducer, height uint32) {
	producer := s.getProducer(p.NodePublicKey)
	if producer == nil {
		return
	}
	s.History.Append(height, func() {
		producer.activateRequestHeight = height
	}, func() {
		producer.activateRequestHeight = math.MaxUint32
	})
}

// processVotes takes a transaction, if the transaction including any vote
// inputs or outputs, validate and update producers votes.
func (s *State) processVotes(tx interfaces.Transaction, height uint32) {
	if tx.Version() >= common2.TxVersion09 {
		// Votes to producers.
		for i, output := range tx.Outputs() {
			if output.Type != common2.OTVote && output.Type != common2.OTDposV2Vote {
				continue
			}
			p, _ := output.Payload.(*outputpayload.VoteOutput)
			if p.Version == outputpayload.VoteProducerVersion {
				op := common2.NewOutPoint(tx.Hash(), uint16(i))
				s.History.Append(height, func() {
					s.Votes[op.ReferKey()] = struct{}{}
				}, func() {
					delete(s.Votes, op.ReferKey())
				})
				s.processVoteOutput(output, height)
			} else {
				var exist bool
				for _, content := range p.Contents {
					if content.VoteType == outputpayload.Delegate {
						exist = true
						break
					}
				}
				if exist {
					op := common2.NewOutPoint(tx.Hash(), uint16(i))
					s.History.Append(height, func() {
						s.Votes[op.ReferKey()] = struct{}{}
					}, func() {
						delete(s.Votes, op.ReferKey())
					})
					s.processVoteOutput(output, height)
				}
			}
		}
	}
}

// processNewVotes takes a transaction, if the transaction including any votes
// validate and update producers votes.
func (s *State) processStake(tx interfaces.Transaction, height uint32) {
	ot := tx.Outputs()[0]
	pld := ot.Payload.(*outputpayload.ExchangeVotesOutput)
	s.History.Append(height, func() {
		s.DposV2VoteRights[pld.StakeAddress] += ot.Value
	}, func() {
		s.DposV2VoteRights[pld.StakeAddress] -= ot.Value
	})
}

// processNewVotes takes a transaction, if the transaction including any votes
// validate and update producers votes.
func (s *State) processVoting(tx interfaces.Transaction, height uint32) {
	switch tx.PayloadVersion() {
	case payload.VoteVersion:
		s.processVotingContent(tx, height)
	case payload.RenewalVoteVersion:
		s.processRenewalVotingContent(tx, height)
	}
}

//
//func (s *State) processCancelVoting(tx interfaces.Transaction, height uint32) {
//	// get stake address(program hash)
//	code := tx.Programs()[0].Code
//	ct, _ := contract.CreateStakeContractByCode(code)
//	stakeAddress := ct.ToProgramHash()
//
//	pld := tx.Payload().(*payload.CancelVotes)
//	for _, k := range pld.ReferKeys {
//		key := k
//		detailVoteInfo, ok := s.DetailDPoSV1Votes[key]
//		if ok && detailVoteInfo.VoteType == outputpayload.Delegate {
//			var maxVotes common.Fixed64
//			for _, i := range detailVoteInfo.Info {
//				info := i
//				if info.Votes > maxVotes {
//					maxVotes = info.Votes
//				}
//
//				producer := s.getProducer(info.Candidate)
//				if producer == nil {
//					continue
//				}
//				s.History.Append(height, func() {
//					producer.votes -= info.Votes
//				}, func() {
//					producer.votes += info.Votes
//				})
//			}
//
//			s.History.Append(height, func() {
//				s.UsedDPoSVotes[*stakeAddress] -= maxVotes
//			}, func() {
//				s.UsedDPoSVotes[*stakeAddress] += maxVotes
//			})
//
//			s.History.Append(height, func() {
//				delete(s.DetailDPoSV1Votes, key)
//			}, func() {
//				s.DetailDPoSV1Votes[key] = detailVoteInfo
//			})
//		}
//	}
//}

func (s *State) processVotingContent(tx interfaces.Transaction, height uint32) {
	// get stake address(program hash)
	code := tx.Programs()[0].Code
	signType, _ := crypto.GetScriptType(code)
	var prefixType byte
	if signType == vm.CHECKSIG {
		prefixType = byte(contract.PrefixStandard)
	} else if signType == vm.CHECKMULTISIG {
		prefixType = byte(contract.PrefixMultiSig)
	}

	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	pld := tx.Payload().(*payload.Voting)
	for _, cont := range pld.Contents {
		content := cont
		switch content.VoteType {
		case outputpayload.Delegate:
			var maxVotes common.Fixed64
			for _, vote := range content.VotesInfo {
				if maxVotes < vote.Votes {
					maxVotes = vote.Votes
				}
			}

			oriUsedDPoSVotes := s.UsedDposVotes[*stakeAddress]
			s.History.Append(height, func() {
				s.UsedDposVotes[*stakeAddress] = content.VotesInfo
			}, func() {
				s.UsedDposVotes[*stakeAddress] = oriUsedDPoSVotes
			})

			for _, v := range oriUsedDPoSVotes {
				vt := v
				producer := s.getProducer(v.Candidate)
				if producer == nil {
					continue
				}
				s.History.Append(height, func() {
					producer.votes -= vt.Votes
				}, func() {
					producer.votes += vt.Votes
				})
			}

			for _, v := range content.VotesInfo {
				vt := v
				producer := s.getProducer(v.Candidate)
				if producer == nil {
					continue
				}
				s.History.Append(height, func() {
					producer.votes += vt.Votes
				}, func() {
					producer.votes -= vt.Votes
				})
			}

		case outputpayload.DposV2:
			var totalVotes common.Fixed64
			for _, vote := range content.VotesInfo {
				totalVotes += vote.Votes
			}
			s.History.Append(height, func() {
				s.UsedDposV2Votes[*stakeAddress] += totalVotes
			}, func() {
				s.UsedDposV2Votes[*stakeAddress] -= totalVotes
			})

			for _, v := range content.VotesInfo {
				producer := s.getDPoSV2Producer(v.Candidate)
				if producer == nil {
					continue
				}
				voteInfo := v
				dvi := payload.DetailedVoteInfo{
					StakeProgramHash: *stakeAddress,
					TransactionHash:  tx.Hash(),
					BlockHeight:      height,
					PayloadVersion:   tx.PayloadVersion(),
					VoteType:         content.VoteType,
					Info:             []payload.VotesWithLockTime{voteInfo},
					PrefixType:       prefixType,
				}
				s.History.Append(height, func() {
					if producer.detailedDPoSV2Votes == nil {
						producer.detailedDPoSV2Votes = make(map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo)
					}
					if _, ok := producer.detailedDPoSV2Votes[*stakeAddress]; !ok {
						producer.detailedDPoSV2Votes[*stakeAddress] = make(map[common.Uint256]payload.DetailedVoteInfo)
					}
					producer.detailedDPoSV2Votes[*stakeAddress][dvi.ReferKey()] = dvi
					producer.dposV2Votes += voteInfo.Votes

					voteRights := producer.GetTotalDPoSV2VoteRights()
					if voteRights >= float64(s.ChainParams.DPoSV2EffectiveVotes) {
						s.DposV2EffectedProducers[hex.EncodeToString(producer.OwnerPublicKey())] = producer
					}
				}, func() {
					delete(producer.detailedDPoSV2Votes[*stakeAddress], dvi.ReferKey())
					producer.dposV2Votes -= voteInfo.Votes

					voteRights := producer.GetTotalDPoSV2VoteRights()
					if voteRights < float64(s.ChainParams.DPoSV2EffectiveVotes) {
						delete(s.DposV2EffectedProducers, hex.EncodeToString(producer.OwnerPublicKey()))
					}
				})
			}
		}
	}
}

func (s *State) processRenewalVotingContent(tx interfaces.Transaction, height uint32) {
	// get stake address
	code := tx.Programs()[0].Code
	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	signType, _ := crypto.GetScriptType(code)
	var prefixType byte
	if signType == vm.CHECKSIG {
		prefixType = byte(contract.PrefixStandard)
	} else if signType == vm.CHECKMULTISIG {
		prefixType = byte(contract.PrefixMultiSig)
	}

	pld := tx.Payload().(*payload.Voting)
	for _, cont := range pld.RenewalContents {
		content := cont
		// get producer and update the votes
		producer := s.getDPoSV2Producer(content.VotesInfo.Candidate)
		if producer == nil {
			log.Info("can not find producer ", hex.EncodeToString(content.VotesInfo.Candidate))
			continue
		}
		voteInfo, _ := producer.GetDetailedDPoSV2Votes(*stakeAddress, content.ReferKey)

		// record all new votes information
		detailVoteInfo := payload.DetailedVoteInfo{
			StakeProgramHash: *stakeAddress,
			TransactionHash:  tx.Hash(),
			BlockHeight:      voteInfo.BlockHeight,
			PayloadVersion:   voteInfo.PayloadVersion,
			VoteType:         outputpayload.DposV2,
			Info:             []payload.VotesWithLockTime{content.VotesInfo},
			PrefixType:       prefixType,
		}

		referKey := detailVoteInfo.ReferKey()
		s.History.Append(height, func() {
			producer.detailedDPoSV2Votes[*stakeAddress][referKey] = detailVoteInfo
			delete(producer.detailedDPoSV2Votes[*stakeAddress], content.ReferKey)
		}, func() {
			delete(producer.detailedDPoSV2Votes[*stakeAddress], referKey)
			producer.detailedDPoSV2Votes[*stakeAddress][content.ReferKey] = voteInfo
		})
	}
}

// processDeposit takes a transaction output with deposit program hash.
func (s *State) processDeposit(tx interfaces.Transaction, height uint32) {
	for i, output := range tx.Outputs() {
		if contract.GetPrefixType(output.ProgramHash) ==
			contract.PrefixDeposit {
			if s.addProducerAssert(output, height) {
				op := common2.NewOutPoint(tx.Hash(), uint16(i))
				s.DepositOutputs[op.ReferKey()] = output.Value
			}
		}
	}
}

// getProducerByDepositHash will try to get producer with specified program
// hash, note the producer state should be pending active or inactive.
func (s *State) getProducerByDepositHash(hash common.Uint168) *Producer {
	for _, producer := range s.PendingProducers {
		if producer.depositHash.IsEqual(hash) {
			return producer
		}
	}
	for _, producer := range s.ActivityProducers {
		if producer.depositHash.IsEqual(hash) {
			return producer
		}
	}
	for _, producer := range s.InactiveProducers {
		if producer.depositHash.IsEqual(hash) {
			return producer
		}
	}
	for _, producer := range s.CanceledProducers {
		if producer.depositHash.IsEqual(hash) {
			return producer
		}
	}
	for _, producer := range s.IllegalProducers {
		if producer.depositHash.IsEqual(hash) {
			return producer
		}
	}
	return nil
}

// addProducerAssert will plus deposit amount for producers referenced in
// program hash of transaction output.
func (s *State) addProducerAssert(output *common2.Output, height uint32) bool {
	if producer := s.getProducerByDepositHash(output.ProgramHash); producer != nil {
		s.History.Append(height, func() {
			producer.totalAmount += output.Value
		}, func() {
			producer.totalAmount -= output.Value
		})
		return true
	}
	return false
}

// processCancelVotes takes a transaction output with vote payload.
func (s *State) processCancelVotes(tx interfaces.Transaction, height uint32) {
	var exist bool
	for _, input := range tx.Inputs() {
		referKey := input.ReferKey()
		if _, ok := s.Votes[referKey]; ok {
			exist = true
		}
	}
	if !exist {
		return
	}

	references, err := s.GetTxReference(tx)
	if err != nil {
		log.Errorf("get tx reference failed, tx hash:%s", common.ToReversedString(tx.Hash()))
		return
	}
	for _, input := range tx.Inputs() {
		referKey := input.ReferKey()
		_, ok := s.Votes[referKey]
		if ok {
			out := references[input]
			s.processVoteCancel(&out, height)
		}
	}
}

// processVoteOutput takes a transaction output with vote payload.
func (s *State) processVoteOutput(output *common2.Output, height uint32) {
	countByGross := func(producer *Producer) {
		s.History.Append(height, func() {
			producer.votes += output.Value
		}, func() {
			producer.votes -= output.Value
		})
	}

	countByVote := func(producer *Producer, vote common.Fixed64) {
		s.History.Append(height, func() {
			producer.votes += vote
		}, func() {
			producer.votes -= vote
		})
	}

	p := output.Payload.(*outputpayload.VoteOutput)
	for _, vote := range p.Contents {
		for _, cv := range vote.CandidateVotes {
			producer := s.getProducer(cv.Candidate)
			if producer == nil {
				continue
			}

			switch vote.VoteType {
			case outputpayload.Delegate:
				if p.Version == outputpayload.VoteProducerVersion {
					countByGross(producer)
				} else {
					v := cv.Votes
					countByVote(producer, v)
				}
			}
		}
	}
}

// processVoteCancel takes a previous vote output and decrease producers votes.
func (s *State) processVoteCancel(output *common2.Output, height uint32) {
	subtractByGross := func(producer *Producer) {
		s.History.Append(height, func() {
			producer.votes -= output.Value
		}, func() {
			producer.votes += output.Value
		})
	}

	subtractByVote := func(producer *Producer, vote common.Fixed64) {
		s.History.Append(height, func() {
			producer.votes -= vote
		}, func() {
			producer.votes += vote
		})
	}

	p := output.Payload.(*outputpayload.VoteOutput)
	for _, vote := range p.Contents {
		for _, cv := range vote.CandidateVotes {
			producer := s.getProducer(cv.Candidate)
			if producer == nil {
				continue
			}
			switch vote.VoteType {
			case outputpayload.Delegate:
				if p.Version == outputpayload.VoteProducerVersion {
					subtractByGross(producer)
				} else {
					v := cv.Votes
					subtractByVote(producer, v)
				}
			}
		}
	}
}

// ReturnDeposit change producer state to ReturnedDeposit with lock
func (s *State) ReturnDeposit(tx interfaces.Transaction, height uint32) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.returnDeposit(tx, height)
}

// returnDeposit change producer state to ReturnedDeposit
func (s *State) returnDeposit(tx interfaces.Transaction, height uint32) {
	var inputValue common.Fixed64
	for _, input := range tx.Inputs() {
		inputValue += s.DepositOutputs[input.ReferKey()]
	}

	for _, program := range tx.Programs() {
		pk := program.Code[1 : len(program.Code)-1]
		if producer := s.getProducer(pk); producer != nil {

			// check deposit coin
			hash, err := contract.PublicKeyToDepositProgramHash(producer.info.OwnerPublicKey)
			if err != nil {
				log.Error("owner public key to deposit program hash: failed")
				return
			}

			var changeValue common.Fixed64
			var outputValue common.Fixed64
			for _, output := range tx.Outputs() {
				if output.ProgramHash.IsEqual(*hash) {
					changeValue += output.Value
				} else {
					outputValue += output.Value
				}
			}

			returnAction := func(producer *Producer) {
				s.History.Append(height, func() {
					producer.totalAmount -= inputValue
					if producer.state == Canceled &&
						producer.totalAmount+changeValue-producer.penalty <=
							s.ChainParams.MinTransactionFee {
						producer.state = Returned
					}
				}, func() {
					producer.totalAmount += inputValue
					producer.state = Canceled
				})
			}

			returnAction(producer)
		}
	}
}

// processNextTurnDPOSInfo change NeedNextTurnDposInfo  status
func (s *State) processNextTurnDPOSInfo(tx interfaces.Transaction, height uint32) {
	_, ok := tx.Payload().(*payload.NextTurnDPOSInfo)
	if !ok {
		return
	}
	log.Warnf("processNextTurnDPOSInfo tx: %s, %d", common.ToReversedString(tx.Hash()), height)
	oriNeedNextTurnDposInfo := s.NeedNextTurnDPOSInfo
	s.History.Append(height, func() {
		s.NeedNextTurnDPOSInfo = false
	}, func() {
		s.NeedNextTurnDPOSInfo = oriNeedNextTurnDposInfo
	})
}

func (s *State) getCRMembersOwnerPublicKey(CRCommitteeDID common.Uint168) []byte {
	if s.getCurrentCRMembers != nil && s.getNextCRMembers != nil {
		for _, cr := range s.getCurrentCRMembers() {
			if cr.Info.DID.IsEqual(CRCommitteeDID) {
				return cr.Info.Code[1 : len(cr.Info.Code)-1]
			}
		}
		for _, cr := range s.getNextCRMembers() {
			if cr.Info.DID.IsEqual(CRCommitteeDID) {
				return cr.Info.Code[1 : len(cr.Info.Code)-1]
			}
		}
	}
	return nil
}

func (s *State) getCurrentCRNodePublicKeyStr(strOwnerPublicKey string) string {
	for nodePubKey, nodeOwnerPubKey := range s.CurrentCRNodeOwnerKeys {
		if strOwnerPublicKey == nodeOwnerPubKey {
			return nodePubKey
		}
	}
	return ""
}

func (s *State) getNextCRNodePublicKeyStr(strOwnerPublicKey string) string {
	for nodePubKey, nodeOwnerPubKey := range s.NextCRNodeOwnerKeys {
		if strOwnerPublicKey == nodeOwnerPubKey {
			return nodePubKey
		}
	}
	return ""
}

func (s *State) processCRCouncilMemberClaimNode(tx interfaces.Transaction, height uint32) {
	claimNodePayload := tx.Payload().(*payload.CRCouncilMemberClaimNode)
	strNewNodePublicKey := common.BytesToHexString(claimNodePayload.NodePublicKey)

	ownerPublicKey := s.getCRMembersOwnerPublicKey(claimNodePayload.CRCouncilCommitteeDID)
	if ownerPublicKey == nil {
		log.Error("processCRCouncilMemberClaimNode cr member is not exist")
		return
	}
	strOwnerPubkey := common.BytesToHexString(ownerPublicKey)

	switch tx.PayloadVersion() {
	case payload.CurrentCRClaimDPoSNodeVersion:
		strOldNodePublicKey := s.getCurrentCRNodePublicKeyStr(strOwnerPubkey)
		s.History.Append(height, func() {
			s.CurrentCRNodeOwnerKeys[strNewNodePublicKey] = strOwnerPubkey
			if strOldNodePublicKey != "" {
				delete(s.CurrentCRNodeOwnerKeys, strOldNodePublicKey)
			}
		}, func() {
			delete(s.CurrentCRNodeOwnerKeys, strNewNodePublicKey)
			if strOldNodePublicKey != "" {
				s.CurrentCRNodeOwnerKeys[strOldNodePublicKey] = strOwnerPubkey
			}
		})
	case payload.NextCRClaimDPoSNodeVersion:
		strOldNodePublicKey := s.getNextCRNodePublicKeyStr(strOwnerPubkey)
		s.History.Append(height, func() {
			s.NextCRNodeOwnerKeys[strNewNodePublicKey] = strOwnerPubkey
			if strOldNodePublicKey != "" {
				delete(s.NextCRNodeOwnerKeys, strOldNodePublicKey)
			}
		}, func() {
			delete(s.NextCRNodeOwnerKeys, strNewNodePublicKey)
			if strOldNodePublicKey != "" {
				s.NextCRNodeOwnerKeys[strOldNodePublicKey] = strOwnerPubkey
			}
		})
	}
}

func (s *State) processRevertToPOW(tx interfaces.Transaction, height uint32) {
	oriNoProducers := s.NoProducers
	oriNoClaimDPOSNode := s.NoClaimDPOSNode
	oriDPOSWorkHeight := s.DPOSWorkHeight
	oriRevertToPOWBlockHeight := s.RevertToPOWBlockHeight
	s.History.Append(height, func() {
		s.ConsensusAlgorithm = POW
		s.NoProducers = false
		s.NoClaimDPOSNode = false
		s.DPOSWorkHeight = 0
		s.RevertToPOWBlockHeight = height
	}, func() {
		s.ConsensusAlgorithm = DPOS
		s.NoProducers = oriNoProducers
		s.NoClaimDPOSNode = oriNoClaimDPOSNode
		s.DPOSWorkHeight = oriDPOSWorkHeight
		s.RevertToPOWBlockHeight = oriRevertToPOWBlockHeight

	})

	pld := tx.Payload().(*payload.RevertToPOW)
	log.Infof("[processRevertToPOW], revert to POW at height:%d, "+
		"revert type:%s", height, pld.Type.String())
}

// updateVersion record the update period during that inactive Arbiters
// will not need to pay the penalty
func (s *State) updateVersion(tx interfaces.Transaction, height uint32) {
	p, ok := tx.Payload().(*payload.UpdateVersion)
	if !ok {
		log.Error("tx payload cast failed, tx:", common.ToReversedString(tx.Hash()))
		return
	}

	oriVersionStartHeight := s.VersionStartHeight
	oriVersionEndHeight := s.VersionEndHeight
	s.History.Append(height, func() {
		s.VersionStartHeight = p.StartHeight
		s.VersionEndHeight = p.EndHeight
	}, func() {
		s.VersionStartHeight = oriVersionStartHeight
		s.VersionEndHeight = oriVersionEndHeight
	})
}

func (s *State) getClaimedCRMembersMap() map[string]*state.CRMember {
	crMembersMap := make(map[string]*state.CRMember)
	if s.getCurrentCRMembers == nil {
		return crMembersMap
	}
	crMembers := s.getCurrentCRMembers()
	for _, m := range crMembers {
		if len(m.DPOSPublicKey) != 0 {
			crMembersMap[hex.EncodeToString(m.Info.Code[1:len(m.Info.Code)-1])] = m
		}
	}
	return crMembersMap
}

func (s *State) processReturnVotes(tx interfaces.Transaction, height uint32) {
	pld := tx.Payload().(*payload.ReturnVotes)
	var code []byte
	if tx.PayloadVersion() == payload.ReturnVotesVersionV0 {
		code = pld.Code
	} else {
		code = tx.Programs()[0].Code
	}

	//1. get stake address
	ct, _ := contract.CreateStakeContractByCode(code)
	addr := ct.ToProgramHash()

	s.History.Append(height, func() {
		s.DposV2VoteRights[*addr] -= pld.Value
		s.VotesWithdrawableTxInfo[tx.Hash()] = common2.OutputInfo{
			Recipient: pld.ToAddr,
			Amount:    pld.Value,
		}
	}, func() {
		s.DposV2VoteRights[*addr] += pld.Value
		delete(s.VotesWithdrawableTxInfo, tx.Hash())
	})
}

func (s *State) processDposV2ClaimReward(tx interfaces.Transaction, height uint32) {
	pld := tx.Payload().(*payload.DPoSV2ClaimReward)
	var code []byte
	if tx.PayloadVersion() == payload.DposV2ClaimRewardVersionV0 {
		code = pld.Code
	} else {
		code = tx.Programs()[0].Code
	}

	programHash, _ := utils.GetProgramHashByCode(code)
	addr, _ := programHash.ToAddress()
	s.History.Append(height, func() {
		s.DPoSV2RewardInfo[addr] -= pld.Value
		s.DposV2RewardClaimingInfo[addr] += pld.Value
		s.WithdrawableTxInfo[tx.Hash()] = common2.OutputInfo{
			Recipient: pld.ToAddr,
			Amount:    pld.Value,
		}
		s.ClaimingRewardAddr[tx.Hash()] = *programHash
	}, func() {
		s.DPoSV2RewardInfo[addr] += pld.Value
		s.DposV2RewardClaimingInfo[addr] -= pld.Value
		delete(s.WithdrawableTxInfo, tx.Hash())
		delete(s.ClaimingRewardAddr, tx.Hash())

	})
}

func (s *State) processRetVotesRewardRealWithdraw(tx interfaces.Transaction, height uint32) {
	txs := make(map[common.Uint256]common2.OutputInfo)
	for k, v := range s.StateKeyFrame.VotesWithdrawableTxInfo {
		txs[k] = v
	}
	withdrawPayload := tx.Payload().(*payload.VotesRealWithdrawPayload)
	s.History.Append(height, func() {
		for _, realReturnVotes := range withdrawPayload.VotesRealWithdraw {
			delete(s.StateKeyFrame.VotesWithdrawableTxInfo, realReturnVotes.ReturnVotesTXHash)
		}
	}, func() {
		s.StateKeyFrame.VotesWithdrawableTxInfo = txs
	})
}

func (s *State) processCreateNFT(tx interfaces.Transaction, height uint32) {
	nftPayload := tx.Payload().(*payload.CreateNFT)
	producers := s.getDposV2Producers()
	for _, producer := range producers {
		for stakeAddress, votesInfo := range producer.GetAllDetailedDPoSV2Votes() {
			for referKey, detailVoteInfo := range votesInfo {
				if referKey.IsEqual(nftPayload.ID) {
					ct, _ := contract.CreateStakeContractByCode(referKey.Bytes())
					nftStakeAddress := ct.ToProgramHash()
					s.History.Append(height, func() {
						if _, ok:= producer.detailedDPoSV2Votes[*nftStakeAddress]; !ok {
							producer.detailedDPoSV2Votes[*nftStakeAddress] = make(map[common.Uint256]payload.DetailedVoteInfo)
						}
						producer.detailedDPoSV2Votes[*nftStakeAddress][referKey] = detailVoteInfo
						delete(producer.detailedDPoSV2Votes[stakeAddress], nftPayload.ID)
					}, func() {
						delete(producer.detailedDPoSV2Votes[*nftStakeAddress], referKey)
						producer.detailedDPoSV2Votes[stakeAddress][nftPayload.ID] = detailVoteInfo
					})
					return
				}
			}
		}
	}
}

func (s *State) processDposV2ClaimRewardRealWithdraw(tx interfaces.Transaction, height uint32) {
	txs := make(map[common.Uint256]common2.OutputInfo)
	for k, v := range s.StateKeyFrame.WithdrawableTxInfo {
		txs[k] = v
	}

	oriRewardClaimingAddr := copyRewardClaimingAddrMap(s.ClaimingRewardAddr)
	oriClaimingInfo := copyFixed64Map(s.DposV2RewardClaimingInfo)
	oriClaimedInfo := copyFixed64Map(s.DposV2RewardClaimedInfo)

	withdrawPayload := tx.Payload().(*payload.DposV2ClaimRewardRealWithdraw)

	s.History.Append(height, func() {
		for _, hash := range withdrawPayload.WithdrawTransactionHashes {
			info := s.StateKeyFrame.WithdrawableTxInfo[hash]
			prgramHash := s.ClaimingRewardAddr[hash]
			addr, _ := prgramHash.ToAddress()
			s.DposV2RewardClaimingInfo[addr] -= info.Amount
			s.DposV2RewardClaimedInfo[addr] += info.Amount
			delete(s.StateKeyFrame.WithdrawableTxInfo, hash)
			delete(s.StateKeyFrame.ClaimingRewardAddr, hash)
		}
	}, func() {
		s.StateKeyFrame.WithdrawableTxInfo = txs
		s.DposV2RewardClaimingInfo = oriClaimingInfo
		s.DposV2RewardClaimedInfo = oriClaimedInfo
		s.StateKeyFrame.ClaimingRewardAddr = oriRewardClaimingAddr
	})
}

func (s *State) processRevertToDPOS(Payload *payload.RevertToDPOS, height uint32) {
	oriWorkHeight := s.DPOSWorkHeight
	oriNeedRevertToDPOSTX := s.NeedRevertToDPOSTX
	s.History.Append(height, func() {
		s.DPOSWorkHeight = height + Payload.WorkHeightInterval
		s.NeedRevertToDPOSTX = false
	}, func() {
		s.DPOSWorkHeight = oriWorkHeight
		s.NeedRevertToDPOSTX = oriNeedRevertToDPOSTX
	})
}

func (s *State) getClaimedCRMemberDPOSPublicKeyMap() map[string]*state.CRMember {
	crMembersMap := make(map[string]*state.CRMember)
	if s.getCurrentCRMembers == nil {
		return crMembersMap
	}
	crMembers := s.getCurrentCRMembers()
	for _, m := range crMembers {
		if len(m.DPOSPublicKey) != 0 {
			crMembersMap[hex.EncodeToString(m.DPOSPublicKey)] = m
		}
	}
	return crMembersMap
}

// processEmergencyInactiveArbitrators change producer state according to
// emergency inactive Arbiters
func (s *State) processEmergencyInactiveArbitrators(
	inactivePayload *payload.InactiveArbitrators, height uint32) {

	addEmergencyInactiveArbitrator := func(key string, producer *Producer) {
		s.History.Append(height, func() {
			s.setInactiveProducer(producer, key, height, true)
			s.EmergencyInactiveArbiters[key] = struct{}{}
		}, func() {
			s.revertSettingInactiveProducer(producer, key, height, true)
			delete(s.EmergencyInactiveArbiters, key)
		})
	}

	for _, v := range inactivePayload.Arbitrators {
		nodeKey := hex.EncodeToString(v)
		key, ok := s.NodeOwnerKeys[nodeKey]
		if !ok {
			continue
		}

		// todo consider CR member

		if p, ok := s.ActivityProducers[key]; ok {
			addEmergencyInactiveArbitrator(key, p)
		}
		if p, ok := s.InactiveProducers[key]; ok {
			addEmergencyInactiveArbitrator(key, p)
		}
	}
}

// recordSpecialTx record hash of a special tx
func (s *State) recordSpecialTx(hash common.Uint256, height uint32) {
	s.History.Append(height, func() {
		s.SpecialTxHashes[hash] = struct{}{}
	}, func() {
		delete(s.SpecialTxHashes, hash)
	})
}

// removeSpecialTx record hash of a special tx
func (s *State) RemoveSpecialTx(hash common.Uint256) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	delete(s.SpecialTxHashes, hash)
}

func (s *State) getIllegalPenaltyByHeight(height uint32) common.Fixed64 {
	var illegalPenalty common.Fixed64
	if height >= s.DPoSV2ActiveHeight {
		illegalPenalty = s.ChainParams.DPoSConfiguration.DPoSV2IllegalPenalty
	} else if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
		illegalPenalty = s.ChainParams.DPoSConfiguration.IllegalPenalty
	}

	return illegalPenalty
}

// processIllegalEvidence takes the illegal evidence payload and change producer
// state according to the evidence.
func (s *State) processIllegalEvidence(payloadData interfaces.Payload,
	height uint32) {
	// Get illegal producers from evidence.
	var illegalProducers [][]byte
	switch p := payloadData.(type) {
	case *payload.DPOSIllegalProposals:
		illegalProducers = [][]byte{p.Evidence.Proposal.Sponsor}

	case *payload.DPOSIllegalVotes:
		illegalProducers = [][]byte{p.Evidence.Vote.Signer}

	case *payload.DPOSIllegalBlocks:
		signers := make(map[string]interface{})
		for _, pk := range p.Evidence.Signers {
			signers[hex.EncodeToString(pk)] = nil
		}

		for _, pk := range p.CompareEvidence.Signers {
			key := hex.EncodeToString(pk)
			if _, ok := signers[key]; ok {
				illegalProducers = append(illegalProducers, pk)
			}
		}

	case *payload.SidechainIllegalData:
		illegalProducers = [][]byte{p.IllegalSigner}

	default:
		return
	}

	crMembersMap := s.getClaimedCRMemberDPOSPublicKeyMap()
	illegalPenalty := s.getIllegalPenaltyByHeight(height)

	// Set illegal producers to FoundBad state
	for _, pk := range illegalProducers {
		if cr, ok := crMembersMap[hex.EncodeToString(pk)]; ok {
			if len(cr.DPOSPublicKey) == 0 {
				continue
			}
			oriState := cr.MemberState
			s.History.Append(height, func() {
				s.tryUpdateCRMemberIllegal(cr.Info.DID, height, illegalPenalty)
			}, func() {
				s.tryRevertCRMemberIllegal(cr.Info.DID, oriState, height, illegalPenalty)
			})
		}
		key, ok := s.NodeOwnerKeys[hex.EncodeToString(pk)]
		if !ok {
			continue
		}
		if producer, ok := s.ActivityProducers[key]; ok {
			oriPenalty := producer.penalty
			oriState := producer.state
			oriIllegalHeight := producer.illegalHeight
			s.History.Append(height, func() {
				producer.state = Illegal
				producer.illegalHeight = height
				s.IllegalProducers[key] = producer
				producer.activateRequestHeight = math.MaxUint32
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					producer.penalty += illegalPenalty
				}
				delete(s.ActivityProducers, key)
			}, func() {
				producer.state = oriState
				producer.penalty = oriPenalty
				producer.illegalHeight = oriIllegalHeight
				s.ActivityProducers[key] = producer
				producer.activateRequestHeight = math.MaxUint32
				delete(s.IllegalProducers, key)
			})
			continue
		}

		if producer, ok := s.InactiveProducers[key]; ok {
			oriPenalty := producer.penalty
			oriState := producer.state
			oriIllegalHeight := producer.illegalHeight
			s.History.Append(height, func() {
				producer.state = Illegal
				producer.illegalHeight = height
				s.IllegalProducers[key] = producer
				producer.activateRequestHeight = math.MaxUint32
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					producer.penalty += illegalPenalty
				}
				delete(s.InactiveProducers, key)
			}, func() {
				producer.state = oriState
				producer.penalty = oriPenalty
				producer.illegalHeight = oriIllegalHeight
				s.InactiveProducers[key] = producer
				producer.activateRequestHeight = math.MaxUint32
				delete(s.IllegalProducers, key)
			})
			continue
		}

		if producer, ok := s.IllegalProducers[key]; ok {
			oriPenalty := producer.penalty
			oriIllegalHeight := producer.illegalHeight
			s.History.Append(height, func() {
				producer.illegalHeight = height
				producer.activateRequestHeight = math.MaxUint32
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					producer.penalty += illegalPenalty
				}
			}, func() {
				producer.penalty = oriPenalty
				producer.illegalHeight = oriIllegalHeight
				producer.activateRequestHeight = math.MaxUint32
			})
			continue
		}

		if producer, ok := s.CanceledProducers[key]; ok {
			oriPenalty := producer.penalty
			oriState := producer.state
			s.History.Append(height, func() {
				producer.state = Illegal
				producer.illegalHeight = height
				s.IllegalProducers[key] = producer
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					producer.penalty += illegalPenalty
				}
				delete(s.CanceledProducers, key)
			}, func() {
				producer.state = oriState
				producer.illegalHeight = 0
				producer.penalty = oriPenalty
				s.CanceledProducers[key] = producer
				delete(s.IllegalProducers, key)
			})
			continue
		}
	}
}

// ProcessIllegalBlockEvidence takes a illegal block payload and change the
// producers state immediately.  This is a spacial case that can be handled
// before it packed into a block.
func (s *State) ProcessSpecialTxPayload(p interfaces.Payload, height uint32) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if inactivePayload, ok := p.(*payload.InactiveArbitrators); ok {
		s.processEmergencyInactiveArbitrators(inactivePayload, 0)
	} else {
		s.processIllegalEvidence(p, 0)
	}

	// Commit changes here if no errors found.
	s.History.Commit(height)
}

// setInactiveProducer set active producer to inactive state
func (s *State) setInactiveProducer(producer *Producer, key string,
	height uint32, emergency bool) {
	producer.inactiveSince = height
	producer.activateRequestHeight = math.MaxUint32
	producer.state = Inactive
	producer.selected = false
	s.InactiveProducers[key] = producer
	delete(s.ActivityProducers, key)

	var penalty = s.ChainParams.DPoSConfiguration.InactivePenalty
	if height < s.VersionStartHeight || height >= s.VersionEndHeight {
		if !emergency {
			if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
				producer.penalty += penalty
			}
		} else {
			producer.penalty += s.ChainParams.DPoSConfiguration.EmergencyInactivePenalty

		}
	}
}

// revertSettingInactiveProducer revert operation about setInactiveProducer
func (s *State) revertSettingInactiveProducer(producer *Producer, key string,
	height uint32, emergency bool) {
	producer.inactiveSince = 0
	producer.activateRequestHeight = math.MaxUint32
	producer.state = Active
	s.ActivityProducers[key] = producer
	delete(s.InactiveProducers, key)

	var penalty = s.ChainParams.DPoSConfiguration.InactivePenalty
	if height < s.VersionStartHeight || height >= s.VersionEndHeight {
		if emergency {
			penalty = s.ChainParams.DPoSConfiguration.EmergencyInactivePenalty
		}

		if producer.penalty < penalty {
			producer.penalty = common.Fixed64(0)
		} else {
			producer.penalty -= penalty
		}
	}
}

// countArbitratorsInactivity count Arbiters inactive rounds, and change to
// inactive if more than "MaxInactiveRounds"
func (s *State) countArbitratorsInactivityV3(height uint32,
	confirm *payload.Confirm, dutyIndex int) {
	// check inactive Arbiters after producers has participated in
	if height < s.ChainParams.DPoSV2StartHeight {
		return
	}

	lastPosition := dutyIndex == s.ChainParams.DPoSConfiguration.NormalArbitratorsCount+len(s.ChainParams.DPoSConfiguration.CRCArbiters)-1

	isDPOSAsCR := height > s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight

	// changingArbiters indicates the arbiters that should reset inactive
	// counting state. With the value of true means the producer is on duty or
	// is not current arbiter any more, or just becoming current arbiter; and
	// false means producer is arbiter in both heights and not on duty.
	changingArbiters := make(map[string]bool)
	for _, a := range s.GetArbiters() {
		var key string
		if isDPOSAsCR {
			if !a.IsNormal {
				continue
			}
			key = s.getProducerKey(a.NodePublicKey)
			changingArbiters[key] = false
		} else {
			if !a.IsNormal || (a.IsCRMember && !a.ClaimedDPOSNode) {
				continue
			}
			key = s.getProducerKey(a.NodePublicKey)
			changingArbiters[key] = false
		}
	}
	currSponsor := s.getProducerKey(confirm.Proposal.Sponsor)
	changingArbiters[currSponsor] = true
	crMembersMap := s.getClaimedCRMembersMap()
	// CRC producers are not in the ActivityProducers,
	// so they will not be inactive
	for k, v := range changingArbiters {
		needReset := v // avoiding pass iterator to closure

		if s.isInElectionPeriod != nil && s.isInElectionPeriod() {
			if cr, ok := crMembersMap[k]; ok {
				if cr.MemberState != state.MemberElected {
					continue
				}
				key := k // avoiding pass iterator to closure
				_, ok = s.ActivityProducers[key]
				if !ok {
					mem := cr
					workedInRound := mem.WorkedInRound
					s.updateCRMemberInactiveCountV2(lastPosition, needReset, workedInRound, mem, height, key)
					if needReset && workedInRound != true {
						s.History.Append(height, func() {
							mem.WorkedInRound = true
						}, func() {
							mem.WorkedInRound = false
						})
					}
				}
			}
		}

		key := k // avoiding pass iterator to closure
		producer, ok := s.ActivityProducers[key]
		if !ok {
			continue
		}
		workedInRound := producer.workedInRound
		// if it's the last position and not working in Round then we should add inactiveCountV2++
		s.updateInactiveCountV2(lastPosition, needReset, workedInRound, producer, height, key)

		if needReset && producer.workedInRound != true {
			s.History.Append(height, func() {
				producer.workedInRound = true
			}, func() {
				producer.workedInRound = false
			})
		}
	}

	if lastPosition {
		ps := s.getAllProducers()
		for _, p := range ps {
			cp := p
			// reset workedInRound value
			s.History.Append(height, func() {
				cp.workedInRound = false
			}, func() {
				cp.workedInRound = true
			})
		}

		ms := s.getCurrentCRMembers()
		for _, m := range ms {
			cm := m
			// reset workedInRound value
			s.History.Append(height, func() {
				cm.WorkedInRound = false
			}, func() {
				cm.WorkedInRound = true
			})
		}
	}
}

func (s *State) updateCRMemberInactiveCountV2(lastPosition, needReset, workedInRound bool, member *state.CRMember, height uint32, key string) {
	// if it's the last position and not working in Round then we should add inactiveCountV2++
	if lastPosition && !needReset && !workedInRound {
		originInactiveCountV2 := member.InactiveCountV2
		s.History.Append(height, func() {
			member.InactiveCountV2 += 1
			if member.InactiveCountV2 >= 3 {
				member.MemberState = state.MemberInactive
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					s.updateCRInactivePenalty(member.Info.CID, height)
				}
				member.InactiveCountV2 = 0
			}
		}, func() {
			if member.MemberState == state.MemberInactive && member.InactiveCountV2 >= 3 {
				member.MemberState = state.MemberElected
				if height >= s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight {
					s.revertUpdateCRInactivePenalty(member.Info.CID, height)
				}
			}
			member.InactiveCountV2 = originInactiveCountV2
		})
	} else if lastPosition && (needReset == true || workedInRound) {
		originInactiveCountV2 := member.InactiveCountV2
		s.History.Append(height, func() {
			member.InactiveCountV2 = 0
		}, func() {
			member.InactiveCountV2 = originInactiveCountV2
		})
	}
}

func (s *State) updateInactiveCountV2(lastPosition, needReset, workedInRound bool, producer *Producer, height uint32, key string) {
	// if it's the last position and not working in Round then we should add inactiveCountV2++
	if lastPosition && !needReset && !workedInRound {
		originInactiveCountV2 := producer.inactiveCountV2
		s.History.Append(height, func() {
			producer.inactiveCountV2 += 1
			if producer.inactiveCountV2 >= 3 {
				s.setInactiveProducer(producer, key, height, false)
				producer.inactiveCountV2 = 0
			}
		}, func() {
			if producer.state == Inactive && producer.inactiveCountV2 >= 3 {
				s.revertSettingInactiveProducer(producer, key, height, false)
			}
			producer.inactiveCountV2 = originInactiveCountV2
		})
	} else if lastPosition && (needReset == true || workedInRound) {
		originInactiveCountV2 := producer.inactiveCountV2
		s.History.Append(height, func() {
			producer.inactiveCountV2 = 0
		}, func() {
			producer.inactiveCountV2 = originInactiveCountV2
		})
	}
}

// countArbitratorsInactivity count Arbiters inactive rounds, and change to
// inactive if more than "MaxInactiveRounds"
func (s *State) countArbitratorsInactivityV2(height uint32,
	confirm *payload.Confirm) {
	// check inactive Arbiters after producers has participated in
	if height < s.ChainParams.PublicDPOSHeight {
		return
	}

	isDPOSAsCR := height > s.ChainParams.CRConfiguration.ChangeCommitteeNewCRHeight

	// changingArbiters indicates the arbiters that should reset inactive
	// counting state. With the value of true means the producer is on duty or
	// is not current arbiter any more, or just becoming current arbiter; and
	// false means producer is arbiter in both heights and not on duty.
	changingArbiters := make(map[string]bool)
	for _, a := range s.GetArbiters() {
		if isDPOSAsCR {
			if !a.IsNormal {
				continue
			}
			key := s.getProducerKey(a.NodePublicKey)
			changingArbiters[key] = false
		} else {
			if !a.IsNormal || (a.IsCRMember && !a.ClaimedDPOSNode) {
				continue
			}
			key := s.getProducerKey(a.NodePublicKey)
			changingArbiters[key] = false
		}
	}
	changingArbiters[s.getProducerKey(confirm.Proposal.Sponsor)] = true

	crMembersMap := s.getClaimedCRMembersMap()
	// CRC producers are not in the ActivityProducers,
	// so they will not be inactive
	for k, v := range changingArbiters {
		needReset := v // avoiding pass iterator to closure

		if s.isInElectionPeriod != nil && s.isInElectionPeriod() {
			if cr, ok := crMembersMap[k]; ok {
				if cr.MemberState != state.MemberElected {
					continue
				}
				if isDPOSAsCR && len(cr.DPOSPublicKey) == 0 {
					key := k // avoiding pass iterator to closure
					producer, ok := s.ActivityProducers[key]
					if !ok {
						continue
					}
					oriInactiveCount := uint32(0)
					if producer.selected {
						oriInactiveCount = producer.randomCandidateInactiveCount
					} else {
						oriInactiveCount = producer.inactiveCount
					}
					oriLastUpdateInactiveHeight := producer.lastUpdateInactiveHeight
					oriSelected := producer.selected
					s.History.Append(height, func() {
						s.tryUpdateInactivityV2(key, producer, needReset, height)
					}, func() {
						s.tryRevertInactivity(key, producer, needReset, height,
							oriInactiveCount, oriLastUpdateInactiveHeight, oriSelected)
					})
				} else {
					oriState := cr.MemberState
					oriInactiveCount := cr.InactiveCount
					s.History.Append(height, func() {
						s.tryUpdateCRMemberInactivity(cr.Info.DID, needReset, height)
					}, func() {
						s.tryRevertCRMemberInactivity(cr.Info.DID, oriState, oriInactiveCount, height)
					})
				}
				continue
			}
		}

		key := k // avoiding pass iterator to closure
		producer, ok := s.ActivityProducers[key]
		if !ok {
			continue
		}
		oriInactiveCount := uint32(0)
		if producer.selected {
			oriInactiveCount = producer.randomCandidateInactiveCount
		} else {
			oriInactiveCount = producer.inactiveCount
		}
		oriLastUpdateInactiveHeight := producer.lastUpdateInactiveHeight
		oriSelected := producer.selected
		s.History.Append(height, func() {
			s.tryUpdateInactivityV2(key, producer, needReset, height)
		}, func() {
			s.tryRevertInactivity(key, producer, needReset, height,
				oriInactiveCount, oriLastUpdateInactiveHeight, oriSelected)
		})
	}
}

// CountArbitratorsInactivityV1 count Arbiters inactive rounds, and change to
// inactive if more than "MaxInactiveRounds" with lock
func (s *State) CountArbitratorsInactivityV1(height uint32,
	confirm *payload.Confirm) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.countArbitratorsInactivityV1(height, confirm)
}

// countArbitratorsInactivity count Arbiters inactive rounds, and change to
// inactive if more than "MaxInactiveRounds"
func (s *State) countArbitratorsInactivityV1(height uint32,
	confirm *payload.Confirm) {
	// check inactive Arbiters after producers has participated in
	if height < s.ChainParams.PublicDPOSHeight {
		return
	}
	// changingArbiters indicates the arbiters that should reset inactive
	// counting state. With the value of true means the producer is on duty or
	// is not current arbiter any more, or just becoming current arbiter; and
	// false means producer is arbiter in both heights and not on duty.
	changingArbiters := make(map[string]bool)
	for _, a := range s.GetArbiters() {
		if !a.IsNormal || (a.IsCRMember && !a.ClaimedDPOSNode) {
			continue
		}
		key := s.getProducerKey(a.NodePublicKey)
		changingArbiters[key] = false
	}
	changingArbiters[s.getProducerKey(confirm.Proposal.Sponsor)] = true

	crMembersMap := s.getClaimedCRMembersMap()
	// CRC producers are not in the ActivityProducers,
	// so they will not be inactive
	for k, v := range changingArbiters {
		needReset := v // avoiding pass iterator to closure

		if s.isInElectionPeriod != nil && s.isInElectionPeriod() {
			if cr, ok := crMembersMap[k]; ok {
				if cr.MemberState != state.MemberElected {
					continue
				}
				oriState := cr.MemberState
				oriInactiveCount := cr.InactiveCountingHeight
				s.History.Append(height, func() {
					s.tryUpdateCRMemberInactivity(cr.Info.DID, needReset, height)
				}, func() {
					s.tryRevertCRMemberInactivity(cr.Info.DID, oriState, oriInactiveCount, height)
				})
				continue
			}
		}

		key := k // avoiding pass iterator to closure
		producer, ok := s.ActivityProducers[key]
		if !ok {
			continue
		}

		oriInactiveCount := producer.inactiveCount
		oriLastUpdateInactiveHeight := producer.lastUpdateInactiveHeight
		oriSelected := producer.selected
		s.History.Append(height, func() {
			s.tryUpdateInactivity(key, producer, needReset, height)
		}, func() {
			s.tryRevertInactivity(key, producer, needReset, height,
				oriInactiveCount, oriLastUpdateInactiveHeight, oriSelected)
		})
	}
}

// countArbitratorsInactivity count Arbiters inactive rounds, and change to
// inactive if more than "MaxInactiveRounds"
func (s *State) countArbitratorsInactivityV0(height uint32,
	confirm *payload.Confirm) {
	// check inactive Arbiters after producers has participated in
	if height < s.ChainParams.PublicDPOSHeight {
		return
	}

	// changingArbiters indicates the arbiters that should reset inactive
	// counting state. With the value of true means the producer is on duty or
	// is not current arbiter any more, or just becoming current arbiter; and
	// false means producer is arbiter in both heights and not on duty.
	changingArbiters := make(map[string]bool)
	for k := range s.PreBlockArbiters {
		changingArbiters[k] = true
	}
	s.PreBlockArbiters = make(map[string]struct{})
	for _, a := range s.GetArbiters() {
		key := s.getProducerKey(a.NodePublicKey)
		s.PreBlockArbiters[key] = struct{}{}
		if _, exist := changingArbiters[key]; exist {
			changingArbiters[key] = false
		}
	}
	changingArbiters[s.getProducerKey(confirm.Proposal.Sponsor)] = true

	// CRC producers are not in the ActivityProducers,
	// so they will not be inactive
	for k, v := range changingArbiters {
		needReset := v // avoiding pass iterator to closure

		key := k // avoiding pass iterator to closure
		producer, ok := s.ActivityProducers[key]
		if !ok {
			continue
		}

		oriInactiveCount := uint32(0)
		if producer.selected {
			oriInactiveCount = producer.randomCandidateInactiveCount
		} else {
			oriInactiveCount = producer.inactiveCount
		}
		oriLastUpdateInactiveHeight := producer.lastUpdateInactiveHeight
		oriSelected := producer.selected
		s.History.Append(height, func() {
			s.tryUpdateInactivity(key, producer, needReset, height)
		}, func() {
			s.tryRevertInactivity(key, producer, needReset, height,
				oriInactiveCount, oriLastUpdateInactiveHeight, oriSelected)
		})
	}
}

func (s *State) tryUpdateInactivityV2(key string, producer *Producer,
	needReset bool, height uint32) {
	if needReset {
		if producer.selected {
			producer.randomCandidateInactiveCount = 0
		} else {
			producer.inactiveCount = 0
		}
		producer.lastUpdateInactiveHeight = height
		return
	}

	if height != producer.lastUpdateInactiveHeight+1 {
		if producer.selected {
			producer.randomCandidateInactiveCount = 0
		}
	}

	if producer.selected {
		producer.randomCandidateInactiveCount++
		if producer.randomCandidateInactiveCount >= s.ChainParams.DPoSConfiguration.MaxInactiveRoundsOfRandomNode {
			s.setInactiveProducer(producer, key, height, false)
		}
	} else {
		producer.inactiveCount++
		if producer.inactiveCount >= s.ChainParams.DPoSConfiguration.MaxInactiveRounds {
			s.setInactiveProducer(producer, key, height, false)
			producer.inactiveCount = 0
		}
	}
	producer.lastUpdateInactiveHeight = height
}

func (s *State) tryUpdateInactivity(key string, producer *Producer,
	needReset bool, height uint32) {
	if needReset {
		producer.inactiveCountingHeight = 0
		return
	}

	if producer.inactiveCountingHeight == 0 {
		producer.inactiveCountingHeight = height
	}

	if height-producer.inactiveCountingHeight >= s.ChainParams.DPoSConfiguration.MaxInactiveRounds {
		s.setInactiveProducer(producer, key, height, false)
		producer.inactiveCountingHeight = 0
	}
}

func (s *State) tryRevertInactivity(key string, producer *Producer,
	needReset bool, height, oriInactiveCount uint32,
	oriLastUpdateInactiveHeight uint32, oriSelected bool) {
	producer.lastUpdateInactiveHeight = oriLastUpdateInactiveHeight
	producer.selected = oriSelected
	if needReset {
		if producer.selected {
			producer.randomCandidateInactiveCount = oriInactiveCount

		} else {
			producer.inactiveCount = oriInactiveCount
		}
		return
	}

	if producer.state == Inactive {
		s.revertSettingInactiveProducer(producer, key, height, false)
	}
}

// OnRollbackSeekTo restores the database state to the given height.
func (s *State) RollbackSeekTo(height uint32) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.History.RollbackSeekTo(height)
}

// RollbackTo restores the database state to the given height, if no enough
// History to rollback to return error.
func (s *State) RollbackTo(height uint32) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.History.RollbackTo(height)
}

// GetHistory returns a History state instance storing the producers and votes
// on the historical height.
func (s *State) GetHistory(height uint32) (*StateKeyFrame, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	// Seek to state to target height.
	if err := s.History.SeekTo(height); err != nil {
		return nil, err
	}

	// Take a SnapshotByHeight of the History.
	return s.snapshot(), nil
}

func (s *State) GetLastIrreversibleHeight() uint32 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.LastIrreversibleHeight
}

func (s *State) tryUpdateLastIrreversibleHeight(height uint32) {
	if height < s.ChainParams.DPoSConfiguration.RevertToPOWStartHeight {
		return
	}

	oriLastIrreversibleHeight := s.LastIrreversibleHeight
	oriDPOSStartHeight := s.DPOSStartHeight
	//init LastIrreversibleHeight
	if s.LastIrreversibleHeight == 0 {
		s.History.Append(height, func() {
			s.LastIrreversibleHeight = height - IrreversibleHeight
			s.DPOSStartHeight = s.LastIrreversibleHeight
			log.Debugf("[tryUpdateLastIrreversibleHeight] init LastIrreversibleHeight %d, DPOSStartHeight",
				s.LastIrreversibleHeight, s.DPOSStartHeight)
		}, func() {
			s.LastIrreversibleHeight = oriLastIrreversibleHeight
			s.DPOSStartHeight = oriDPOSStartHeight
			log.Debugf("[tryUpdateLastIrreversibleHeight] init rollback LastIrreversibleHeight %d, DPOSStartHeight",
				s.LastIrreversibleHeight, s.DPOSStartHeight)
		})

	} else if s.ConsensusAlgorithm == DPOS {
		//from pow to dpow
		if s.DPOSWorkHeight != 0 && height == s.DPOSWorkHeight+1 {
			s.History.Append(height, func() {
				s.DPOSStartHeight = height
				log.Debugf("[tryUpdateLastIrreversibleHeight] from pow to dpow  DPOSStartHeight",
					s.DPOSStartHeight)
			}, func() {
				s.DPOSStartHeight = oriDPOSStartHeight
				log.Debugf("[tryUpdateLastIrreversibleHeight] from pow to dpow rollback DPOSStartHeight",
					s.DPOSStartHeight)
			})
		}
		if height-s.DPOSStartHeight >= IrreversibleHeight {
			s.History.Append(height, func() {
				s.DPOSStartHeight++
				s.LastIrreversibleHeight = s.DPOSStartHeight
				log.Debugf("[tryUpdateLastIrreversibleHeight] LastIrreversibleHeight %d, DPOSStartHeight %d",
					s.LastIrreversibleHeight, s.DPOSStartHeight)
			}, func() {
				s.LastIrreversibleHeight = oriLastIrreversibleHeight
				s.DPOSStartHeight = oriDPOSStartHeight
				log.Debugf("[tryUpdateLastIrreversibleHeight] rollback LastIrreversibleHeight %d, DPOSStartHeight %d",
					s.LastIrreversibleHeight, s.DPOSStartHeight)
			})
		}
	}
}

// is this Height Irreversible
func (s *State) IsIrreversible(curBlockHeight uint32, detachNodesLen int) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if curBlockHeight <= s.ChainParams.CRCOnlyDPOSHeight {
		return false
	}

	if curBlockHeight-uint32(detachNodesLen) <= s.LastIrreversibleHeight {
		return true
	}
	if curBlockHeight >= s.ChainParams.DPoSConfiguration.RevertToPOWStartHeight {
		if s.ConsensusAlgorithm == DPOS {
			if detachNodesLen >= IrreversibleHeight {
				return true
			}
		}
	} else {
		if detachNodesLen > IrreversibleHeight {
			return true
		}
	}
	return false
}

func (s *State) handleEvents(event *events.Event) {
	switch event.Type {
	case events.ETCRCChangeCommittee:
		s.mtx.Lock()
		nodePublicKeyMap := s.getAllNodePublicKey()
		for nodePubKey := range s.NodeOwnerKeys {
			_, ok := nodePublicKeyMap[nodePubKey]
			if !ok {
				delete(s.NodeOwnerKeys, nodePubKey)
			}
		}
		s.LastCRNodeOwnerKeys = copyStringMap(s.CurrentCRNodeOwnerKeys)
		s.CurrentCRNodeOwnerKeys = copyStringMap(s.NextCRNodeOwnerKeys)
		s.NextCRNodeOwnerKeys = make(map[string]string)
		s.mtx.Unlock()
	}
}

// NewState returns a new State instance.
func NewState(chainParams *config.Configuration, getArbiters func() []*ArbiterInfo,
	getCRMembers func() []*state.CRMember,
	getNextCRMembers func() []*state.CRMember,
	isInElectionPeriod func() bool,
	getProducerDepositAmount func(common.Uint168) (common.Fixed64, error),
	tryUpdateCRMemberInactivity func(did common.Uint168, needReset bool, height uint32),
	tryRevertCRMemberInactivityfunc func(did common.Uint168, oriState state.MemberState, oriInactiveCount uint32, height uint32),
	tryUpdateCRMemberIllegal func(did common.Uint168, height uint32, illegalPenalty common.Fixed64),
	tryRevertCRMemberIllegal func(did common.Uint168, oriState state.MemberState, height uint32,
		illegalPenalty common.Fixed64),
	updateCRInactivePenalty func(cid common.Uint168, height uint32),
	revertUpdateCRInactivePenalty func(cid common.Uint168, height uint32)) *State {
	state := State{
		ChainParams:                   chainParams,
		GetArbiters:                   getArbiters,
		getCurrentCRMembers:           getCRMembers,
		getNextCRMembers:              getNextCRMembers,
		isInElectionPeriod:            isInElectionPeriod,
		GetProducerDepositAmount:      getProducerDepositAmount,
		History:                       utils.NewHistory(maxHistoryCapacity),
		StateKeyFrame:                 NewStateKeyFrame(),
		tryUpdateCRMemberInactivity:   tryUpdateCRMemberInactivity,
		tryRevertCRMemberInactivity:   tryRevertCRMemberInactivityfunc,
		tryUpdateCRMemberIllegal:      tryUpdateCRMemberIllegal,
		tryRevertCRMemberIllegal:      tryRevertCRMemberIllegal,
		updateCRInactivePenalty:       updateCRInactivePenalty,
		revertUpdateCRInactivePenalty: revertUpdateCRInactivePenalty,
	}
	events.Subscribe(state.handleEvents)
	return &state
}
