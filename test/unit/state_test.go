// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	state2 "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

func TestState_GetCandidatesRelated(t *testing.T) {
	keyFrame := *randomStateKeyFrame(5, true)
	state := state2.State{
		StateKeyFrame: keyFrame,
	}

	// get single candidate
	for k, v := range keyFrame.Candidates {
		v2 := state.GetCandidate(k)
		assert.True(t, candidateEqual(v, v2))

		v3 := state.GetCandidate(v.Info.CID)
		assert.True(t, candidateEqual(v, v3))
	}

	// get candidates
	candidates := state.GetAllCandidates()
	assert.Equal(t, 15, len(candidates))

	pending := state.GetCandidates(state2.Pending)
	assert.Equal(t, 5, len(pending))

	actives := state.GetCandidates(state2.Active)
	assert.Equal(t, 5, len(actives))

	cancels := state.GetCandidates(state2.Canceled)
	assert.Equal(t, 3, len(cancels))

	returns := state.GetCandidates(state2.Returned)
	assert.Equal(t, 2, len(returns))
}

func TestState_ExistCandidateRelated(t *testing.T) {
	keyFrame := *randomStateKeyFrame(5, true)
	state := state2.State{
		StateKeyFrame: keyFrame,
	}

	assert.False(t, state.ExistCandidate(make([]byte, 34)))
	assert.False(t, state.ExistCandidateByCID(common.Uint168{}))
	assert.False(t, state.ExistCandidateByNickname(""))

	for _, v := range keyFrame.Candidates {
		assert.True(t, state.ExistCandidate(v.Info.Code))
		assert.True(t, state.ExistCandidateByCID(v.Info.CID))
		assert.True(t, state.ExistCandidateByNickname(v.Info.NickName))
	}
}

func TestState_ProcessBlock_PendingUpdateThenCancel(t *testing.T) {
	cfg := &config.DefaultParams
	cfg.CRVotingStartHeight = 0
	currentHeight := uint32(1)
	committee := state2.NewCommittee(cfg)
	committee.RegisterFuncitons(&state2.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
		GetUTXO: func(programHash *common.Uint168) ([]*common2.UTXO, error) {
			return []*common2.UTXO{}, nil
		},
	})
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	code := getCodeFromStr(publicKeyStr1)
	cid := *getCID(code)
	nickname := randomString()

	assert.False(t, committee.GetState().ExistCandidate(code))
	assert.False(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))

	registerFuncs(committee.GetState())

	// register CR
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateRegisterCR(code, cid, nickname),
		},
	}, nil)
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.True(t, committee.GetState().ExistCandidateByNickname(nickname))
	candidate := committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Pending, candidate.State)

	// update pending CR
	nickname2 := randomString()
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateUpdateCR(code, cid, nickname2),
		},
	}, nil)
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))
	assert.True(t, committee.GetState().ExistCandidateByNickname(nickname2))
	candidate = committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Pending, candidate.State)

	//cancel pending CR
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname2))
	candidate = committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Canceled, candidate.State)
	assert.Equal(t, 0, len(committee.GetState().GetCandidates(state2.Pending)))
	assert.Equal(t, 1, len(committee.GetState().GetCandidates(state2.Canceled)))
}

func TestState_ProcessBlock_PendingActiveThenCancel(t *testing.T) {
	cfg := &config.DefaultParams
	cfg.CRVotingStartHeight = 0
	currentHeight := uint32(1)
	committee := state2.NewCommittee(cfg)
	committee.RegisterFuncitons(&state2.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
		GetUTXO: func(programHash *common.Uint168) ([]*common2.UTXO, error) {
			return []*common2.UTXO{}, nil
		},
	})
	nickname := randomString()
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	code := getCodeFromStr(publicKeyStr1)
	cid := *getCID(code)

	assert.False(t, committee.GetState().ExistCandidate(code))
	assert.False(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))

	registerFuncs(committee.GetState())

	// register CR
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateRegisterCR(code, cid, nickname),
		},
	}, nil)
	currentHeight++
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.True(t, committee.GetState().ExistCandidateByNickname(nickname))
	candidate := committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Pending, candidate.State)

	// register CR then after 6 block should be active State
	for i := 0; i < 5; i++ {
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil)
		currentHeight++
	}
	candidate = committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Active, candidate.State)

	// update active CR
	nickname2 := randomString()
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateUpdateCR(code, cid, nickname2),
		},
	}, nil)
	currentHeight++
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))
	assert.True(t, committee.GetState().ExistCandidateByNickname(nickname2))
	candidate = committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Active, candidate.State)

	// cancel active CR
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	assert.True(t, committee.GetState().ExistCandidate(code))
	assert.True(t, committee.GetState().ExistCandidateByCID(cid))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname))
	assert.False(t, committee.GetState().ExistCandidateByNickname(nickname2))
	candidate = committee.GetState().GetCandidate(cid)
	assert.Equal(t, state2.Canceled, candidate.State)
	assert.Equal(t, 0, len(committee.GetState().GetCandidates(state2.Pending)))
	assert.Equal(t, 1, len(committee.GetState().GetCandidates(state2.Canceled)))
}

