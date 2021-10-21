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

const WithdrawOutputVersion byte = 0x00

// CrossChainOutput defines the output payload for cross chain.
type Withdraw struct {
	Version                  byte
	GenesisBlockAddress      string
	SideChainTransactionHash common.Uint256
	TargetData               []byte
}

func (o *Withdraw) Data() []byte {
	return nil
}

func (o *Withdraw) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{byte(o.Version)}); err != nil {
		return err
	}

	if err := common.WriteVarString(w, o.GenesisBlockAddress); err != nil {
		return err
	}

	if err := o.SideChainTransactionHash.Serialize(w); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, o.TargetData); err != nil {
		return err
	}
	return nil
}

func (o *Withdraw) Deserialize(r io.Reader) error {
	version, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	o.Version = version[0]

	o.GenesisBlockAddress, err = common.ReadVarString(r)
	if err != nil {
		return err
	}

	if err = o.SideChainTransactionHash.Deserialize(r); err != nil {
		return err
	}

	o.TargetData, err = common.ReadVarBytes(r, MaxTargetDataSize, "target data")
	if err != nil {
		return err
	}
	return nil
}

func (o *Withdraw) GetVersion() byte {
	return o.Version
}

func (o *Withdraw) Validate() error {
	if o == nil {
		return errors.New("vote output payload is nil")
	}
	if o.Version > WithdrawOutputVersion {
		return errors.New("invalid vote version")
	}
	if o.GenesisBlockAddress == "" {
		return errors.New("target address is nil")
	}

	return nil
}

func (o Withdraw) String() string {
	return fmt.Sprint("Withdraw: {\n\t\t\t",
		"Version: ", o.Version, "\n\t\t\t",
		"GenesisBlockAddress: ", o.GenesisBlockAddress, "\n\t\t\t",
		"SideChainTransactionHash: ", o.SideChainTransactionHash.String(), "\n\t\t\t",
		"TargetData: ", string(o.TargetData), "\n\t\t\t}")
}
