// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"io"
)

type Stake struct {
}

func (p *Stake) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *Stake) Serialize(w io.Writer, version byte) error {
	return nil
}

func (p *Stake) Deserialize(r io.Reader, version byte) error {
	return nil
}
