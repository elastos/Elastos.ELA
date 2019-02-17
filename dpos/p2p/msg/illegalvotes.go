package msg

import (
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"io"
)

const MaxIllegalVoteSize = 1000000

type IllegalVotes struct {
	Votes payload.DposIllegalVotes
}

func (msg *IllegalVotes) CMD() string {
	return CmdIllegalVotes
}

func (msg *IllegalVotes) MaxLength() uint32 {
	return MaxIllegalVoteSize
}

func (msg *IllegalVotes) Serialize(w io.Writer) error {
	if err := msg.Votes.Serialize(w, payload.PayloadIllegalVoteVersion); err != nil {
		return err
	}

	return nil
}

func (msg *IllegalVotes) Deserialize(r io.Reader) error {
	if err := msg.Votes.Deserialize(r, payload.PayloadIllegalVoteVersion); err != nil {
		return err
	}

	return nil
}
