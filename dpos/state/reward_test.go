// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/stretchr/testify/assert"
)

func TestCommittee_ChangeCommitteeReward(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	// new committee
	committee := state.NewCommittee(&config.DefaultParams)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.CRCArbiters = cfg.CRCArbiters[0:2]
	cfg.CRMemberCount = 2
	abt.chainParams.GeneralArbiters = 24

	// avoid getting UTXOs from database
	currentHeight := cfg.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: currentHeight,
		},
		Transactions: []*types.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: currentHeight,
		},
		Transactions: []*types.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes())

	// end first voting period
	currentHeight = cfg.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: types.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetAllMembers()))

	var bestHeight uint32
	arbitrators, _ := NewArbitrators(&config.DefaultParams,
		committee, nil, nil, nil, nil, nil)
	arbitrators.RegisterFunction(func() uint32 { return bestHeight },
		nil, nil)

	// Create 200 producers info.
	producers := make([]*payload.ProducerInfo, 200)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register 10 producers on one height.
	for i := 0; i < 20; i++ {
		txs := make([]*types.Transaction, 10)
		for i, p := range producers[i*10 : (i+1)*10] {
			txs[i] = mockRegisterProducerTx(p)
		}
		arbitrators.ProcessBlock(mockBlock(uint32(i+1), txs...), nil)
	}
	// at this point, we have 50 pending, 50 active and 100 in total producers.
	if !assert.Equal(t, 50, len(arbitrators.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 150, len(arbitrators.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 200, len(arbitrators.GetProducers())) {
		t.FailNow()
	}

	// Vote 140 producers.
	publicKeys := make([][]byte, 140)
	for i, p := range producers[10:150] {
		publicKeys[i] = p.OwnerPublicKey
	}
	voteTX := mockVoteTx(publicKeys)
	arbitrators.ProcessBlock(mockBlock(23, voteTX), nil)
	for _, pk := range publicKeys {
		p := arbitrators.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.Votes()) {
			t.FailNow()
		}
	}

	arbitrators.ProcessBlock(mockBlock(arbitrators.chainParams.PublicDPOSHeight-
		arbitrators.chainParams.PreConnectOffset-1), nil)
	arbitrators.ProcessBlock(mockBlock(arbitrators.chainParams.PublicDPOSHeight-1), nil)

	arbitrators.dutyIndex = 25
	arbitrators.ProcessBlock(mockBlock(1000000+72), nil)

	lastVote := common.Fixed64(0)
	for _, v := range arbitrators.NextReward.OwnerVotesInRound {
		if lastVote != 0 {
			assert.Equal(t, true, v == lastVote, "invalid reward")
		}
		lastVote = v
	}
	assert.Equal(t, 98, len(arbitrators.NextReward.OwnerVotesInRound),
		"invalid reward count")
}

func getCIDByPublicKeyStr(publicKey string) *common.Uint168 {
	code1 := getCodeByPubKeyStr(publicKey)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	return ct1.ToProgramHash()
}

func getDIDByPublicKey(publicKey string) *common.Uint168 {
	code1 := getCodeByPubKeyStr(publicKey)

	ct1, _ := contract.CreateCRIDContractByCode(code1)
	return ct1.ToProgramHash()
}

func getCodeByPubKeyStr(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}

func getRegisterCRTx(publicKeyStr, privateKeyStr, nickName string) *types.Transaction {
	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	cid1, _ := getCIDByCode(code1)
	did1, _ := getDIDByCode(code1)
	hash1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)

	txn := new(types.Transaction)
	txn.TxType = types.RegisterCR
	txn.Version = types.TxVersion09
	crInfoPayload := &payload.CRInfo{
		Code:     code1,
		CID:      *cid1,
		DID:      *did1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1
	txn.Payload = crInfoPayload

	txn.Programs = []*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn.Outputs = []*types.Output{&types.Output{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}}
	return txn
}

func getCIDByCode(code []byte) (*common.Uint168, error) {
	ct1, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return nil, err
	}
	return ct1.ToProgramHash(), err
}

func getDIDByCode(code []byte) (*common.Uint168, error) {
	didCode := make([]byte, len(code))
	copy(didCode, code)
	didCode = append(didCode[:len(code)-1], common.DID)
	ct1, err := contract.CreateCRIDContractByCode(didCode)
	if err != nil {
		return nil, err
	}
	return ct1.ToProgramHash(), err
}

func getVoteCRTx(amount common.Fixed64,
	candidateVotes []outputpayload.CandidateVotes) *types.Transaction {
	return &types.Transaction{
		Version: 0x09,
		TxType:  types.TransferAsset,
		Outputs: []*types.Output{
			{
				AssetID:     common.Uint256{},
				Value:       amount,
				OutputLock:  0,
				ProgramHash: common.Uint168{123},
				Type:        types.OTVote,
				Payload: &outputpayload.VoteOutput{
					Version: outputpayload.VoteProducerAndCRVersion,
					Contents: []outputpayload.VoteContent{
						outputpayload.VoteContent{
							VoteType:       outputpayload.CRC,
							CandidateVotes: candidateVotes,
						},
					},
				},
			},
		},
	}
}
