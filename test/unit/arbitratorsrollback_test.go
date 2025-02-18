// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"errors"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"

	"github.com/stretchr/testify/assert"
)

var abt *state.Arbiters
var abtList [][]byte

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = *config.GetDefaultParams()
}

func initArbiters() {
	arbitratorsStr := []string{
		"023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
		"030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
		"0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
		"03e281f89d85b3a7de177c240c4961cb5b1f2106f09daa42d15874a38bbeae85dd",
		"0393e823c2087ed30871cbea9fa5121fa932550821e9f3b17acef0e581971efab0",
	}

	abtList = make([][]byte, 0)
	for _, v := range arbitratorsStr {
		a, _ := common.HexStringToBytes(v)
		abtList = append(abtList, a)
	}
	ckpManager := checkpoint.NewManager(config.GetDefaultParams())
	activeNetParams := &config.DefaultParams
	activeNetParams.DPoSConfiguration.CRCArbiters = []string{
		"03e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b6",
		"038a1829b4b2bee784a99bebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca771",
	}
	bestHeight := uint32(0)

	abt, _ = state.NewArbitrators(activeNetParams, nil, nil,
		nil, nil, nil,
		nil, nil, nil, ckpManager)
	abt.RegisterFunction(func() uint32 { return bestHeight },
		func() *common.Uint256 { return &common.Uint256{} },
		nil, nil)
	abt.State = state.NewState(activeNetParams, nil, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
}

func checkPointEqual(first, second *state.CheckPoint) bool {
	if !stateKeyFrameEqual(&first.StateKeyFrame, &second.StateKeyFrame) {
		return false
	}

	if first.Height != second.Height || first.DutyIndex != second.DutyIndex ||
		first.CRCChangedHeight != second.CRCChangedHeight ||
		first.AccumulativeReward != second.AccumulativeReward ||
		first.FinalRoundChange != second.FinalRoundChange ||
		first.ClearingHeight != second.ClearingHeight ||
		len(first.NextCRCArbitersMap) != len(second.NextCRCArbitersMap) ||
		len(first.ArbitersRoundReward) != len(second.ArbitersRoundReward) ||
		len(first.IllegalBlocksPayloadHashes) !=
			len(second.IllegalBlocksPayloadHashes) {
		return false
	}

	//	rewardEqual
	if !arrayEqual(first.CurrentArbitrators, second.CurrentArbitrators) ||
		!arrayEqual(first.NextArbitrators, second.NextArbitrators) ||
		!arrayEqual(first.CurrentCandidates, second.CurrentCandidates) ||
		!arrayEqual(first.NextCandidates, second.NextCandidates) ||
		!rewardEqual(&first.CurrentReward, &second.CurrentReward) ||
		!rewardEqual(&first.NextReward, &second.NextReward) {
		return false
	}

	for k, v := range first.NextCRCArbitersMap {
		a, ok := second.NextCRCArbitersMap[k]
		if !ok {
			return false
		}
		if !arbiterMemberEqual(v, a) {
			return false
		}
	}

	for k, v := range first.ArbitersRoundReward {
		a, ok := second.ArbitersRoundReward[k]
		if !ok {
			return false
		}
		if v != a {
			return false
		}
	}

	for k, _ := range first.IllegalBlocksPayloadHashes {
		_, ok := second.IllegalBlocksPayloadHashes[k]
		if !ok {
			return false
		}
	}

	return true
}

func checkArbiterResult(t *testing.T, A, B, C, D *state.CheckPoint) {
	assert.Equal(t, true, checkPointEqual(A, C))
	assert.Equal(t, false, checkPointEqual(A, B))
	assert.Equal(t, true, checkPointEqual(B, D))
	assert.Equal(t, false, checkPointEqual(B, C))
}

func getRegisterProducerTx(ownerPublicKey, nodePublicKey []byte,
	nickName string) interfaces.Transaction {
	pk, _ := crypto.DecodePoint(ownerPublicKey)
	depositCont, _ := contract.CreateDepositContractByPubKey(pk)
	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		&payload.ProducerInfo{
			OwnerKey:      ownerPublicKey,
			NodePublicKey: nodePublicKey,
			NickName:      nickName,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(5000 * 1e8),
			},
		},
		0,
		[]*program.Program{},
	)
	return txn
}

func getVoteProducerTx(amount common.Fixed64,
	candidateVotes []outputpayload.CandidateVotes) interfaces.Transaction {

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       amount,
				OutputLock:  0,
				ProgramHash: *randomUint168(),
				Type:        common2.OTVote,
				Payload: &outputpayload.VoteOutput{
					Version: outputpayload.VoteProducerAndCRVersion,
					Contents: []outputpayload.VoteContent{{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: candidateVotes,
					},
					},
				},
			},
		},
		0,
		[]*program.Program{},
	)

	return txn
}