func TestState_ProcessBlock_MixedCRProcessing(t *testing.T) {
	cfg := &config.DefaultParams
	cfg.CRVotingStartHeight = 0
	currentHeight := uint32(1)
	committee := state2.NewCommittee(cfg)
	committee.GetState().StateKeyFrame = *randomStateKeyFrame(5, true)
	committee.RegisterFuncitons(&state2.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
		GetUTXO: func(programHash *common.Uint168) ([]*common2.UTXO, error) {
			return []*common2.UTXO{}, nil
		},
	})
	registerFuncs(committee.GetState())

	assert.Equal(t, 15, len(committee.GetState().GetAllCandidates()))
	assert.Equal(t, 5, len(committee.GetState().GetCandidates(state2.Pending)))
	assert.Equal(t, 5, len(committee.GetState().GetCandidates(state2.Active)))
	assert.Equal(t, 3, len(committee.GetState().GetCandidates(state2.Canceled)))
	assert.Equal(t, 2, len(committee.GetState().GetCandidates(state2.Returned)))

	for i := 0; i < 10; i++ {
		code := randomBytes(34)
		nickname := randomString()
		cid := *randomUint168()

		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
			Transactions: []interfaces.Transaction{
				generateRegisterCR(code, cid, nickname),
			},
		}, nil)
		currentHeight++
	}
	assert.Equal(t, 25, len(committee.GetState().GetAllCandidates()))
	assert.Equal(t, 5, len(committee.GetState().GetCandidates(state2.Pending)))
	assert.Equal(t, 15, len(committee.GetState().GetCandidates(state2.Active)))
	assert.Equal(t, 3, len(committee.GetState().GetCandidates(state2.Canceled)))
	assert.Equal(t, 2, len(committee.GetState().GetCandidates(state2.Returned)))

	for i := 0; i < 5; i++ {
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil)
		currentHeight++
	}
	assert.Equal(t, 25, len(committee.GetState().GetAllCandidates()))
	assert.Equal(t, 0, len(committee.GetState().GetCandidates(state2.Pending)))
	assert.Equal(t, 20, len(committee.GetState().GetCandidates(state2.Active)))
	assert.Equal(t, 3, len(committee.GetState().GetCandidates(state2.Canceled)))
	assert.Equal(t, 2, len(committee.GetState().GetCandidates(state2.Returned)))
}

func TestState_ProcessBlock_VotingAndCancel(t *testing.T) {
	cfg := &config.DefaultParams
	cfg.CRVotingStartHeight = 0
	currentHeight := uint32(1)
	keyframe := randomStateKeyFrame(5, true)
	committee := state2.NewCommittee(cfg)
	committee.GetState().StateKeyFrame = *keyframe
	committee.RegisterFuncitons(&state2.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
		GetUTXO: func(programHash *common.Uint168) ([]*common2.UTXO, error) {
			return []*common2.UTXO{}, nil
		},
	})

	activeCIDs := make([][]byte, 0, 5)
	for k, v := range keyframe.Candidates {
		v.Votes = 0
		activeCIDs = append(activeCIDs, k.Bytes())
	}

	registerFuncs(committee.GetState())
	references := make(map[*common2.Input]common2.Output)
	committee.GetState().GetTxReference = func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error) {
		return references, nil
	}

	// vote for the active candidates
	voteTx := mockNewVoteTx(activeCIDs)
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{voteTx},
	}, nil)
	currentHeight++

	for i, v := range activeCIDs {
		did, _ := common.Uint168FromBytes(v)
		candidate := committee.GetState().GetCandidate(*did)
		assert.Equal(t, common.Fixed64((i+1)*10), candidate.Votes)
	}

	input := &common2.Input{
		Previous: *common2.NewOutPoint(voteTx.Hash(), uint16(0)),
	}
	references[input] = *voteTx.Outputs()[0]

	// cancel Votes the active candidates
	var txn []interfaces.Transaction
	tx := functions.CreateTransaction(
		0,
		common2.ActivateProducer,
		0,
		&payload.ActivateProducer{},
		[]*common2.Attribute{},
		[]*common2.Input{input},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	txn = append(txn, tx)
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: txn,
	}, nil)

	for _, v := range activeCIDs {
		did, _ := common.Uint168FromBytes(v)
		candidate := committee.GetState().GetCandidate(*did)
		assert.Equal(t, common.Fixed64(0), candidate.Votes)
	}
}

