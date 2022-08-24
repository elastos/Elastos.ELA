// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

/*
   returnsidechaindepositcoin filter is a filter of for SPV module, it filters transactions
   ReturnSideChainDepositCoin and CustomId related transactions and also the transactions
   related to the add addresses.
*/

package returnsidechaindepositcoinfilter

import (
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
	"github.com/elastos/Elastos.ELA/elanet/filter"
)

// Ensure Filter implements the TxFilter interface.
var _ filter.TxFilter = (*ReturnSidechainDepositCoinFilter)(nil)

// Filter defines the side filter instance, it implements the TxFilter
// interface.
type ReturnSidechainDepositCoinFilter struct {
	bloom.TxFilter
}

// Load loads the transaction filter.
func (f *ReturnSidechainDepositCoinFilter) Load(filter []byte) error {
	return f.TxFilter.Load(filter)
}

// Add adds new data into filter.
func (f *ReturnSidechainDepositCoinFilter) Add(data []byte) error {
	return f.TxFilter.Add(data)
}

// MatchConfirmed returns if a confirmed (packed into a block) transaction
// matches the filter.
func (f *ReturnSidechainDepositCoinFilter) MatchConfirmed(tx interfaces.Transaction) bool {
	return f.TxFilter.MatchConfirmed(tx) || tx.IsNextTurnDPOSInfoTx() ||
		tx.IsCustomIDRelatedTx() || tx.IsRevertToPOW() || tx.IsRevertToDPOS() || tx.IsReturnSideChainDepositCoinTx()
}

// MatchUnconfirmed returns if a unconfirmed (not packed into a block yet)
// transaction matches the filter.
func (f *ReturnSidechainDepositCoinFilter) MatchUnconfirmed(tx interfaces.Transaction) bool {
	return f.TxFilter.MatchUnconfirmed(tx)
}

// New returns a new ReturnSidechainDepositCoinFilter instance.
func New() *ReturnSidechainDepositCoinFilter {
	return &ReturnSidechainDepositCoinFilter{}
}
