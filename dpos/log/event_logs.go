package log

import "github.com/elastos/Elastos.ELA/log"

type EventLogs struct {
}

func (e *EventLogs) OnProposalArrived(prop ProposalEvent) {
	log.Info("[OnProposalArrived], Proposal:", prop.Proposal,
		"BlockHash:", prop.BlockHash, "ReceivedTime:", prop.ReceivedTime, "Result:", prop.Result)
}

func (e *EventLogs) OnProposalFinished(prop ProposalEvent) {
	log.Info("[OnProposalFinished], Proposal:", prop.Proposal,
		"BlockHash:", prop.BlockHash, "EndTime:", prop.EndTime, "Result:", prop.Result)
}

func (e *EventLogs) OnVoteArrived(vote VoteEvent) {
	log.Info("[OnVoteArrived], Signer:", vote.Signer, "ReceivedTime:", vote.ReceivedTime, "Result:", vote.Result)
}

func (e *EventLogs) OnViewStarted(view ViewEvent) {
	log.Info("[OnViewStarted], OnDutyArbitrator:", view.OnDutyArbitrator,
		"StartTime:", view.StartTime, "Offset:", view.Offset, "Height", view.Height)
}

func (e *EventLogs) OnConsensusStarted(cons ConsensusEvent) {
	log.Info("[OnConsensusStarted], StartTime:", cons.StartTime, "Height:", cons.Height)
}

func (e *EventLogs) OnConsensusFinished(cons ConsensusEvent) {
	log.Info("[OnConsensusFinished], EndTime:", cons.EndTime, "Height:", cons.EndTime)
}
