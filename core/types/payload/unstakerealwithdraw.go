package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const UnstakeRealWithdrawPayloadVersion byte = 0x00

type UnstakeRealWidhdraw struct {
	UnstakeTXHash common.Uint256
	StakeAddress  common.Uint168
	Value         common.Fixed64
}

func (p *UnstakeRealWidhdraw) Serialize(w io.Writer) error {
	if err := p.UnstakeTXHash.Serialize(w); err != nil {
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

func (p *UnstakeRealWidhdraw) Deserialize(r io.Reader) error {
	if err := p.UnstakeTXHash.Deserialize(r); err != nil {
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

type UnstakeRealWithdrawPayload struct {
	UnstakeRealWithdraw []UnstakeRealWidhdraw
}

func (p *UnstakeRealWithdrawPayload) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (p *UnstakeRealWithdrawPayload) Serialize(w io.Writer, version byte) error {
	if err := common.WriteVarUint(w, uint64(len(p.UnstakeRealWithdraw))); err != nil {
		return err
	}

	for _, returnVote := range p.UnstakeRealWithdraw {
		if err := returnVote.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

func (p *UnstakeRealWithdrawPayload) Deserialize(r io.Reader, version byte) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	p.UnstakeRealWithdraw = make([]UnstakeRealWidhdraw, count)

	for i := uint64(0); i < count; i++ {
		err := p.UnstakeRealWithdraw[i].Deserialize(r)
		if err != nil {
			return err
		}
	}

	return nil
}
