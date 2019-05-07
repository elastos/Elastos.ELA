package msg

import "github.com/elastos/Elastos.ELA/core/types/payload"

type VoteReject struct {
	VoteAccept
}

func NewVoteReject(payload *payload.DPOSProposalVote) *VoteReject {
	return &VoteReject{VoteAccept{Payload: *payload}}
}

func (msg *VoteReject) CMD() string {
	return CmdVoteReject
}
