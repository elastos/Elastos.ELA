package store

import (
	"sort"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

type EventStoreAnalyzerConfig struct {
	InactiveEliminateCount uint32
	Store                  IDposStore
	DutyState              *state.DutyState
}

type EventStoreAnalyzer struct {
	cfg EventStoreAnalyzerConfig
}

func (e *EventStoreAnalyzer) ParseInactiveArbiters() (
	result []string) {

	viewCount := e.getLastConsensusViewCount()

	arbitratorsVoteCount := map[string]int{}
	totalVotes := e.getLastConsensusVoteEvents()
	for _, v := range totalVotes {
		if _, exists := arbitratorsVoteCount[v.Signer]; exists {
			arbitratorsVoteCount[v.Signer] += 1
		} else {
			arbitratorsVoteCount[v.Signer] = 0
		}
	}

	type sortItem struct {
		Ratio float64
		PK    string
	}
	var sortItems []sortItem
	currentArbiters := e.cfg.DutyState.GetArbiters()
	for _, v := range currentArbiters {
		hexPk := common.BytesToHexString(v)

		ratio := float64(0)
		if count, exists := arbitratorsVoteCount[hexPk]; exists {
			ratio = float64(count) / float64(viewCount)
		}

		sortItems = append(sortItems, sortItem{
			Ratio: ratio,
			PK:    hexPk,
		})
	}

	sort.Slice(sortItems, func(i, j int) bool {
		return sortItems[i].Ratio > sortItems[j].Ratio
	})
	for i := 0; i < len(sortItems) &&
		i < int(e.cfg.InactiveEliminateCount); i++ {
		result = append(result, sortItems[i].PK)
	}

	sort.Strings(result)
	return result
}

func (e *EventStoreAnalyzer) getLastConsensusViewCount() uint32 {
	//todo complete me
	return 0
}

func (e *EventStoreAnalyzer) getLastConsensusVoteEvents() []log.VoteEvent {
	//todo complete me
	return nil
}

func NewEventStoreAnalyzer(cfg EventStoreAnalyzerConfig) *EventStoreAnalyzer {
	return &EventStoreAnalyzer{cfg: cfg}
}
