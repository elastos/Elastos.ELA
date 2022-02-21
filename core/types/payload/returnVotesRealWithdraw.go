package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const ReturnVotesRealWithdrawPayloadVersion byte = 0x00

type ReturnVoteRealWidhdraw struct {
	RetVotesTXHash common.Uint256
	StakeAddress   common.Uint168
	Value          common.Fixed64
}

func (p *ReturnVoteRealWidhdraw) Serialize(w io.Writer) error {
	if err := p.RetVotesTXHash.Serialize(w); err != nil {
		return err
	}
	if err := p.StakeAddress.Serialize(w); err != nil {
		return err
	}
	if err := p.Value.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *ReturnVoteRealWidhdraw) Deserialize(r io.Reader) error {
	if err := p.RetVotesTXHash.Deserialize(r); err != nil {
		return err
	}

	if err := p.StakeAddress.Deserialize(r); err != nil {
		return err
	}

	if err := p.Value.Deserialize(r); err != nil {
		return err
	}

	return nil
}

type ReturnVotesRealWithdrawPayload struct {
	ReturnVotesRealWithdraw []ReturnVoteRealWidhdraw
}

func (p *ReturnVotesRealWithdrawPayload) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (p *ReturnVotesRealWithdrawPayload) Serialize(w io.Writer, version byte) error {

	if err := common.WriteUint64(w, uint64(len(p.ReturnVotesRealWithdraw))); err != nil {
		return err
	}

	for _, returnVote := range p.ReturnVotesRealWithdraw {
		if err := returnVote.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

func (p *ReturnVotesRealWithdrawPayload) Deserialize(r io.Reader, version byte) error {

	count, err := common.ReadUint64(r)
	if err != nil {
		return err
	}
	p.ReturnVotesRealWithdraw = make([]ReturnVoteRealWidhdraw, count)

	for i := uint64(0); i < count; i++ {
		err := p.ReturnVotesRealWithdraw[i].Deserialize(r)
		if err != nil {
			return err
		}
	}

	return nil
}
