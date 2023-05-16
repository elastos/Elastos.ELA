// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package msg

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p"
)

// Ensure Ping implement p2p.Message interface.
var _ p2p.Message = (*Daddr)(nil)

// Addr represents a DPoS network address for connect to a peer.
type Daddr struct {
	Addr string
}

func NewDaddr(addr string) *Daddr {
	return &Daddr{Addr: addr}
}

func (msg *Daddr) CMD() string {
	return p2p.CmdDAddr
}

func (msg *Daddr) MaxLength() uint32 {
	return maxHostLength * 2
}

func (msg *Daddr) Serialize(w io.Writer) error {
	return common.WriteVarString(w, msg.Addr)
}

func (msg *Daddr) Deserialize(r io.Reader) error {
	var err error
	msg.Addr, err = common.ReadVarString(r)
	return err
}

func (msg *Daddr) String() string {
	return msg.Addr
}

func (msg *Daddr) Network() string {
	return "tcp"
}
