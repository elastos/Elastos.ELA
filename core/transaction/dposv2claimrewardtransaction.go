// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/vm"
)

type DPoSV2ClaimRewardTransaction struct {
	BaseTransaction
}

func (t *DPoSV2ClaimRewardTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *DPoSV2ClaimRewardTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 1 {
		return errors.New("dposV2 claim reward transactions should have one and only one program")
	}

	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}
	// Check programs
	for _, p := range t.Programs() {
		if p.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if len(p.Code) < program.MinProgramCodeSize {
			return fmt.Errorf("invalid program code size")
		}
		if p.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}
	return nil
}

func (t *DPoSV2ClaimRewardTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.DPoSV2ClaimReward:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *DPoSV2ClaimRewardTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *DPoSV2ClaimRewardTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	if t.parameters.BlockHeight < t.parameters.Config.DPoSV2StartHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not claim reward before dposv2startheight")), true
	}

	claimReward, ok := t.Payload().(*payload.DPoSV2ClaimReward)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload for dposV2claimReward")), true
	}

	version := t.PayloadVersion()
	var code []byte
	if version == payload.DposV2ClaimRewardVersionV0 {
		code = claimReward.Code
	} else if version == payload.DposV2ClaimRewardVersionV1 {
		code = t.Programs()[0].Code
	} else {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload version")), true
	}

	addr, err := utils.GetStakeAddressByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("Programs code to address error")), true
	}
	claimAmount := t.parameters.BlockChain.GetState().GetDPoSV2RewardInfo(addr)
	if claimAmount == 0 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("no reward to claim for such address")), true
	}

	if claimAmount < claimReward.Value {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claim reward exceeded , max claim reward "+claimAmount.String()+"current:"+claimAmount.String())), true
	}

	if claimReward.Value <= t.parameters.Config.CRConfiguration.RealWithdrawSingleFee {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claim reward should be bigger than RealWithdrawSingleFee")), true
	}

	if version == payload.DposV2ClaimRewardVersionV0 {
		signedBuf := new(bytes.Buffer)
		err = claimReward.SerializeUnsigned(signedBuf, payload.DposV2ClaimRewardVersionV0)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("claimReward Serialize error")), true
		}
		err = t.checkClaimRewardSignature(code, claimReward.Signature, signedBuf.Bytes())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	}

	return nil, false
}

func getParameterBySignature(signature []byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signature)))
	buf.Write(signature)
	return buf.Bytes()
}

func (t *DPoSV2ClaimRewardTransaction) checkClaimRewardSignature(code []byte, signature []byte, data []byte) error {
	signType, err := crypto.GetScriptType(code)
	if err != nil {
		return errors.New("invalid code")
	}
	if signType == vm.CHECKSIG {
		// check code and signature
		if err := blockchain.CheckStandardSignature(program.Program{
			Code:      code,
			Parameter: getParameterBySignature(signature),
		}, data); err != nil {
			return err
		}
	} else if signType == vm.CHECKMULTISIG {
		// check code and signature
		if err := crypto.CheckMultiSigSignatures(program.Program{
			Code:      code,
			Parameter: signature,
		}, data); err != nil {
			return err
		}
	} else {
		return errors.New("invalid code type")
	}

	return nil
}
