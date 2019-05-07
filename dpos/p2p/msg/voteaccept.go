package msg

import (
	"io"

	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const votePayloadSize = 297 //164+67+1+65

type VoteAccept struct {
	Payload payload.DPOSProposalVote
}

func NewVoteAccept(payload *payload.DPOSProposalVote) *VoteAccept {
	return &VoteAccept{Payload: *payload}
}

func (msg *VoteAccept) CMD() string {
	return CmdVoteAccept
}

func (msg *VoteAccept) MaxLength() uint32 {
	return votePayloadSize
}

func (msg *VoteAccept) Serialize(w io.Writer) error {
	return msg.Payload.Serialize(w)
}

func (msg *VoteAccept) Deserialize(r io.Reader) error {
	return msg.Payload.Deserialize(r)
}
