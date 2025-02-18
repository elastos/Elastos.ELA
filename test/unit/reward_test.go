// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	state2 "github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
}

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

	// set count of CR member
	config.DefaultParams = *config.GetDefaultParams()
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.DPoSConfiguration.NormalArbitratorsCount = 24

	ckpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, ckpManager)

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
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
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	var bestHeight uint32
	arbitrators, _ := state2.NewArbitrators(&config.DefaultParams,
		committee, nil, nil, nil, nil,
		nil, nil, nil, ckpManager)
	arbitrators.RegisterFunction(func() uint32 { return bestHeight },
		func() *common.Uint256 { return &common.Uint256{} },
		nil, nil)
	arbitrators.ChainParams.DPoSConfiguration.NoCRCDPOSNodeHeight = 2000000

	// Create 200 producers info.
	producers := make([]*payload.ProducerInfo, 200)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerKey:      randomPublicKey(),
			NodePublicKey: make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register 10 producers on one height.
	for i := 0; i < 20; i++ {
		txs := make([]interfaces.Transaction, 10)
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
		publicKeys[i] = p.OwnerKey
	}
	voteTX := mockVoteTx(publicKeys)
	arbitrators.ProcessBlock(mockBlock(23, voteTX), nil)
	for _, pk := range publicKeys {
		p := arbitrators.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.Votes()) {
			t.FailNow()
		}
	}

	arbitrators.ProcessBlock(mockBlock(arbitrators.ChainParams.PublicDPOSHeight-
		arbitrators.ChainParams.DPoSConfiguration.PreConnectOffset-1), nil)
	arbitrators.ProcessBlock(mockBlock(arbitrators.ChainParams.PublicDPOSHeight-1), nil)

	arbitrators.DutyIndex = 25
	arbitrators.ProcessBlock(mockBlock(1000000+72), nil)

	lastVote := common.Fixed64(0)
	for _, v := range arbitrators.NextReward.OwnerVotesInRound {
		if lastVote != 0 {
			assert.Equal(t, true, v == lastVote, "invalid reward")
		}
		lastVote = v
	}
	assert.Equal(t, 96, len(arbitrators.NextReward.OwnerVotesInRound),
		"invalid reward count")
}
