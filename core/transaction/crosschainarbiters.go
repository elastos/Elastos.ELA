// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/crypto"
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

		var arbitersCount int
		var minCount uint32
		if height >= chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight {
			for _, arbiter := range blockchain.DefaultLedger.Arbitrators.GetArbitrators() {
				if arbiter.IsNormal {
					arbitersCount++
				}
			}
			minCount = uint32(chainParams.DPoSConfiguration.NormalArbitratorsCount) + 1
		} else {
			for _, arbiter := range blockchain.DefaultLedger.Arbitrators.GetCRCArbiters() {
				if arbiter.IsNormal {
					arbitersCount++
				}
			}
			minCount = chainParams.CRConfiguration.CRAgreementCount
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
