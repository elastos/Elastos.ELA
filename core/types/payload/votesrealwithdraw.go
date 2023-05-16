package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const VotesRealWithdrawPayloadVersion byte = 0x00

type VotesRealWidhdraw struct {
	ReturnVotesTXHash common.Uint256
	StakeAddress      common.Uint168
	Value             common.Fixed64
}

func (p *VotesRealWidhdraw) Serialize(w io.Writer) error {
	if err := p.ReturnVotesTXHash.Serialize(w); err != nil {
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

func (p *VotesRealWidhdraw) Deserialize(r io.Reader) error {
	if err := p.ReturnVotesTXHash.Deserialize(r); err != nil {
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

type VotesRealWithdrawPayload struct {
	VotesRealWithdraw []VotesRealWidhdraw
}

func (p *VotesRealWithdrawPayload) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (p *VotesRealWithdrawPayload) Serialize(w io.Writer, version byte) error {
	if err := common.WriteVarUint(w, uint64(len(p.VotesRealWithdraw))); err != nil {
		return err
	}

	for _, returnVote := range p.VotesRealWithdraw {
		if err := returnVote.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

func (p *VotesRealWithdrawPayload) Deserialize(r io.Reader, version byte) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	for i := uint64(0); i < count; i++ {
		var withDraw VotesRealWidhdraw
		err := withDraw.Deserialize(r)
		if err != nil {
			return err
		}
		p.VotesRealWithdraw = append(p.VotesRealWithdraw, withDraw)
	}

	return nil
}
