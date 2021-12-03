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
	"github.com/elastos/Elastos.ELA/crypto"
	"math"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DposV2ClaimRewardTransaction struct {
	BaseTransaction
}

func (t *DposV2ClaimRewardTransaction) CheckTransactionInput() error {
	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *DposV2ClaimRewardTransaction) CheckTransactionOutput() error {

	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(t.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *DposV2ClaimRewardTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
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
	return true
}

func (t *DposV2ClaimRewardTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	if t.parameters.BlockHeight < t.parameters.Config.DposV2StartHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not claim reward before dposv2startheight")), false
	}

	claimReward, ok := t.Payload().(*payload.DposV2ClaimReward)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload for dposV2claimReward")), false
	}
	if len(t.Inputs()) != 0 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("inputs must be zero")), false
	}

	if len(t.Outputs()) != 0 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("outputs must be zero")), false
	}

	pub := t.Programs()[0].Code[1 : len(t.Programs()[0].Code)-1]
	u168, err := contract.PublicKeyToStandardProgramHash(pub)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), false
	}
	addr, err := u168.ToAddress()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), false
	}
	claimAmount, ok := t.parameters.BlockChain.GetState().DposV2RewardInfo[addr]
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("no reward to claim for such adress")), false
	}

	if claimAmount < claimReward.Amount {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claim reward exceeded , max claim reward "+claimAmount.String())), false
	}

	err = t.checkClaimRewardSignature(pub, claimReward)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), false
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
