// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DposV2ClaimRewardTransaction struct {
	BaseTransaction
}

func (t *DposV2ClaimRewardTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *DposV2ClaimRewardTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 1 {
		return errors.New("dposV2 claim reward transactions should have one and only one program")
	}
	return nil
}

func (t *DposV2ClaimRewardTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.DposV2ClaimReward:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *DposV2ClaimRewardTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *DposV2ClaimRewardTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	if t.parameters.BlockHeight < t.parameters.Config.DPoSV2StartHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not claim reward before dposv2startheight")), true
	}

	claimReward, ok := t.Payload().(*payload.DposV2ClaimReward)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload for dposV2claimReward")), true
	}

	pub := t.Programs()[0].Code[1 : len(t.Programs()[0].Code)-1]
	u168, err := contract.PublicKeyToStandardProgramHash(pub)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	addr, err := u168.ToAddress()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	claimAmount, ok := t.parameters.BlockChain.GetState().DposV2RewardInfo[addr]
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("no reward to claim for such adress")), true
	}

	if claimAmount < claimReward.Amount {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claim reward exceeded , max claim reward "+claimAmount.String())), true
	}

	if claimReward.Amount < t.parameters.Config.RealWithdrawSingleFee {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claim reward is less than RealWithdrawSingleFee")), true
	}

	err = t.checkClaimRewardSignature(pub, claimReward)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	return nil, true
}

func (t *DposV2ClaimRewardTransaction) checkClaimRewardSignature(pub []byte, claimReward *payload.DposV2ClaimReward) error {

	// check signature
	publicKey, err := crypto.DecodePoint(pub)
	if err != nil {
		return errors.New("invalid public key in payload")
	}
	signedBuf := new(bytes.Buffer)
	err = claimReward.SerializeUnsigned(signedBuf, payload.DposV2ClaimRewardVersion)
	if err != nil {
		return err
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), claimReward.Signature)
	if err != nil {
		return errors.New("invalid signature in payload")
	}
	return nil
}
