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
)

const CrossChainOutputVersion byte = 0x00

// CrossChainOutput defines the output payload for cross chain.
type CrossChainOutput struct {
	Version       byte
	TargetAddress string
	TargetAmount  common.Fixed64
}

func (o *CrossChainOutput) Data() []byte {
	return nil
}

func (o *CrossChainOutput) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{byte(o.Version)}); err != nil {
		return err
	}
	if err := common.WriteVarString(w, o.TargetAddress); err != nil {
		return err
	}
	if err := o.TargetAmount.Serialize(w); err != nil {
		return err
	}
	return nil
}

func (o *CrossChainOutput) Deserialize(r io.Reader) error {
	version, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	o.Version = version[0]

	address, err := common.ReadVarString(r)
	if err != nil {
		return err
	}
	o.TargetAddress = address

	if err = o.TargetAmount.Deserialize(r); err != nil {
		return err
	}
	return nil
}

func (o *CrossChainOutput) GetVersion() byte {
	return o.Version
}

func (o *CrossChainOutput) Validate() error {
	if o == nil {
		return errors.New("vote output payload is nil")
	}
	if o.Version > CrossChainOutputVersion {
		return errors.New("invalid vote version")
	}
	if o.TargetAddress == "" {
		return errors.New("target address is nil")
	}
	if o.TargetAmount <= 0 {
		return errors.New("invalid target amount")
	}

	return nil
}

func (o CrossChainOutput) String() string {
	return fmt.Sprint("Vote: {\n\t\t\t",
		"Version: ", o.Version, "\n\t\t\t",
		"TargetAddress: ", o.TargetAddress, "\n\t\t\t}")
}
