package state

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/events"
)

const (
	// minSignRatio defines the minimum sign ratio of the DPOS arbiters.
	minSignRatio = float64(2) / float64(3)
)

// crcArbiter defines a struct for CRC arbiter.
type crcArbiter struct {
	// PublicKey indicates the arbiter's public key bytes.
	PublicKey []byte

	// NetAddress indicates the arbiter's network address (host:port).
	NetAddress string
}

// DutyState is a memory state of on duty arbiters.
type DutyState struct {
	*State
	chainParams   *config.Params
	bestHeight    func() uint32
	arbitersCount uint32
	orgArbiters   [][]byte

	mtx                     sync.Mutex
	dutyIndex               uint32
	arbiters                [][]byte
	candidates              [][]byte
	arbitersProgramHashes   []*common.Uint168
	candidatesProgramHashes []*common.Uint168
	nextArbiters            [][]byte
	nextCandidates          [][]byte
}

func (d *DutyState) ProcessBlock(block *types.Block, confirm *payload.Confirm) {
	d.mtx.Lock()
	// get on duty arbiter before lock in case of recurrent lock
	onDutyArbiter := d.getOnDutyArbiter(block.Height, 0)
	arbitersCount := d.getArbitersCount()
	d.State.ProcessBlock(block, confirm)
	if confirm != nil {
		d.countArbitersInactivity(block.Height, arbitersCount,
			onDutyArbiter, confirm)
	}
	d.history.append(block.Height, func() {
		d.processState(block.Height)
	}, func() {
		d.rollbackState(block.Height)
	})

	// Commit changes here if no errors found.
	d.history.commit(block.Height)
	d.mtx.Unlock()
}

// countArbitersInactivity count arbitrators inactive rounds, and change to
// inactive if more than "MaxInactiveRounds"
func (d *DutyState) countArbitersInactivity(height, totalArbitersCount uint32,
	onDutyArbiter []byte, confirm *payload.Confirm) {
	// check inactive arbitrators after producers has participated in
	if int(totalArbitersCount) == len(d.chainParams.CRCArbiters) {
		return
	}

	key := d.getProducerKey(onDutyArbiter)

	isMissOnDuty := !bytes.Equal(onDutyArbiter, confirm.Proposal.Sponsor)
	if producer, ok := d.activityProducers[key]; ok {
		count, existMissingRecord := d.onDutyMissingCounts[key]

		d.history.append(height, func() {
			d.tryUpdateOnDutyInactivity(isMissOnDuty, existMissingRecord,
				key, producer, height)
		}, func() {
			d.tryRevertOnDutyInactivity(isMissOnDuty, existMissingRecord, key,
				producer, height, count)
		})
	}
}

func (d *DutyState) tryRevertOnDutyInactivity(isMissOnDuty bool,
	existMissingRecord bool, key string, producer *Producer,
	height uint32, count uint32) {
	if isMissOnDuty {
		if existMissingRecord {
			if producer.state == Inactivate {
				d.revertSettingInactiveProducer(producer, key, height)
				d.onDutyMissingCounts[key] = d.chainParams.MaxInactiveRounds
			} else {
				if count > 1 {
					d.onDutyMissingCounts[key]--
				} else {
					delete(d.onDutyMissingCounts, key)
				}
			}
		}
	} else {
		if existMissingRecord {
			d.onDutyMissingCounts[key] = count
		}
	}
}

func (d *DutyState) tryUpdateOnDutyInactivity(isMissOnDuty bool,
	existMissingRecord bool, key string, producer *Producer, height uint32) {
	if isMissOnDuty {
		if existMissingRecord {
			d.onDutyMissingCounts[key]++

			if d.onDutyMissingCounts[key] >=
				d.chainParams.MaxInactiveRounds {
				d.setInactiveProducer(producer, key, height)
				delete(d.onDutyMissingCounts, key)
			}
		} else {
			d.onDutyMissingCounts[key] = 1
		}
	} else {
		if existMissingRecord {
			delete(d.onDutyMissingCounts, key)
		}
	}
}

