// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package transaction

import (
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

// checkCrossChainArbiterPrograms ensures each CrossChain witness uses the
// current arbitrator set and satisfies the active signature threshold.
func checkCrossChainArbiterPrograms(txn interfaces.Transaction, height uint32,
	chainParams *config.Configuration) error {
	if len(txn.Programs()) == 0 {
		return errors.New("CrossChain transaction has no programs")
	}

	for _, p := range txn.Programs() {
		publicKeys, m, n, err := crypto.ParseCrossChainScriptV1(p.Code)
		if err != nil {
			return err
		}

		var arbiters []*state.ArbiterInfo
		var minCount uint32
		if height >= chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetArbitrators()
			minCount = uint32(chainParams.DPoSConfiguration.NormalArbitratorsCount) + 1
		} else {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetCRCArbiters()
			minCount = chainParams.CRConfiguration.CRAgreementCount
		}

		var arbitersCount int
		for _, arbiter := range arbiters {
			if arbiter.IsNormal {
				arbitersCount++
			}
		}
		if n != arbitersCount {
			return errors.New("invalid arbiters total count in code")
		}
		if m < int(minCount) {
			return errors.New("invalid arbiters sign count in code")
		}
		if err := checkCrossChainArbitrators(publicKeys); err != nil {
			return err
		}
	}

	return nil
}
