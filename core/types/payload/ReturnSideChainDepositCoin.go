// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"io"
)

const ReturnSideChainDepositCoinVersion byte = 0x00

type ReturnSideChainDepositCoin struct {
}

func (s *ReturnSideChainDepositCoin) Data(version byte) []byte {
	return nil
}

func (s *ReturnSideChainDepositCoin) Serialize(w io.Writer, version byte) error {
	return nil
}

func (s *ReturnSideChainDepositCoin) Deserialize(r io.Reader, version byte) error {
	return nil
}