func (d *DutyState) ForceChange(height uint32) error {
	d.mtx.Lock()
	if err := d.updateNextArbiters(height); err != nil {
		return err
	}

	if err := d.changeCurrentArbiters(); err != nil {
		return err
	}
	d.mtx.Unlock()

	events.Notify(events.ETNewArbiterElection, d.nextArbiters)

	return nil
}

func (d *DutyState) NormalChange(height uint32) error {
	d.mtx.Lock()
	if err := d.changeCurrentArbiters(); err != nil {
		return err
	}

	if err := d.updateNextArbiters(height); err != nil {
		return err
	}
	d.mtx.Unlock()

	events.Notify(events.ETNewArbiterElection, d.nextArbiters)

	return nil
}

func (d *DutyState) processState(height uint32) {
	forceChange, normalChange := d.isNewElection(height + 1)
	if forceChange {
		d.ForceChange(height)
	} else if normalChange {
		d.NormalChange(height)
	} else {
		d.dutyIndex++
	}
}

func (d *DutyState) rollbackState(height uint32) {
	if d.dutyIndex == 0 {
		//todo complete me
	} else {
		d.dutyIndex--
	}
}

func (d *DutyState) GetDutyIndex() uint32 {
	d.mtx.Lock()
	index := d.dutyIndex
	d.mtx.Unlock()
	return index
}

func (d *DutyState) SetDutyIndex(index uint32) {
	d.mtx.Lock()
	d.dutyIndex = index
	d.mtx.Unlock()
}

func (d *DutyState) IsArbiter(pk []byte) bool {
	for _, v := range d.arbiters {
		if bytes.Equal(pk, v) {
			return true
		}
	}
	return false
}

func (d *DutyState) GetArbiters() [][]byte {
	d.mtx.Lock()
	result := d.arbiters
	d.mtx.Unlock()
	return result
}

func (d *DutyState) GetCandidates() [][]byte {
	d.mtx.Lock()
	result := d.candidates
	d.mtx.Unlock()
	return result
}

func (d *DutyState) GetNextArbiters() [][]byte {
	d.mtx.Lock()
	result := d.nextArbiters
	d.mtx.Unlock()
	return result
}

func (d *DutyState) GetNextCandidates() [][]byte {
	d.mtx.Lock()
	result := d.nextCandidates
	d.mtx.Unlock()
	return result
}

func (d *DutyState) GetArbiterProgramHashes() []*common.Uint168 {
	d.mtx.Lock()
	result := d.arbitersProgramHashes
	d.mtx.Unlock()
	return result
}

func (d *DutyState) GetCandidateProgramHashes() []*common.Uint168 {
	d.mtx.Lock()
	result := d.candidatesProgramHashes
	d.mtx.Unlock()
	return result
}

func (d *DutyState) getOnDutyArbiter(height, offset uint32) []byte {
	// DPOS consensus started.
	if height >= d.chainParams.DPOSStartHeight {
		arbLen := len(d.arbiters)
		if arbLen == 0 {
			return nil
		}
		index := (d.dutyIndex + offset) % uint32(arbLen)
		return d.arbiters[index]
	}

	// Before DPOS consensus started.
	return d.getOnDutyArbiterV0(offset)
}

func (d *DutyState) GetOnDutyArbiterByHeight(height uint32) []byte {
	d.mtx.Lock()
	arbiter := d.getOnDutyArbiter(height, 0)
	d.mtx.Unlock()
	return arbiter
}

func (d *DutyState) GetOnDutyArbiter() []byte {
	d.mtx.Lock()
	arbiter := d.getOnDutyArbiter(d.bestHeight()+1, 0)
	d.mtx.Unlock()
	return arbiter
}

