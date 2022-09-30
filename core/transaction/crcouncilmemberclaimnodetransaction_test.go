// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCRCouncilMemberClaimNodeTransaction() {

	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	//publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	//publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	//errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	//errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)
	did := randomUint168()
	{
		claimPayload := &payload.CRCouncilMemberClaimNode{
			NodePublicKey: publicKey1,
		}

		programs := []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn := functions.CreateTransaction(
			0,
			common2.CRCouncilMemberClaimNode,
			payload.CurrentCRClaimDPoSNodeVersion,
			claimPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:CRCouncilMemberClaimNode must during election period")

		s.Chain.GetCRCommittee().InElectionPeriod = true
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:the originator must be members")

		s.Chain.BestChain.Height = 20000000
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:the originator must be members")

		s.Chain.GetCRCommittee().Members[*did] = &state.CRMember{}
		s.Chain.GetCRCommittee().ClaimedDPoSKeys[hex.EncodeToString(publicKey1)] = struct{}{}
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:producer already registered")

		s.Chain.GetState().NodeOwnerKeys[hex.EncodeToString(publicKey1)] = randomString()
		s.Chain.GetCRCommittee().ClaimedDPoSKeys = make(map[string]struct{}, 0)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:producer already registered")
	}

	{
		claimPayload := &payload.CRCouncilMemberClaimNode{
			NodePublicKey: publicKey1,
		}

		programs := []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn := functions.CreateTransaction(
			0,
			common2.CRCouncilMemberClaimNode,
			payload.NextCRClaimDPoSNodeVersion,
			claimPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.Chain.BestChain.Height = 20000000
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:producer already registered")

		s.Chain.GetCRCommittee().NextClaimedDPoSKeys[hex.EncodeToString(publicKey1)] = struct{}{}
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:producer already registered")

		s.Chain.GetCRCommittee().NextMembers[*did] = &state.CRMember{
			MemberState: state.MemberIllegal,
		}
		s.Chain.GetCRCommittee().NextClaimedDPoSKeys = make(map[string]struct{}, 0)
		s.Chain.GetState().NodeOwnerKeys = make(map[string]string, 0)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:CR Council Member should be an elected or inactive CR members")

		s.Chain.GetCRCommittee().NextMembers[*did] = &state.CRMember{
			MemberState:   state.MemberElected,
			DPOSPublicKey: publicKey1,
		}
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:NodePublicKey is the same as crMember.DPOSPublicKey")

		claimPayload = &payload.CRCouncilMemberClaimNode{
			NodePublicKey: randomBytes(22),
		}
		txn.SetPayload(claimPayload)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid operating public key")

		code := getCodeByPubKeyStr(publicKeyStr1)
		s.Chain.GetCRCommittee().NextMembers[*did] = &state.CRMember{
			MemberState:   state.MemberElected,
			DPOSPublicKey: randomBytes(32),
			Info: payload.CRInfo{
				Code: code,
			},
		}
		claimPayload = &payload.CRCouncilMemberClaimNode{
			NodePublicKey: publicKey1,
		}
		buf := new(bytes.Buffer)
		claimPayload.SerializeUnsigned(buf, 0)
		sig, _ := crypto.Sign(privateKey1, buf.Bytes())
		claimPayload.CRCouncilCommitteeSignature = sig
		txn.SetPayload(claimPayload)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}
