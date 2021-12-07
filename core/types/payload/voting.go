// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
)

type Voting struct {
	Vote outputpayload.VoteOutput
}

func (p *Voting) Data(version byte) []byte {

	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *Voting) Serialize(w io.Writer, version byte) error {
	if err := p.Vote.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *Voting) Deserialize(r io.Reader, version byte) error {
	if err := p.Vote.Deserialize(r); err != nil {
		return err
	}

	return nil
}
