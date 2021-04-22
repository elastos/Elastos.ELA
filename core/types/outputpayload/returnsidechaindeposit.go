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

const ReturnSideChainDepositVersion byte = 0x00

// CrossChainOutput defines the output payload for cross chain.
type ReturnSideChainDeposit struct {
	Version                byte
	GenesisBlockAddress    string
	DepositTransactionHash common.Uint256
}

func (o *ReturnSideChainDeposit) Data() []byte {
	return nil
}

func (o *ReturnSideChainDeposit) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{byte(o.Version)}); err != nil {
		return err
	}

	if err := common.WriteVarString(w, o.GenesisBlockAddress); err != nil {
		return err
	}

	if err := o.DepositTransactionHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DepositTxs")
	}

	return nil
}

func (o *ReturnSideChainDeposit) Deserialize(r io.Reader) error {
	version, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	o.Version = version[0]

	o.GenesisBlockAddress, err = common.ReadVarString(r)
	if err != nil {
		return err
	}

	if err := o.DepositTransactionHash.Deserialize(r); err != nil {
		return err
	}

	return nil
}

func (o *ReturnSideChainDeposit) GetVersion() byte {
	return o.Version
}

func (o *ReturnSideChainDeposit) Validate() error {
	if o == nil {
		return errors.New("vote output payload is nil")
	}
	if o.Version > ReturnSideChainDepositVersion {
		return errors.New("invalid vote version")
	}
	if o.GenesisBlockAddress == "" {
		return errors.New("target address is nil")
	}

	return nil
}

func (o ReturnSideChainDeposit) String() string {
	return fmt.Sprint("Withdraw: {\n\t\t\t",
		"Version: ", o.Version, "\n\t\t\t",
		"GenesisBlockAddress: ", o.GenesisBlockAddress, "\n\t\t\t",
		"DepositTransactionHash: ", o.DepositTransactionHash.String(), "\n\t\t\t}")
}
