// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package outputpayload

import (
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
)

const ExchangeVotesOutputVersion byte = 0x00

type ExchangeVotesOutput struct {
	Version      byte
	StakeAddress common.Uint168
}

func (ev *ExchangeVotesOutput) Data() []byte {
	return nil
}

func (ev *ExchangeVotesOutput) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{ev.Version}); err != nil {
		return err
	}
	if err := ev.StakeAddress.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (ev *ExchangeVotesOutput) Deserialize(r io.Reader) error {
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

func (ev *ExchangeVotesOutput) GetVersion() byte {
	return ev.Version
}

func (ev *ExchangeVotesOutput) Validate() error {
	if ev == nil {
		return errors.New("exchange votes output payload is nil")
	}
	if ev.Version > ExchangeVotesOutputVersion {
		return errors.New("invalid exchange vote version")
	}

	if contract.GetPrefixType(ev.StakeAddress) != contract.PrefixDPoSV2 {
		return errors.New("second output address need to be Standard or MultiSig")
	}
	return nil
}

func (ev *ExchangeVotesOutput) String() string {
	addr, _ := ev.StakeAddress.ToAddress()
	return fmt.Sprint("{\n\t\t\t\t",
		"StakeAddress: ", addr, "\n\t\t\t\t")
}
