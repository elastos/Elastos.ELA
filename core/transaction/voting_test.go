package transaction

import (
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"path/filepath"
)

func (s *txValidatorTestSuite) TestCheckDposV2VoteProducerOutput() {
	// 1. Generate a vote output v0
	publicKeyStr1 := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd47e944507292ea08dd"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	referKey := randomUint256()
	outputs1 := []*payload.Voting{
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: 0x05,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     -100,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{
				{
					ReferKey: *referKey,
					VotesInfo: payload.VotesWithLockTime{
						Candidate: publicKey1,
						Votes:     10000,
						LockTime:  100000,
					},
				},
				{
					ReferKey: *referKey,
					VotesInfo: payload.VotesWithLockTime{
						Candidate: publicKey1,
						Votes:     10000,
						LockTime:  100000,
					},
				},
			},
		},
	}

	// 2. Check output payload v0
	err := outputs1[0].Validate()
	s.NoError(err)
	err = outputs1[1].Validate()
	s.EqualError(err, "duplicate vote type")
	err = outputs1[2].Validate()
	s.EqualError(err, "invalid vote type")
	err = outputs1[3].Validate()
	s.EqualError(err, "duplicate candidate")
	err = outputs1[4].Validate()
	s.EqualError(err, "invalid candidate votes")
	err = outputs1[5].Validate()
	s.EqualError(err, "duplicate refer key")

}

func (s *txValidatorTestSuite) TestCheckVoteProducerOutput() {
	// 1. Generate a vote output v0
	publicKeyStr1 := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd47e944507292ea08dd"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	outputs1 := []*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 3,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: 2,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
	}

	// 2. Check output payload v0
	err := outputs1[0].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs1[1].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid public key count")

	err = outputs1[2].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate candidate")

	err = outputs1[3].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid vote version")

	err = outputs1[4].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate vote type")

	err = outputs1[5].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs1[6].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	// 3. Generate a vote output v1
	outputs := []*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 3,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: 2,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
	}

	// 2. Check output payload v1
	err = outputs[0].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs[1].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid public key count")

	err = outputs[2].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate candidate")

	err = outputs[3].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid vote version")

	err = outputs[4].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate vote type")

	err = outputs[5].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs[6].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid candidate votes")
}

