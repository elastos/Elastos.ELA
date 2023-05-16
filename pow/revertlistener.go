// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package pow

import (
	"time"

	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const CheckRevertToPOWInterval = time.Minute

func (pow *Service) ListenForRevert() {
	go func() {
		for {
			time.Sleep(CheckRevertToPOWInterval)
			if pow.chain.BestChain.Height < pow.chainParams.DPoSConfiguration.RevertToPOWStartHeight {
				continue
			}
			if pow.arbiters.IsInPOWMode() {
				continue
			}
			lastBlockTimestamp := int64(pow.arbiters.GetLastBlockTimestamp())
			localTimestamp := pow.chain.TimeSource.AdjustedTime().Unix()
			log.Info("ListenForRevert lastBlockTimestamp:", lastBlockTimestamp,
				"localTimestamp:", localTimestamp, "RevertToPOWNoBlockTime:", pow.chainParams.DPoSConfiguration.RevertToPOWNoBlockTime)
			if localTimestamp-lastBlockTimestamp < pow.chainParams.DPoSConfiguration.RevertToPOWNoBlockTime {
				continue
			}

			revertToPOWPayload := payload.RevertToPOW{
				Type:          payload.NoBlock,
				WorkingHeight: pow.chain.BestChain.Height + 1,
			}
			tx := functions.CreateTransaction(
				common2.TxVersion09,
				common2.RevertToPOW,
				payload.RevertToPOWVersion,
				&revertToPOWPayload,
				[]*common2.Attribute{},
				[]*common2.Input{},
				[]*common2.Output{},
				0,
				[]*program.Program{},
			)

			err := pow.txMemPool.AppendToTxPoolWithoutEvent(tx)
			if err != nil {
				log.Error("failed to append revertToPOW transaction to " +
					"transaction pool, err:" + err.Error())
			}
		}
	}()
}
