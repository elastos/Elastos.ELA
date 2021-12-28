// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

type CancelVotes struct {
	ReferKeys []common.Uint256
}

func (p *CancelVotes) Data(version byte) []byte {

	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *CancelVotes) Serialize(w io.Writer, version byte) error {

	if err := common.WriteVarUint(w, uint64(len(p.ReferKeys))); err != nil {
		return err
	}
	for _, referKey := range p.ReferKeys {
		if err := referKey.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (p *CancelVotes) Deserialize(r io.Reader, version byte) error {

	keysCount, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	p.ReferKeys = make([]common.Uint256, 0)
	for i := uint64(0); i < keysCount; i++ {
		var referKey common.Uint256
		if err := referKey.Deserialize(r); err != nil {
			return err
		}
		p.ReferKeys = append(p.ReferKeys, referKey)
	}

	return nil
}