func (s *txValidatorTestSuite) TestCheckVoteOutputs() {

	references := make(map[*common2.Input]common2.Output)
	outputs := []*common2.Output{{Type: common2.OTNone}}
	s.NoError(s.Chain.CheckVoteOutputs(0, outputs, references, nil, nil, nil))

	publicKey1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	publicKey2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	publicKey3 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	privateKeyStr3 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"

	registerCRTxn1 := s.getRegisterCRTx(publicKey1, privateKeyStr1,
		"nickName1", payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn2 := s.getRegisterCRTx(publicKey2, privateKeyStr2,
		"nickName2", payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn3 := s.getRegisterCRTx(publicKey3, privateKeyStr3,
		"nickName3", payload.CRInfoVersion, &common.Uint168{})

	s.CurrentHeight = 1
	ckpManager := checkpoint.NewManager(&config.DefaultParams)
	ckpManager.SetDataPath(filepath.Join(config.DefaultParams.DataDir, "checkpoints"))
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams(), ckpManager))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)
	code1 := getCodeByPubKeyStr(publicKey1)
	code2 := getCodeByPubKeyStr(publicKey2)
	code3 := getCodeByPubKeyStr(publicKey3)

	candidate1, _ := common.HexStringToBytes(publicKey1)
	candidate2, _ := common.HexStringToBytes(publicKey2)
	candidateCID1 := getCID(code1)
	candidateCID2 := getCID(code2)
	candidateCID3 := getCID(code3)

	producersMap := make(map[string]struct{})
	producersMap[publicKey1] = struct{}{}
	producersMap2 := make(map[string]uint32)
	producersMap2[publicKey1] = 0
	crsMap := make(map[common.Uint168]struct{})

	crsMap[*candidateCID1] = struct{}{}
	crsMap[*candidateCID3] = struct{}{}

	hashStr := "21c5656c65028fe21f2222e8f0cd46a1ec734cbdb6"
	hashByte, _ := common.HexStringToBytes(hashStr)
	hash, _ := common.Uint168FromBytes(hashByte)

	// Check vote output of v0 with delegate type and wrong output program hash
	outputs1 := []*common2.Output{{Type: common2.OTNone}}
	outputs1 = append(outputs1, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs1, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs2 := []*common2.Output{{Type: common2.OTNone}}
	outputs2 = append(outputs2, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs2, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs20 := []*common2.Output{{Type: common2.OTNone}}
	outputs20 = append(outputs20, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs20, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs21 := []*common2.Output{{Type: common2.OTNone}}
	outputs21 = append(outputs21, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCImpeachment,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs21, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs22 := []*common2.Output{{Type: common2.OTNone}}
	outputs22 = append(outputs22, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs22, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v1 with wrong output program hash
	outputs3 := []*common2.Output{{Type: common2.OTNone}}
	outputs3 = append(outputs3, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs3, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	references[&common2.Input{}] = common2.Output{
		ProgramHash: *hash,
	}

	// Check vote output of v0 with delegate type and invalid candidate
	outputs4 := []*common2.Output{{Type: common2.OTNone}}
	outputs4 = append(outputs4, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate2, 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs4, references, producersMap, nil, crsMap),
		"invalid vote output payload producer candidate: "+publicKey2)

	// Check vote output of v0 with delegate type and invalid candidate
	outputs23 := []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate2, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap),
		"invalid vote output payload producer candidate: "+publicKey2)

	outputs23 = []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap),
		fmt.Sprintf("payload VoteDposV2Version not support vote DposV2"))

	outputs23 = []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: outputpayload.VoteDposV2Version,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap))

	// Check vote output v0 with correct output program hash
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs1, references, producersMap, nil, crsMap))
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs2, references, producersMap, nil, crsMap))
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs3, references, producersMap, nil, crsMap))

	// Check vote output of v0 with crc type and invalid candidate
	outputs5 := []*common2.Output{{Type: common2.OTNone}}
	outputs5 = append(outputs5, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs5, references, producersMap, nil, crsMap),
		"payload VoteProducerVersion not support vote CR")

	// Check vote output of v1 with crc type and invalid candidate
	outputs6 := []*common2.Output{{Type: common2.OTNone}}
	outputs6 = append(outputs6, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs6, references, producersMap, nil, crsMap),
		"invalid vote output payload CR candidate: "+candidateCID2.String())

	// Check vote output of v0 with invalid candidate
	outputs7 := []*common2.Output{{Type: common2.OTNone}}
	outputs7 = append(outputs7, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs7, references, producersMap, nil, crsMap),
		"payload VoteProducerVersion not support vote CR")

	// Check vote output of v1 with delegate type and wrong votes
	outputs8 := []*common2.Output{{Type: common2.OTNone}}
	outputs8 = append(outputs8, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 20},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs8, references, producersMap, nil, crsMap),
		"votes larger than output amount")

	// Check vote output of v1 with crc type and wrong votes
	outputs9 := []*common2.Output{{Type: common2.OTNone}}
	outputs9 = append(outputs9, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID1.Bytes(), 10},
						{candidateCID3.Bytes(), 10},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs9, references, producersMap, nil, crsMap),
		"total votes larger than output amount")

	// Check vote output of v1 with wrong votes
	outputs10 := []*common2.Output{{Type: common2.OTNone}}
	outputs10 = append(outputs10, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 20},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 20},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs10, references, producersMap, nil, crsMap),
		"votes larger than output amount")

	// Check vote output v1 with correct votes
	outputs11 := []*common2.Output{{Type: common2.OTNone}}
	outputs11 = append(outputs11, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 10},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 10},
					},
				},
			},
		},
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs11, references, producersMap, nil, crsMap))

	// Check vote output of v1 with wrong votes
	outputs12 := []*common2.Output{{Type: common2.OTNone}}
	outputs12 = append(outputs12, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 1},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 1},
					},
				},
			},
		},
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs12, references, producersMap, nil, crsMap))

	// Check vote output v1 with correct votes
	proposalHashStr1 := "5df40cc0a4c6791acb5ebe89a96dd4f3fe21c94275589a65357406216a27ae36"
	proposalHash1, _ := common.Uint256FromHexString(proposalHashStr1)
	outputs13 := []*common2.Output{{Type: common2.OTNone}}
	outputs13 = append(outputs13, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{proposalHash1.Bytes(), 10},
					},
				},
			},
		},
	})
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash1] =
		&crstate.ProposalState{Status: 1}
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs13, references, producersMap, nil, crsMap))

	// Check vote output of v1 with wrong votes
	proposalHashStr2 := "9c5ab8998718e0c1c405a719542879dc7553fca05b4e89132ec8d0e88551fcc0"
	proposalHash2, _ := common.Uint256FromHexString(proposalHashStr2)
	outputs14 := []*common2.Output{{Type: common2.OTNone}}
	outputs14 = append(outputs14, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{proposalHash2.Bytes(), 10},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRConfiguration.CRVotingStartHeight,
		outputs14, references, producersMap, nil, crsMap),
		"invalid CRCProposal: c0fc5185e8d0c82e13894e5ba0fc5375dc79285419a705c4c1e0188799b85a9c")
}
