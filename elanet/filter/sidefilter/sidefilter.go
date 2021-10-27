// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

/*
Side filter is a filter of for SideChain SPV module, it filters transactions
that will change DPOS producers state like RegisterProducer, CancelProducer etc.
and also the transactions related to the SideChain addresses.
*/

package sidefilter

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
	"github.com/elastos/Elastos.ELA/elanet/filter"
)

// Ensure Filter implements the TxFilter interface.
var _ filter.TxFilter = (*Filter)(nil)

// Filter defines the side filter instance, it implements the TxFilter
// interface.
type Filter struct {
	bloom.TxFilter
	state *state.State
}

// Load loads the transaction filter.
func (f *Filter) Load(filter []byte) error {
	return f.TxFilter.Load(filter)
}

// Add adds new data into filter.
func (f *Filter) Add(data []byte) error {
	return f.TxFilter.Add(data)
}

// MatchConfirmed returns if a confirmed (packed into a block) transaction
// matches the filter.
func (f *Filter) MatchConfirmed(tx *transactions.BaseTransaction) bool {
	return f.TxFilter.MatchConfirmed(tx) || f.state.IsDPOSTransaction(tx) ||
		tx.IsRevertToPOW() || tx.IsRevertToDPOS()
}

// MatchUnconfirmed returns if a unconfirmed (not packed into a block yet)
// transaction matches the filter.
func (f *Filter) MatchUnconfirmed(tx *transactions.BaseTransaction) bool {
	switch tx.TxType {
	case common2.IllegalProposalEvidence:
		fallthrough
	case common2.IllegalVoteEvidence:
		fallthrough
	case common2.IllegalBlockEvidence:
		fallthrough
	case common2.IllegalSidechainEvidence:
		fallthrough
	case common2.InactiveArbitrators:
		return true
	}
	return false
}

// New returns a new Filter instance.
func New(state *state.State) *Filter {
	return &Filter{state: state}
}