func (d *DutyState) GetNextOnDutyArbiter(offset uint32) []byte {
	d.mtx.Lock()
	arbiter := d.getOnDutyArbiter(d.bestHeight()+1, offset)
	d.mtx.Unlock()
	return arbiter
}

func (d *DutyState) GetArbitersCount() uint32 {
	d.mtx.Lock()
	count := d.getArbitersCount()
	d.mtx.Unlock()
	return count
}

func (d *DutyState) GetArbitersMajorityCount() uint32 {
	d.mtx.Lock()
	count := uint32(float64(d.getArbitersCount()) * minSignRatio)
	d.mtx.Unlock()
	return count
}

func (d *DutyState) HasArbitersMajorityCount(num uint32) bool {
	return num > d.GetArbitersMajorityCount()
}

func (d *DutyState) HasArbitersMinorityCount(num uint32) bool {
	return num >= d.chainParams.ArbitersCount-d.GetArbitersMajorityCount()
}

func (d *DutyState) isNewElection(height uint32) (forceChange bool, normalChange bool) {
	if height >= d.chainParams.DPOSStartHeight {

		// when change to "H1" or "H2" height should fire new election immediately
		if height == d.chainParams.DPOSStartHeight ||
			height == d.chainParams.OpenArbitersHeight {
			return true, false
		}
		return false, d.dutyIndex == d.arbitersCount-1
	}

	return false, false
}

func (d *DutyState) changeCurrentArbiters() error {
	d.arbiters = d.nextArbiters
	d.candidates = d.nextCandidates

	// Sort arbiters in public key increasing order.
	sort.Slice(d.arbiters, func(i, j int) bool {
		return bytes.Compare(d.arbiters[i], d.arbiters[j]) < 0
	})

	if err := d.updateArbitersProgramHashes(); err != nil {
		return err
	}

	d.dutyIndex = 1
	return nil
}

func (d *DutyState) getGeneralArbiters(height, needArbiters uint32) ([][]byte, error) {
	// After open general arbiters into DPOS consensus.
	if height >= d.chainParams.OpenArbitersHeight {
		producers := d.GetActiveProducers()
		if len(producers) < int(needArbiters) {
			return nil, fmt.Errorf("candidates not enough, need %d got"+
				" %d", needArbiters, len(producers))
		}
		sort.Slice(producers, func(i, j int) bool {
			return producers[i].Votes() > producers[j].Votes()
		})

		result := make([][]byte, needArbiters)
		for i := uint32(0); i < needArbiters; i++ {
			result[i] = producers[i].NodePublicKey()
		}
		return result, nil
	}

	// Before general arbiters join into DPOS consensus.
	if height >= d.chainParams.DPOSStartHeight {
		return nil, nil
	}

	return d.getArbitersV0(), nil
}

func (d *DutyState) getGeneralCandidates(height, needArbiters uint32) ([][]byte, error) {
	// After open general arbiters into DPOS consensus.
	if height+1 >= d.chainParams.OpenArbitersHeight {
		producers := d.GetActiveProducers()
		if len(producers) < int(needArbiters) {
			return nil, fmt.Errorf("candidates not enough, need %d got"+
				" %d", needArbiters, len(producers))
		}
		sort.Slice(producers, func(i, j int) bool {
			return producers[i].Votes() > producers[j].Votes()
		})

		result := make([][]byte, 0)
		for i := needArbiters; i < uint32(len(producers)) && i < needArbiters+d.
			chainParams.CandidatesCount; i++ {
			result = append(result, producers[i].NodePublicKey())
		}
		return result, nil
	}

	return nil, nil
}

