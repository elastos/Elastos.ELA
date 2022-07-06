// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

// CandidateState defines states during a CR candidates lifetime
type CandidateState byte

const (
	// Pending indicates the producer is just registered and didn't get 6
	// confirmations yet.
	Pending CandidateState = iota

	// Active indicates the CR is registered and confirmed by more than
	// 6 blocks.
	Active

	// Canceled indicates the CR was canceled.
	Canceled

	// Returned indicates the CR has canceled and deposit returned.
	Returned
)

// candidateStateStrings is a array of CR states back to their constant
// names for pretty printing.
var candidateStateStrings = []string{"Pending", "Active", "Canceled",
	"Returned", "Impeached"}

func (ps CandidateState) String() string {
	if int(ps) < len(candidateStateStrings) {
		return candidateStateStrings[ps]
	}
	return fmt.Sprintf("CandidateState-%d", ps)
}

// Candidate defines information about CR candidates during the CR vote period
type Candidate struct {
	Info           payload.CRInfo
	State          CandidateState
	Votes          common.Fixed64
	RegisterHeight uint32
	CancelHeight   uint32
	DepositHash    common.Uint168
}

func (c *Candidate) Serialize(w io.Writer) (err error) {
	if err = c.Info.SerializeUnsigned(w, payload.CRInfoDIDVersion); err != nil {
		return
	}

	if err = common.WriteUint8(w, uint8(c.State)); err != nil {
		return
	}

	if err = common.WriteUint64(w, uint64(c.Votes)); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.RegisterHeight); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.CancelHeight); err != nil {
		return
	}

	return c.DepositHash.Serialize(w)
}

func (c *Candidate) Deserialize(r io.Reader) (err error) {
	if err = c.Info.DeserializeUnsigned(r, payload.CRInfoDIDVersion); err != nil {
		return
	}

	var state uint8
	if state, err = common.ReadUint8(r); err != nil {
		return
	}
	c.State = CandidateState(state)

	var votes uint64
	if votes, err = common.ReadUint64(r); err != nil {
		return
	}
	c.Votes = common.Fixed64(votes)

	if c.RegisterHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	c.CancelHeight, err = common.ReadUint32(r)

	return c.DepositHash.Deserialize(r)
}