func getUpdateProducerTx(ownerPublicKey, nodePublicKey []byte,
	nickName string) interfaces.Transaction {
	txn := functions.CreateTransaction(
		0,
		common2.UpdateProducer,
		0,
		&payload.ProducerInfo{
			OwnerKey:      ownerPublicKey,
			NodePublicKey: nodePublicKey,
			NickName:      nickName,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	return txn
}

func getCancelProducer(publicKey []byte) interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CancelProducer,
		0,
		&payload.ProcessProducer{
			OwnerKey: publicKey,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return txn
}

func getReturnProducerDeposit(publicKey []byte, amount common.Fixed64) interfaces.Transaction {
	pk, _ := crypto.DecodePoint(publicKey)
	code, _ := contract.CreateStandardRedeemScript(pk)
	txn := functions.CreateTransaction(
		0,
		common2.ReturnDepositCoin,
		0,
		&payload.ReturnDepositCoin{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: amount},
		},
		0,
		[]*program.Program{
			{Code: code},
		},
	)
	return txn
}

func TestArbitrators_RollbackRegisterProducer(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}
	arbiterStateA := abt.Snapshot()
	assert.Equal(t, 0, len(abt.PendingProducers))

	// process
	abt.ProcessBlock(block1, nil)
	arbiterStateB := abt.Snapshot()
	assert.Equal(t, 4, len(abt.PendingProducers))

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(block1, nil)
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)

	for i := uint32(0); i < 4; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.PendingProducers))
	assert.Equal(t, 0, len(abt.ActivityProducers))
	arbiterStateA2 := abt.Snapshot()

	// process
	currentHeight++
	blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
	abt.ProcessBlock(blockEx, nil)
	arbiterStateB2 := abt.Snapshot()
	assert.Equal(t, 0, len(abt.PendingProducers))
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// rollback
	currentHeight--
	err = abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC2 := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(blockEx, nil)
	arbiterStateD2 := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA2, arbiterStateB2, arbiterStateC2, arbiterStateD2)
}

func TestArbitrators_RollbackVoteProducer(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
		})

	// process
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)
	arbiterStateA := abt.Snapshot()
	assert.Equal(t, common.Fixed64(5), abt.GetProducer(abtList[0]).Votes())

	currentHeight++
	updateProducerTx := getUpdateProducerTx(abtList[1], abtList[1], "node1")
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{updateProducerTx}}, nil)
	arbiterStateB := abt.Snapshot()

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{updateProducerTx}}, nil)
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)
}

func TestArbitrators_RollbackUpdateProducer(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
		})
	arbiterStateA := abt.Snapshot()

	// process
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)
	arbiterStateB := abt.Snapshot()
	assert.Equal(t, common.Fixed64(5), abt.GetProducer(abtList[0]).Votes())

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()
	assert.Equal(t, common.Fixed64(0), abt.GetProducer(abtList[0]).Votes())

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)
}

func TestArbitrators_RollbackCancelProducer(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// cancel producer
	cancelProducerTx := getCancelProducer(abtList[0])
	arbiterStateA := abt.Snapshot()

	// process
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{cancelProducerTx}}, nil)
	arbiterStateB := abt.Snapshot()
	assert.Equal(t, 3, len(abt.GetActiveProducers()))

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()
	assert.Equal(t, 4, len(abt.GetActiveProducers()))

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{cancelProducerTx}}, nil)
	arbiterStateD := abt.Snapshot()
	assert.Equal(t, 3, len(abt.GetActiveProducers()))

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)
}

