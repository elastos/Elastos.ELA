// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

/*
CustomId filter is a filter of for SPV module, it filters transactions
NextTurnDPOSInfo and CustomId related transactions and also the transactions
related to the add addresses.
*/

package customidfilter

import (
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
	"github.com/elastos/Elastos.ELA/elanet/filter"
)

// Ensure Filter implements the TxFilter interface.
var _ filter.TxFilter = (*CustomIdFilter)(nil)

// Filter defines the side filter instance, it implements the TxFilter
// interface.
type CustomIdFilter struct {
	bloom.TxFilter
}

// Load loads the transaction filter.
func (f *CustomIdFilter) Load(filter []byte) error {
	return f.TxFilter.Load(filter)
}

// Add adds new data into filter.
func (f *CustomIdFilter) Add(data []byte) error {
	return f.TxFilter.Add(data)
}

// MatchConfirmed returns if a confirmed (packed into a block) transaction
// matches the filter.
func (f *CustomIdFilter) MatchConfirmed(tx *types.Transaction) bool {
	return f.TxFilter.MatchConfirmed(tx) || tx.IsNextTurnDPOSInfoTx() ||
		tx.IsCustomIDRelatedTx() || tx.IsRevertToPOW() || tx.IsRevertToDPOS()
}

// MatchUnconfirmed returns if a unconfirmed (not packed into a block yet)
// transaction matches the filter.
func (f *CustomIdFilter) MatchUnconfirmed(tx *types.Transaction) bool {
	return f.TxFilter.MatchUnconfirmed(tx)
}

// New returns a new CustomIdFilter instance.
func New() *CustomIdFilter {
	return &CustomIdFilter{}
}