func TestState_ProcessBlock_DepositAndReturnDeposit(t *testing.T) {
	cfg := &config.DefaultParams
	cfg.CRVotingStartHeight = 0
	currentHeight := uint32(1)
	committee := state2.NewCommittee(cfg)
	committee.RegisterFuncitons(&state2.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
		GetUTXO: func(programHash *common.Uint168) ([]*common2.UTXO, error) {
			return []*common2.UTXO{}, nil
		},
	})
	registerFuncs(committee.GetState())

	_, pk, _ := crypto.GenerateKeyPair()
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	cid := *getCID(code)

	depositCont, _ := contract.CreateDepositContractByPubKey(pk)

	// register CR
	registerCRTx := functions.CreateTransaction(
		0,
		common2.RegisterCR,
		0,
		&payload.CRInfo{
			Code:     code,
			CID:      cid,
			NickName: randomString(),
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(6000 * 1e8),
			},
		},
		0,
		[]*program.Program{},
	)
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{registerCRTx},
	}, nil)
	currentHeight++
	candidate := committee.GetState().GetCandidate(cid)
	assert.Equal(t, common.Fixed64(5000*1e8),
		committee.GetState().GetDepositAmount(candidate.Info.CID))
	assert.Equal(t, common.Fixed64(6000*1e8),
		committee.GetState().GetTotalAmount(candidate.Info.CID))

	// deposit though normal tx

	tranferTx := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(1000 * 1e8),
			},
		},
		0,
		[]*program.Program{},
	)
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{tranferTx},
	}, nil)
	currentHeight++
	assert.Equal(t, common.Fixed64(5000*1e8),
		committee.GetState().GetDepositAmount(candidate.Info.CID))
	assert.Equal(t, common.Fixed64(7000*1e8),
		committee.GetState().GetTotalAmount(candidate.Info.CID))

	// cancel candidate
	for i := 0; i < 4; i++ {
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil)
		currentHeight++
	}
	assert.Equal(t, state2.Active, candidate.State)
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	cancelHeight := currentHeight
	currentHeight++
	for i := 0; i < 5; i++ {
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil)
		currentHeight++
	}
	assert.Equal(t, state2.Canceled, candidate.State)

	// reached the Height to return deposit amount.
	currentHeight = cancelHeight + committee.Params.CRDepositLockupBlocks
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil)

	// return deposit
	rdTx := generateReturnCRDeposit(code)
	rdTx.SetInputs([]*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  registerCRTx.Hash(),
				Index: 0,
			},
		},
		{
			Previous: common2.OutPoint{
				TxID:  tranferTx.Hash(),
				Index: 0,
			},
		},
	})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{rdTx},
	}, nil)
	committee.GetState().History.Commit(currentHeight)
	assert.Equal(t, common.Fixed64(0),
		committee.GetState().GetDepositAmount(candidate.Info.CID))
}

func mockNewVoteTx(cids [][]byte) interfaces.Transaction {
	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(cids))
	for i, cid := range cids {
		//code := getCode(common.BytesToHexString(pk))
		candidateVotes = append(candidateVotes,
			outputpayload.CandidateVotes{
				Candidate: cid,
				Votes:     common.Fixed64((i + 1) * 10)})
	}
	output := &common2.Output{
		Value: 100,
		Type:  common2.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: outputpayload.VoteProducerAndCRVersion,
			Contents: []outputpayload.VoteContent{
				{outputpayload.CRC, candidateVotes},
			},
		},
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{output},
		0,
		[]*program.Program{},
	)

	return txn
}

func generateReturnCRDeposit(code []byte) interfaces.Transaction {
	txn := functions.CreateTransaction(
		0,
		common2.ReturnCRDepositCoin,
		0,
		&payload.ReturnDepositCoin{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code: code,
		}},
	)
	return txn
}

func getCodeFromStr(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}