func TestArbitrators_RollbackReturnProducerDeposit(t *testing.T) {
	initArbiters()

	register1 := getRegisterProducerTx(abtList[0], abtList[0], "p1")
	register2 := getRegisterProducerTx(abtList[1], abtList[1], "p2")
	register3 := getRegisterProducerTx(abtList[2], abtList[2], "p3")
	register4 := getRegisterProducerTx(abtList[3], abtList[3], "p4")

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			register1,
			register2,
			register3,
			register4,
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// cancel producer
	cancelProducerTx := getCancelProducer(abtList[0])

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{cancelProducerTx}}, nil)
	assert.Equal(t, 3, len(abt.GetActiveProducers()))

	// set get producer deposit amount function
	abt.GetProducerDepositAmount = func(programHash common.Uint168) (
		fixed64 common.Fixed64, err error) {
		producers := abt.GetAllProducers()
		for _, v := range producers {
			hash, _ := contract.PublicKeyToDepositProgramHash(
				v.Info().OwnerKey)
			if hash.IsEqual(programHash) {
				return v.DepositAmount(), nil
			}
		}

		return common.Fixed64(0), errors.New("not found producer")
	}

	assert.Equal(t, common.Fixed64(5000*1e8), abt.GetProducer(abtList[0]).DepositAmount())

	currentHeight += abt.ChainParams.CRConfiguration.DepositLockupBlocks
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{cancelProducerTx}}, nil)

	assert.Equal(t, common.Fixed64(0), abt.GetProducer(abtList[0]).DepositAmount())

	// return deposit
	returnDepositTx := getReturnProducerDeposit(abtList[0], 4999*1e8)
	returnDepositTx.SetInputs([]*common2.Input{{
		Previous: common2.OutPoint{
			TxID:  register1.Hash(),
			Index: 0,
		},
	}})
	arbiterStateA := abt.Snapshot()

	// process
	currentHeight = abt.ChainParams.CRConfiguration.CRVotingStartHeight
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{returnDepositTx}}, nil)
	assert.Equal(t, 1, len(abt.GetReturnedDepositProducers()))
	assert.Equal(t, common.Fixed64(0), abt.GetProducer(abtList[0]).DepositAmount())
	arbiterStateB := abt.Snapshot()

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(abt.GetReturnedDepositProducers()))
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{returnDepositTx}}, nil)
	assert.Equal(t, 1, len(abt.GetReturnedDepositProducers()))
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)
}

func TestArbitrators_RollbackLastBlockOfARound(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// set general arbiters count
	abt.ChainParams.DPoSConfiguration.NormalArbitratorsCount = 2
	arbiterStateA := abt.Snapshot()

	// update next arbiters
	currentHeight = abt.ChainParams.PublicDPOSHeight -
		abt.ChainParams.DPoSConfiguration.PreConnectOffset - 1
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateB := abt.Snapshot()

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)

	// process
	arbiterStateA2 := abt.Snapshot()
	currentHeight = abt.ChainParams.PublicDPOSHeight - 1
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateB2 := abt.Snapshot()

	// rollback
	currentHeight--
	err = abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC2 := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateD2 := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA2, arbiterStateB2, arbiterStateC2, arbiterStateD2)

	for i := 0; i < 3; i++ {
		currentHeight++
		abt.ProcessBlock(&types.Block{
			Header: common2.Header{Height: currentHeight}}, nil)
	}
	arbiterStateA3 := abt.Snapshot()

	// process
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateB3 := abt.Snapshot()

	// rollback
	currentHeight--
	err = abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC3 := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateD3 := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA3, arbiterStateB3, arbiterStateC3, arbiterStateD3)
}

func TestArbitrators_NextTurnDposInfoTX(t *testing.T) {
	initArbiters()
	abt.ChainParams = &config.DefaultParams
	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// set general arbiters count
	abt.ChainParams.DPoSConfiguration.NormalArbitratorsCount = 2
	//arbiterStateA := abt.Snapshot()

	// update next arbiters
	currentHeight = abt.ChainParams.PublicDPOSHeight -
		abt.ChainParams.DPoSConfiguration.PreConnectOffset - 1

	//here generate next turn dpos info tx
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	//arbiterStateB := abt.Snapshot()

	//rawTxStr := "091400022103e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b621038a1829b4b2bee784a99b" +
	//	"ebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca7710221023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19" +
	//	"bbc77a21030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be90000000000000000"
	//data, err2 := common.HexStringToBytes(rawTxStr)
	//if err2 != nil {
	//	fmt.Println("HexStringToBytes err2", err2)
	//	t.Fail()
	//}
	//reader2 := bytes.NewReader(data)
	//nextTurnDPOSInfoTx, err := functions.GetTransactionByBytes(reader2)
	//if err != nil {
	//	fmt.Println("invalid txn2")
	//	t.Fail()
	//}
	//err2 = nextTurnDPOSInfoTx.Deserialize(reader2)
	//if err2 != nil {
	//	fmt.Println("txn2.Deserialize err2", err2)
	//	t.Fail()
	//}
	//
	//abt.ProcessBlock(&types.Block{
	//	Header:       common2.Header{Height: currentHeight},
	//	Transactions: []interfaces.Transaction{nextTurnDPOSInfoTx}}, nil)
	//
	//currentHeight++
	//abt.ProcessBlock(&types.Block{
	//	Header: common2.Header{Height: currentHeight}}, nil)
}