func (d *DutyState) updateNextArbiters(height uint32) error {
	crcCount := uint32(0)
	d.nextArbiters = make([][]byte, 0)
	for _, v := range d.crcArbiters {
		if !d.isInactiveProducer(v.PublicKey) {
			d.nextArbiters = append(d.nextArbiters, v.PublicKey)
		} else {
			crcCount++
		}
	}
	needArbiters := d.chainParams.ArbitersCount + crcCount
	producers, err := d.getGeneralArbiters(height, needArbiters)
	if err != nil {
		return err
	}
	for _, v := range producers {
		d.nextArbiters = append(d.nextArbiters, v)
	}

	d.nextCandidates, err = d.getGeneralCandidates(height, needArbiters)
	return err
}

func (d *DutyState) getArbitersCount() uint32 {
	return uint32(len(d.arbiters))
}

func (d *DutyState) updateArbitersProgramHashes() error {
	d.arbitersProgramHashes = make([]*common.Uint168, len(d.arbiters))
	for index, v := range d.arbiters {
		hash, err := contract.PublicKeyToStandardProgramHash(v)
		if err != nil {
			return err
		}
		d.arbitersProgramHashes[index] = hash
	}

	d.candidatesProgramHashes = make([]*common.Uint168, len(d.candidates))
	for index, v := range d.candidates {
		hash, err := contract.PublicKeyToStandardProgramHash(v)
		if err != nil {
			return err
		}
		d.candidatesProgramHashes[index] = hash
	}

	return nil
}

// NewDutyState creates a new DutyState instance with the given config.
func NewDutyState(chainParams *config.Params, bestHeight func() uint32) (*DutyState, error) {
	originArbiters := make([][]byte, len(chainParams.OriginArbiters))
	originArbitersProgramHashes := make([]*common.Uint168, len(chainParams.OriginArbiters))
	for i, arbiter := range chainParams.OriginArbiters {
		a, err := common.HexStringToBytes(arbiter)
		if err != nil {
			return nil, err
		}
		originArbiters[i] = a

		publicKey, err := common.HexStringToBytes(arbiter)
		if err != nil {
			return nil, err
		}
		hash, err := contract.PublicKeyToStandardProgramHash(publicKey)
		if err != nil {
			return nil, err
		}
		originArbitersProgramHashes[i] = hash
	}

	orgArbiters := make([][]byte, 0, len(chainParams.OriginArbiters))
	for _, arbiter := range chainParams.OriginArbiters {
		pubKey, err := hex.DecodeString(arbiter)
		if err != nil {
			return nil, err
		}
		orgArbiters = append(orgArbiters, pubKey)
	}

	crcArbiters, err := convertArbiters(chainParams.CRCArbiters)
	if err != nil {
		return nil, err
	}

	arbitersCount := chainParams.ArbitersCount + uint32(len(crcArbiters))
	a := &DutyState{
		State:                 NewState(chainParams),
		chainParams:           chainParams,
		bestHeight:            bestHeight,
		arbitersCount:         arbitersCount,
		orgArbiters:           orgArbiters,
		arbiters:              originArbiters,
		arbitersProgramHashes: originArbitersProgramHashes,
		nextArbiters:          originArbiters,
		nextCandidates:        make([][]byte, 0),
	}

	a.crcArbiters = crcArbiters
	a.crcProgramHashes = make(map[common.Uint168]struct{})
	for _, v := range a.crcArbiters {
		a.nextArbiters = append(a.nextArbiters, v.PublicKey)

		hash, err := contract.PublicKeyToStandardProgramHash(v.PublicKey)
		if err != nil {
			return nil, err
		}
		a.crcProgramHashes[*hash] = struct{}{}
	}

	return a, nil
}

func convertArbiters(arbiters []config.CRCArbiter) ([]crcArbiter, error) {
	result := make([]crcArbiter, 0, len(arbiters))
	for _, v := range arbiters {
		arbiterByte, err := common.HexStringToBytes(v.PublicKey)
		if err != nil {
			return nil, err
		}
		result = append(result, crcArbiter{
			PublicKey:  arbiterByte,
			NetAddress: v.NetAddress,
		})
	}

	return result, nil
}
