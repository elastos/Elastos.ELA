// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package outputpayload

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const StakeOutputVersion byte = 0x00

// CandidateVotes defines the voting information for individual candidates.
type StakeOutput struct {
	Version      byte
	StakeAddress common.Uint168
}

func (ev *StakeOutput) Data() []byte {
	return nil
}

func (ev *StakeOutput) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{ev.Version}); err != nil {
		return err
	}
	if err := ev.StakeAddress.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (ev *StakeOutput) Deserialize(r io.Reader) error {
	version, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	ev.Version = version[0]
	if err := ev.StakeAddress.Deserialize(r); err != nil {
		return err
	}

	return nil
}

func (ev *StakeOutput) GetVersion() byte {
	return ev.Version
}

func (ev *StakeOutput) Validate() error {
	if ev == nil {
		return errors.New("exchange vote output payload is nil")
	}
	if ev.Version > StakeOutputVersion {
		return errors.New("invalid exchange vote version")
	}

	if contract.GetPrefixType(ev.StakeAddress) != contract.PrefixDPoSV2 {
		return errors.New("second output address need to be Standard or MultiSig")
	}
	return nil
}

func (ev *StakeOutput) String() string {
	addr, _ := ev.StakeAddress.ToAddress()
	return fmt.Sprint("{\n\t\t\t\t",
		"StakeAddress: ", addr, "\n\t\t\t\t")
}
