// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package pow

import (
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"time"
)

const CheckRevertToPOWInterval = time.Minute

func (pow *Service) ListenForRevert() {
	go func() {
		for {
			time.Sleep(CheckRevertToPOWInterval)

			if pow.arbiters.IsInPowMode() {
				continue
			}

			lastBlockTimestamp := int64(pow.arbiters.GetLastBlockTimestamp())
			localTimestamp := pow.chain.TimeSource.AdjustedTime().Unix()
			if localTimestamp-lastBlockTimestamp < pow.chainParams.RevertToPOWNoBlockTime {
				continue
			}

			revertToPOWPayload := payload.RevertToPOW{
				StartPOWBlockHeight: pow.chain.BestChain.Height,
			}
			tx := &types.Transaction{
				Version:        types.TxVersion09,
				TxType:         types.RevertToPOW,
				PayloadVersion: payload.RevertToPOWVersion,
				Payload:        &revertToPOWPayload,
				Attributes:     []*types.Attribute{},
				Programs:       []*program.Program{},
				LockTime:       0,
			}
			err := pow.txMemPool.AppendToTxPool(tx)
			if err != nil {
				log.Error("failed to append revertToPOW transaction to " +
					"transaction pool, err:" + err.Error())
			}
		}
	}()
}