func TestArbitrators_RollbackRewardBlock(t *testing.T) {
	initArbiters()

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			getRegisterProducerTx(abtList[0], abtList[0], "p1"),
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// set general arbiters count
	abt.ChainParams.DPoSConfiguration.NormalArbitratorsCount = 2

	// preConnect
	currentHeight = abt.ChainParams.PublicDPOSHeight -
		abt.ChainParams.DPoSConfiguration.PreConnectOffset - 1
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)

	currentHeight = abt.ChainParams.PublicDPOSHeight - 1
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)

	for i := 0; i < 4; i++ {
		currentHeight++
		abt.ProcessBlock(&types.Block{
			Header: common2.Header{Height: currentHeight}}, nil)
	}
	arbiterStateA := abt.Snapshot()

	// process reward block
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateB := abt.Snapshot()

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)

	for i := 0; i < 3; i++ {
		currentHeight++
		abt.ProcessBlock(&types.Block{
			Header: common2.Header{Height: currentHeight}}, nil)
	}
	arbiterStateA2 := abt.Snapshot()

	// process reward block
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateB2 := abt.Snapshot()

	// rollback
	currentHeight--
	err = abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	arbiterStateC2 := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	arbiterStateD2 := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA2, arbiterStateB2, arbiterStateC2, arbiterStateD2)
}

func TestArbitrators_RollbackMultipleTransactions(t *testing.T) {
	initArbiters()
	register1 := getRegisterProducerTx(abtList[0], abtList[0], "p1")

	currentHeight := abt.ChainParams.VoteStartHeight
	block1 := &types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			register1,
			getRegisterProducerTx(abtList[1], abtList[1], "p2"),
			getRegisterProducerTx(abtList[2], abtList[2], "p3"),
			getRegisterProducerTx(abtList[3], abtList[3], "p4"),
		},
	}

	abt.ProcessBlock(block1, nil)

	for i := uint32(0); i < 5; i++ {
		currentHeight++
		blockEx := &types.Block{Header: common2.Header{Height: currentHeight}}
		abt.ProcessBlock(blockEx, nil)
	}
	assert.Equal(t, 4, len(abt.ActivityProducers))

	// vote producer
	voteProducerTx := getVoteProducerTx(10,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[0], Votes: 5},
			{Candidate: abtList[1], Votes: 4},
			{Candidate: abtList[2], Votes: 3},
			{Candidate: abtList[3], Votes: 2},
		})

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteProducerTx}}, nil)

	// cancel producer
	cancelProducerTx := getCancelProducer(abtList[0])

	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{cancelProducerTx}}, nil)
	assert.Equal(t, 3, len(abt.GetActiveProducers()))

	// set get producer deposit amount function
	abt.GetProducerDepositAmount = func(programHash common.Uint168) (
		fixed64 common.Fixed64, err error) {
		producers := abt.GetAllProducers()
		for _, v := range producers {
			hash, _ := contract.PublicKeyToDepositProgramHash(
				v.Info().OwnerKey)
			if hash.IsEqual(programHash) {
				return v.DepositAmount(), nil
			}
		}

		return common.Fixed64(0), errors.New("not found producer")
	}

	registerProducerTx2 := getRegisterProducerTx(abtList[4], abtList[4], "p5")
	voteProducerTx2 := getVoteProducerTx(2,
		[]outputpayload.CandidateVotes{
			{Candidate: abtList[1], Votes: 1},
		})
	updateProducerTx2 := getUpdateProducerTx(abtList[1], abtList[1], "node1")
	cancelProducerTx2 := getCancelProducer(abtList[2])
	returnDepositTx2 := getReturnProducerDeposit(abtList[0], 4999*1e8)
	returnDepositTx2.SetInputs([]*common2.Input{{
		Previous: common2.OutPoint{
			TxID:  register1.Hash(),
			Index: 0,
		},
	}})
	assert.Equal(t, common.Fixed64(5000*1e8), abt.GetProducer(abtList[0]).DepositAmount())

	arbiterStateA := abt.Snapshot()

	// process
	currentHeight = abt.ChainParams.CRConfiguration.CRVotingStartHeight
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			registerProducerTx2,
			voteProducerTx2,
			updateProducerTx2,
			cancelProducerTx2,
			returnDepositTx2,
		}}, nil)
	assert.Equal(t, 2, len(abt.GetActiveProducers()))
	assert.Equal(t, 1, len(abt.GetReturnedDepositProducers()))
	assert.Equal(t, common.Fixed64(0), abt.GetProducer(abtList[0]).TotalAmount())
	arbiterStateB := abt.Snapshot()

	// rollback
	currentHeight--
	err := abt.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(abt.GetActiveProducers()))
	assert.Equal(t, 0, len(abt.GetReturnedDepositProducers()))
	arbiterStateC := abt.Snapshot()

	// reprocess
	currentHeight++
	abt.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			registerProducerTx2,
			voteProducerTx2,
			updateProducerTx2,
			cancelProducerTx2,
			returnDepositTx2,
		}}, nil)
	assert.Equal(t, 1, len(abt.GetReturnedDepositProducers()))
	arbiterStateD := abt.Snapshot()

	checkArbiterResult(t, arbiterStateA, arbiterStateB, arbiterStateC, arbiterStateD)
}
