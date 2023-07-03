// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package config

import (
	"math"
	"math/big"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/types"
)

const (
	// ConfigFile for node config
	ConfigFile = "./config.json"
	// DataDir storing the chain data.
	DataDir = "elastos"
)

type Config struct {
	*Configuration `json:"Configuration"`
}

var (
	// DefaultParams defines the default network parameters.
	DefaultParams = *GetDefaultParams()
	Parameters    *Configuration

	// OriginIssuanceAmount is the origin issuance ELA amount.
	OriginIssuanceAmount = 3300 * 10000 * 100000000

	// AfterBurnIssuanceAmount is the new issurance ELA amount after cr proposal #1631.
	AfterBurnIssuanceAmount = 2000 * 10000 * 100000000

	// inflationPerYear is the inflation amount per year.
	inflationPerYear = OriginIssuanceAmount * 4 / 100

	// newInflationPerYear is the new inflation amount per year.
	newInflationPerYear = AfterBurnIssuanceAmount * 4 / 100

	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// powLimit is the highest proof of work value a block can have for the network.
	//  It is the value 2^255 - 1.
	powLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// mainNetFoundation "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	mainNetFoundationProgramHash = &common.Uint168{
		0x12, 0x9e, 0x9c, 0xf1, 0xc5, 0xf3, 0x36,
		0xfc, 0xf3, 0xa6, 0xc9, 0x54, 0x44, 0x4e,
		0xd4, 0x82, 0xc5, 0xd9, 0x16, 0xe5, 0x06,
	}

	// testNetFoundation "8ZNizBf4KhhPjeJRGpox6rPcHE5Np6tFx3"
	testNetFoundationProgramHash = &common.Uint168{
		0x12, 0xc8, 0xa2, 0xe0, 0x67, 0x72, 0x27,
		0x14, 0x4d, 0xf8, 0x22, 0xb7, 0xd9, 0x24,
		0x6c, 0x58, 0xdf, 0x68, 0xeb, 0x11, 0xce,
	}

	// mainNetCRCAddress "8ZZLWQUDSbjWUn8sEdxEFJsZiRFpzg53rJ"
	mainNetCRCProgramHash = &common.Uint168{
		0x12, 0xca, 0xa4, 0xb0, 0x0b, 0x86, 0x8b,
		0xe7, 0x92, 0xe2, 0x40, 0x1e, 0x97, 0x4e,
		0xcd, 0x5f, 0xcb, 0x1f, 0xd9, 0xab, 0x37,
	}

	// testNetCRCAddress "8JJCdEjMRm6x2rVsSMesL5gmoq7ts4wHMo"
	testNetCRCProgramHash = &common.Uint168{
		0x12, 0x23, 0x3d, 0xfb, 0x54, 0x88, 0xbb,
		0xa2, 0xe9, 0xfa, 0x9a, 0x11, 0xbb, 0x07,
		0xac, 0x10, 0xcd, 0x77, 0x29, 0x41, 0x22,
	}

	// DestroyELAAddress "ELANULLXXXXXXXXXXXXXXXXXXXXXYvs3rr"
	DestroyELAProgramHash = &common.Uint168{
		0x21, 0x20, 0xfe, 0xe5, 0xd7, 0xeb, 0x3e,
		0x5c, 0x7d, 0x31, 0x97, 0xfe, 0xcf, 0x6c,
		0x0d, 0xe3, 0x0f, 0x88, 0x9a, 0xce, 0xf7,
	}

	// CRAssetsAddress "CRASSETSXXXXXXXXXXXXXXXXXXXX2qDX5J"
	CRAssetsProgramHash = &common.Uint168{
		0x1c, 0x5f, 0x6c, 0x3e, 0x9f, 0x0c, 0x9a,
		0x71, 0x10, 0xb8, 0xeb, 0x6e, 0x37, 0x02,
		0x63, 0x0f, 0x71, 0x27, 0x4d, 0xf1, 0xc0,
	}

	//CRCExpensesAddress = "CREXPENSESXXXXXXXXXXXXXXXXXX4UdT6b"
	CRCExpensesProgramHash = &common.Uint168{
		0x1c, 0x60, 0x32, 0x09, 0xff, 0x5d, 0x86,
		0x54, 0x2d, 0x26, 0x1a, 0x8d, 0x22, 0x7f,
		0xe0, 0xf5, 0xee, 0x2c, 0x4e, 0xd2, 0xd3,
	}

	// StakePoolAddress "STAKEPooLXXXXXXXXXXXXXXXXXXXpP1PQ2"
	StakePoolProgramHash = &common.Uint168{
		0x3f, 0x40, 0x5c, 0x9e, 0x71, 0x3f, 0x20,
		0x13, 0xa3, 0x16, 0x39, 0x49, 0xbd, 0xd6,
		0xf0, 0xbc, 0xad, 0xd6, 0xc8, 0xf5, 0xab,
	}

	// StakeRewardAddress "STAKEREWARDXXXXXXXXXXXXXXXXXFD5SHU"
	StakeRewardProgramHash = &common.Uint168{
		0x3f, 0x40, 0x5c, 0x9e, 0x88, 0x6f, 0x83,
		0xef, 0xe2, 0xeb, 0x7f, 0x86, 0x5c, 0xd4,
		0xf1, 0x44, 0x5f, 0xb2, 0x11, 0x9c, 0xe6,
	}
)

func SetParameters(configuration *Configuration) {
	Parameters = configuration
}

func GetDefaultParams() *Configuration {
	// DefaultParams defines the default network parameters.
	return &Configuration{
		Magic:      2017001,
		PrintLevel: 1,
		NodePort:   20338,
		DNSSeeds: []string{
			"52.74.28.202:20338",
			"52.62.113.83:20338",
			"35.156.51.127:20338",
			"35.177.89.244:20338",
		},
		DestroyELAProgramHash: DestroyELAProgramHash,
		FoundationProgramHash: mainNetFoundationProgramHash,

		CRConfiguration: CRConfiguration{
			CRCProgramHash:                     mainNetCRCProgramHash,
			CRAssetsProgramHash:                CRAssetsProgramHash,
			CRExpensesProgramHash:              CRCExpensesProgramHash,
			MemberCount:                        12,
			VotingPeriod:                       30 * 720,
			DutyPeriod:                         365 * 720,
			DepositLockupBlocks:                2160,
			CRVotingStartHeight:                537670,
			CRCommitteeStartHeight:             658930,
			CRClaimDPOSNodeStartHeight:         751400,
			CRClaimDPOSNodePeriod:              720 * 14,
			CRAgreementCount:                   8,
			RegisterCRByDIDHeight:              598000,
			CRCProposalV1Height:                751400,
			CRCAppropriatePercentage:           10,
			CRCProposalWithdrawPayloadV1Height: 751400,
			CRCProposalDraftDataStartHeight:    1056600,
			SecretaryGeneral:                   "02712da531804d1c38d159a901313239d2100dfb5b693d71a2f76b15dec3f8fc32",
			MaxProposalTrackingCount:           128,
			CRAssetsRectifyTransactionHeight:   751400,
			ProposalCRVotingPeriod:             7 * 720,
			ProposalPublicVotingPeriod:         7 * 720,
			VoterRejectPercentage:              10,
			MaxCommitteeProposalCount:          128,
			MaxCRAssetsAddressUTXOCount:        800,
			MinCRAssetsAddressUTXOCount:        720,
			RectifyTxFee:                       10000,
			RealWithdrawSingleFee:              10000,
			NewP2PProtocolVersionHeight:        751400,
			ChangeCommitteeNewCRHeight:         932530,
			CheckVoteCRCountHeight:             658930,
			CRClaimPeriod:                      720 * 14,
		},

		DPoSConfiguration: DPoSConfiguration{
			Magic:                             2019000,
			DPoSPort:                          20339,
			CRDPoSNodeHotFixHeight:            0,
			PreConnectOffset:                  360,
			NoCRCDPOSNodeHeight:               932530,
			RevertToPOWStartHeight:            932530,
			RandomCandidatePeriod:             36 * 10,
			MaxInactiveRoundsOfRandomNode:     36 * 8,
			RevertToPOWNoBlockTime:            12 * 3600,
			StopConfirmBlockTime:              11 * 3600,
			DPoSV2IllegalPenalty:              20000000000,
			DPOSNodeCrossChainHeight:          math.MaxUint32,
			DPoSV2DepositCoinMinLockTime:      7200,
			DPoSV2MinVotesLockTime:            7200,
			DPoSV2MaxVotesLockTime:            720000,
			SignTolerance:                     5,
			MaxInactiveRounds:                 720 * 2,
			InactivePenalty:                   0, //there will be no penalty in this version
			IllegalPenalty:                    0,
			EmergencyInactivePenalty:          0, //there will be no penalty in this version
			NormalArbitratorsCount:            24,
			CandidatesCount:                   72,
			DPoSV2RewardAccumulateProgramHash: StakeRewardProgramHash,
			NFTStartHeight:                    1405000,
			NFTV2StartHeight:                  math.MaxUint32, // todo complete me
			DexStartHeight:                    math.MaxUint32, // todo complete me
			OriginArbiters: []string{
				"0248df6705a909432be041e0baa25b8f648741018f70d1911f2ed28778db4b8fe4",
				"02771faf0f4d4235744b30972d5f2c470993920846c761e4d08889ecfdc061cddf",
				"0342196610e57d75ba3afa26e030092020aec56822104e465cba1d8f69f8d83c8e",
				"02fa3e0d14e0e93ca41c3c0f008679e417cf2adb6375dd4bbbee9ed8e8db606a56",
				"03ab3ecd1148b018d480224520917c6c3663a3631f198e3b25cf4c9c76786b7850",
			},
			CRCArbiters: []string{
				"02089d7e878171240ce0e3633d3ddc8b1128bc221f6b5f0d1551caa717c7493062",
				"0268214956b8421c0621d62cf2f0b20a02c2dc8c2cc89528aff9bd43b45ed34b9f",
				"03cce325c55057d2c8e3fb03fb5871794e73b85821e8d0f96a7e4510b4a922fad5",
				"02661637ae97c3af0580e1954ee80a7323973b256ca862cfcf01b4a18432670db4",
				"027d816821705e425415eb64a9704f25b4cd7eaca79616b0881fc92ac44ff8a46b",
				"02d4a8f5016ae22b1acdf8a2d72f6eb712932213804efd2ce30ca8d0b9b4295ac5",
				"029a4d8e4c99a1199f67a25d79724e14f8e6992a0c8b8acf102682bd8f500ce0c1",
				"02871b650700137defc5d34a11e56a4187f43e74bb078e147dd4048b8f3c81209f",
				"02fc66cba365f9957bcb2030e89a57fb3019c57ea057978756c1d46d40dfdd4df0",
				"03e3fe6124a4ea269224f5f43552250d627b4133cfd49d1f9e0283d0cd2fd209bc",
				"02b95b000f087a97e988c24331bf6769b4a75e4b7d5d2a38105092a3aa841be33b",
				"02a0aa9eac0e168f3474c2a0d04e50130833905740a5270e8a44d6c6e85cf6d98c",
			},
		},
		MinTransactionFee:               100,
		MinCrossChainTxFee:              10000,
		CheckAddressHeight:              88812,
		VoteStartHeight:                 290000,
		CRCOnlyDPOSHeight:               343400,
		PublicDPOSHeight:                402680,
		EnableActivateIllegalHeight:     439000,
		CheckRewardHeight:               436812,
		VoteStatisticsHeight:            512881,
		EnableUtxoDB:                    true,
		EnableCORS:                      false,
		WalletPath:                      "keystore.dat",
		RPCServiceLevel:                 ConfigurationPermitted.String(),
		OriginDestroyAmount:             common.Fixed64(45341569050100),
		MemoryFirst:                     false,
		MaxNodePerHost:                  72,
		TxCacheVolume:                   100000,
		CustomIDProposalStartHeight:     932530,
		MaxReservedCustomIDLength:       255,
		HalvingRewardHeight:             1051200, // 4 * 365 * 720
		HalvingRewardInterval:           1051200, // 4 * 365 * 720
		NewELAIssuanceHeight:            919800,  // 3.5 * 365 * 720
		SmallCrossTransferThreshold:     100000000,
		ReturnDepositCoinFee:            100,
		NewCrossChainStartHeight:        1032840,
		ReturnCrossChainCoinStartHeight: 1032840,
		ProhibitTransferToDIDHeight:     1032840,
		DIDSideChainAddress:             "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ",
		DPoSV2EffectiveVotes:            80000 * 100000000,
		DPoSV2StartHeight:               1405000,
		StakePoolProgramHash:            StakePoolProgramHash,
		SchnorrStartHeight:              math.MaxUint32,
		NormalSchnorrStartHeight:        1405000,
		ProducerSchnorrStartHeight:      math.MaxUint32,
		CRSchnorrStartHeight:            math.MaxUint32,
		VotesSchnorrStartHeight:         math.MaxUint32,
		CrossChainMonitorStartHeight:    math.MaxUint32,
		CrossChainMonitorInterval:       100,
		SupportMultiCodeHeight:          math.MaxUint32, // todo complete me
		MultiExchangeVotesStartHeight:   math.MaxUint32, // todo complete me
		HttpInfoPort:                    20333,
		HttpRestPort:                    20334,
		HttpWsPort:                      20335,
		HttpJsonPort:                    20336,
		PowConfiguration: PowConfiguration{
			PowLimit:           powLimit,
			PowLimitBits:       0x1f0008ff,
			TargetTimespan:     24 * time.Hour,  // 24 hours
			TargetTimePerBlock: 2 * time.Minute, // 2 minute
			AdjustmentFactor:   4,               // 25% less, 400% more
			RewardPerBlock:     rewardPerBlock(2 * time.Minute),
			CoinbaseMaturity:   100,
		},
		CheckPointConfiguration: CheckPointConfiguration{
			EnableHistory:      true,
			HistoryStartHeight: uint32(0),
			NeedSave:           true,
		},
		MemoryPoolTxMaximumStayHeight: 10,
	}
}

// TestNet returns the network parameters for the test network.
func (p *Configuration) TestNet() *Configuration {
	p.Magic = 2018101
	p.PrintLevel = 1
	p.NodePort = 21338

	p.DNSSeeds = []string{
		"13.234.24.155:21338",
		"15.206.198.252:21338",
		"13.234.249.168:21338",
	}
	p.DestroyELAProgramHash = DestroyELAProgramHash
	p.FoundationProgramHash = testNetFoundationProgramHash
	p.CRConfiguration.CRCProgramHash = testNetCRCProgramHash
	p.CRConfiguration.CRAssetsProgramHash = CRAssetsProgramHash
	p.CRConfiguration.CRExpensesProgramHash = CRCExpensesProgramHash
	p.StakePoolProgramHash = StakePoolProgramHash
	p.DPoSConfiguration.DPoSV2RewardAccumulateProgramHash = StakeRewardProgramHash

	p.DPoSConfiguration.Magic = 2019100
	p.DPoSConfiguration.DPoSPort = 21339
	p.DPoSConfiguration.OriginArbiters = []string{
		"03e333657c788a20577c0288559bd489ee65514748d18cb1dc7560ae4ce3d45613",
		"02dd22722c3b3a284929e4859b07e6a706595066ddd2a0b38e5837403718fb047c",
		"03e4473b918b499e4112d281d805fc8d8ae7ac0a71ff938cba78006bf12dd90a85",
		"03dd66833d28bac530ca80af0efbfc2ec43b4b87504a41ab4946702254e7f48961",
		"02c8a87c076112a1b344633184673cfb0bb6bce1aca28c78986a7b1047d257a448",
	}
	p.DPoSConfiguration.CRCArbiters = []string{
		"03e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b6",
		"038a1829b4b2bee784a99bebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca771",
		"02435df9a4728e6250283cfa8215f16b48948d71936c4600b3a5b1c6fde70503ae",
		"027d44ee7e7a6c6ff13a130d15b18c75a3b47494c3e54fcffe5f4b10e225351e09",
		"02ad972fbfce4aaa797425138e4f3b22bcfa765ffad88b8a5af0ab515161c0a365",
		"0373eeae2bac0f5f14373ca603fe2c9caa9c7a79c7793246cec415d005e2fe53c0",
		"03503011cc4e44b94f73ed2c76c73182a75b4863f23d1e7083025eead945a8e764",
		"0270b6880e7fab8d02bea7d22639d7b5e07279dd6477baa713dacf99bb1d65de69",
		"030eed9f9c1d70307beba52ddb72a24a02582c0ee626ec93ee1dcef2eb308852dd",
		"026bba43feb19ce5859ffcf0ce9dd8b9d625130b686221da8b445fa9b8f978d7b9",
		"02bf9e37b3db0cbe86acf76a76578c6b17b4146df101ec934a00045f7d201f06dd",
		"03111f1247c66755d369a8c8b3a736dfd5cf464ca6735b659533cbe1268cd102a9",
	}
	p.CRConfiguration.SecretaryGeneral = "0349cb77a69aa35be0bcb044ffd41a616b8367136d3b339d515b1023cc0f302f87"
	p.CRConfiguration.MaxProposalTrackingCount = 128
	p.CheckAddressHeight = 0
	p.VoteStartHeight = 200000
	p.CRCOnlyDPOSHeight = 246700
	p.PublicDPOSHeight = 300000
	p.CRConfiguration.CRVotingStartHeight = 436900
	p.CRConfiguration.CRCommitteeStartHeight = 546500
	p.CRConfiguration.CRClaimDPOSNodeStartHeight = 646700
	p.CRConfiguration.CRClaimDPOSNodePeriod = 720 * 7
	p.CRConfiguration.CRCProposalV1Height = 646700
	p.CRConfiguration.NewP2PProtocolVersionHeight = 646700
	p.CRConfiguration.CRAssetsRectifyTransactionHeight = 646700
	p.CRConfiguration.CRCProposalWithdrawPayloadV1Height = 646700
	p.EnableActivateIllegalHeight = 546500
	p.CheckRewardHeight = 100
	p.VoteStatisticsHeight = 0
	p.CRConfiguration.RegisterCRByDIDHeight = 483500
	p.EnableUtxoDB = true
	p.EnableCORS = false
	p.CRConfiguration.VoterRejectPercentage = 10
	p.CRConfiguration.CRCAppropriatePercentage = 10
	p.CRConfiguration.MaxCommitteeProposalCount = 128
	p.MaxNodePerHost = 10
	p.CRConfiguration.CheckVoteCRCountHeight = 546500
	p.CRConfiguration.MaxCRAssetsAddressUTXOCount = 800
	p.CRConfiguration.ChangeCommitteeNewCRHeight = 815060
	p.CustomIDProposalStartHeight = 815060
	p.DPoSConfiguration.InactivePenalty = 0
	p.DPoSConfiguration.IllegalPenalty = 0
	p.DPoSConfiguration.DPoSV2IllegalPenalty = 200 * 100000000
	p.DPoSConfiguration.NoCRCDPOSNodeHeight = 815060
	p.DPoSConfiguration.RandomCandidatePeriod = 36 * 10
	p.DPoSConfiguration.MaxInactiveRoundsOfRandomNode = 36 * 8
	p.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32 // todo complete me
	p.MaxReservedCustomIDLength = 255
	p.DPoSConfiguration.RevertToPOWNoBlockTime = 12 * 3600
	p.DPoSConfiguration.StopConfirmBlockTime = 11 * 3600
	p.DPoSConfiguration.RevertToPOWStartHeight = 815060
	p.HalvingRewardHeight = 877880    //767000 + 154 * 720
	p.HalvingRewardInterval = 1051200 //4 * 365 * 720
	p.NewELAIssuanceHeight = 774920   //767000 + 720 * 11
	p.SmallCrossTransferThreshold = 100000000
	p.ReturnDepositCoinFee = 100
	p.NewCrossChainStartHeight = 807000
	p.ReturnCrossChainCoinStartHeight = 807000
	p.CRConfiguration.CRCProposalDraftDataStartHeight = 807000
	p.ProhibitTransferToDIDHeight = 807000
	p.DIDSideChainAddress = "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
	p.DPoSV2StartHeight = 965800 + 720*3
	p.SupportMultiCodeHeight = 1171000
	p.DPoSV2EffectiveVotes = 3000 * 100000000
	p.DPoSConfiguration.DPoSV2DepositCoinMinLockTime = 7200 * 3
	p.DPoSConfiguration.DPoSV2MinVotesLockTime = 7200
	p.DPoSConfiguration.DPoSV2MaxVotesLockTime = 720000
	p.CRConfiguration.RealWithdrawSingleFee = 50000
	p.SchnorrStartHeight = 965800 + 720*10
	p.NormalSchnorrStartHeight = 965800 + 720*10
	p.DPoSConfiguration.CRDPoSNodeHotFixHeight = 0
	p.CrossChainMonitorStartHeight = 965800 + 720*3
	p.CrossChainMonitorInterval = 100
	p.CRConfiguration.CRClaimPeriod = 10080
	p.DPoSConfiguration.NFTStartHeight = 1098000
	p.DPoSConfiguration.NFTV2StartHeight = 1171000
	p.DPoSConfiguration.DexStartHeight = 1171000

	p.HttpInfoPort = 21333
	p.HttpRestPort = 21334
	p.HttpWsPort = 21335
	p.HttpJsonPort = 21336
	p.ProducerSchnorrStartHeight = math.MaxUint32 // todo complete me
	p.CRSchnorrStartHeight = math.MaxUint32       // todo complete me
	p.VotesSchnorrStartHeight = math.MaxUint32    // todo complete me
	p.MultiExchangeVotesStartHeight = 1171000

	p.MemoryPoolTxMaximumStayHeight = 10

	return p
}

// RegNet returns the network parameters for the test network.
func (p *Configuration) RegNet() *Configuration {
	p.Magic = 2018201
	p.PrintLevel = 1
	p.NodePort = 22338

	p.DNSSeeds = []string{
		"15.206.236.124:22338",
		"3.111.241.201:22338",
		"65.0.164.47:22338",
	}

	p.FoundationProgramHash = testNetFoundationProgramHash
	p.CRConfiguration.CRCProgramHash = testNetCRCProgramHash
	p.CRConfiguration.CRAssetsProgramHash = CRAssetsProgramHash
	p.CRConfiguration.CRExpensesProgramHash = CRCExpensesProgramHash
	p.DestroyELAProgramHash = DestroyELAProgramHash
	p.StakePoolProgramHash = StakePoolProgramHash
	p.DPoSConfiguration.DPoSV2RewardAccumulateProgramHash = StakeRewardProgramHash

	p.DPoSConfiguration.Magic = 2019200
	p.DPoSConfiguration.DPoSPort = 22339
	p.DPoSConfiguration.OriginArbiters = []string{
		"03e333657c788a20577c0288559bd489ee65514748d18cb1dc7560ae4ce3d45613",
		"02dd22722c3b3a284929e4859b07e6a706595066ddd2a0b38e5837403718fb047c",
		"03e4473b918b499e4112d281d805fc8d8ae7ac0a71ff938cba78006bf12dd90a85",
		"03dd66833d28bac530ca80af0efbfc2ec43b4b87504a41ab4946702254e7f48961",
		"02c8a87c076112a1b344633184673cfb0bb6bce1aca28c78986a7b1047d257a448",
	}
	p.DPoSConfiguration.CRCArbiters = []string{
		"0306e3deefee78e0e25f88e98f1f3290ccea98f08dd3a890616755f1a066c4b9b8",
		"02b56a669d713db863c60171001a2eb155679cad186e9542486b93fa31ace78303",
		"0250c5019a00f8bb4fd59bb6d613c70a39bb3026b87cfa247fd26f59fd04987855",
		"02e00112e3e9defe0f38f33aaa55551c8fcad6aea79ab2b0f1ec41517fdd05950a",
		"020aa2d111866b59c70c5acc60110ef81208dcdc6f17f570e90d5c65b83349134f",
		"03cd41a8ed6104c1170332b02810237713369d0934282ca9885948960ae483a06d",
		"02939f638f3923e6d990a70a2126590d5b31a825a0f506958b99e0a42b731670ca",
		"032ade27506951c25127b0d2cb61d164e0bad8aec3f9c2e6785725a6ab6f4ad493",
		"03f716b21d7ae9c62789a5d48aefb16ba1e797b04a2ec1424cd6d3e2e0b43db8cb",
		"03488b0aace5fe5ee5a1564555819074b96cee1db5e7be1d74625240ef82ddd295",
		"03c559769d5f7bb64c28f11760cb36a2933596ca8a966bc36a09d50c24c48cc3e8",
		"03b5d90257ad24caf22fa8a11ce270ea57f3c2597e52322b453d4919ebec4e6300",
	}
	p.CRConfiguration.SecretaryGeneral = "0349cb77a69aa35be0bcb044ffd41a616b8367136d3b339d515b1023cc0f302f87"
	p.CRConfiguration.MaxProposalTrackingCount = 128
	p.CheckAddressHeight = 0
	p.VoteStartHeight = 170000
	p.CRCOnlyDPOSHeight = 211000
	p.PublicDPOSHeight = 231500
	p.CRConfiguration.CRVotingStartHeight = 292000
	p.CRConfiguration.CRCommitteeStartHeight = 442000
	p.CRConfiguration.CRClaimDPOSNodeStartHeight = 532650
	p.CRConfiguration.CRClaimDPOSNodePeriod = 720
	p.CRConfiguration.CRCProposalV1Height = 530000
	p.CRConfiguration.NewP2PProtocolVersionHeight = 531030
	p.CRConfiguration.CRAssetsRectifyTransactionHeight = 532650
	p.CRConfiguration.CRCProposalWithdrawPayloadV1Height = 532650
	p.EnableActivateIllegalHeight = 256000
	p.CheckRewardHeight = 280000
	p.VoteStatisticsHeight = 0
	p.CRConfiguration.RegisterCRByDIDHeight = 393000

	p.EnableUtxoDB = true
	p.EnableCORS = false
	p.CRConfiguration.VoterRejectPercentage = 10
	p.CRConfiguration.CRCAppropriatePercentage = 10
	p.CRConfiguration.MaxCommitteeProposalCount = 128
	p.MaxNodePerHost = 10
	p.CRConfiguration.CheckVoteCRCountHeight = 435000
	p.CRConfiguration.MaxCRAssetsAddressUTXOCount = 1440
	p.CRConfiguration.ChangeCommitteeNewCRHeight = 706240
	p.CustomIDProposalStartHeight = 706240
	p.DPoSConfiguration.IllegalPenalty = 0
	p.DPoSConfiguration.DPoSV2IllegalPenalty = 20000000000
	p.DPoSConfiguration.InactivePenalty = 0
	p.DPoSConfiguration.NoCRCDPOSNodeHeight = 706240
	p.DPoSConfiguration.RandomCandidatePeriod = 36 * 10
	p.DPoSConfiguration.MaxInactiveRoundsOfRandomNode = 36 * 8
	p.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32 // todo complete me
	p.MaxReservedCustomIDLength = 255
	p.DPoSConfiguration.RevertToPOWNoBlockTime = 12 * 3600
	p.DPoSConfiguration.StopConfirmBlockTime = 11 * 3600
	p.DPoSConfiguration.RevertToPOWStartHeight = 706240
	p.HalvingRewardHeight = 801240    //690360 + 154 * 720
	p.HalvingRewardInterval = 1051200 //4 * 365 * 720
	p.NewELAIssuanceHeight = 691740   //690300 + 720 * 2
	p.SmallCrossTransferThreshold = 100000000
	p.ReturnDepositCoinFee = 100
	p.NewCrossChainStartHeight = 730000
	p.ReturnCrossChainCoinStartHeight = 730000
	p.CRConfiguration.CRCProposalDraftDataStartHeight = 730000
	p.ProhibitTransferToDIDHeight = 730000
	p.DIDSideChainAddress = "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
	p.DPoSV2StartHeight = 875544 + 720*2
	p.SupportMultiCodeHeight = 2000
	p.DPoSV2EffectiveVotes = 300000000000
	p.DPoSConfiguration.DPoSV2DepositCoinMinLockTime = 7200 * 3
	p.DPoSConfiguration.DPoSV2MinVotesLockTime = 7200
	p.DPoSConfiguration.DPoSV2MaxVotesLockTime = 720000
	p.CRConfiguration.RealWithdrawSingleFee = 10000
	p.SchnorrStartHeight = 875544 + 720*5
	p.NormalSchnorrStartHeight = 875544 + 720*5
	p.DPoSConfiguration.CRDPoSNodeHotFixHeight = 0
	p.CrossChainMonitorStartHeight = 875544 + 720*2
	p.CrossChainMonitorInterval = 100
	p.CRConfiguration.CRClaimPeriod = 10080
	p.DPoSConfiguration.NFTStartHeight = 968000
	p.DPoSConfiguration.NFTV2StartHeight = math.MaxUint32 // todo complete me
	p.HttpInfoPort = 22333
	p.HttpRestPort = 22334
	p.HttpWsPort = 22335
	p.HttpJsonPort = 22336
	p.ProducerSchnorrStartHeight = math.MaxUint32
	p.CRSchnorrStartHeight = math.MaxUint32
	p.VotesSchnorrStartHeight = math.MaxUint32
	p.MultiExchangeVotesStartHeight = math.MaxUint32    // todo complete me
	p.DPoSConfiguration.DexStartHeight = math.MaxUint32 // todo complete me

	p.MemoryPoolTxMaximumStayHeight = 10

	return p
}

// Configuration defines the configurable parameters to run a ELA node.
type Configuration struct {
	Conf          string `screw:"--conf" usage:"set the config file path"`
	ActiveNet     string `json:"ActiveNet"`
	Password      string `screw:"short;--password" usage:"password for keystore"`
	DataDir       string `screw:"short;--datadir" usage:"block data and logs storage path default: elastos"`
	HttpInfoPort  uint16 `screw:"--infoport" usage:"port for the http info server"`
	HttpInfoStart bool   `json:"HttpInfoStart"`
	HttpRestPort  int    `screw:"--restport" usage:"port for the http restful server"`
	HttpRestStart bool   `json:"HttpRestStart"`
	HttpWsPort    int    `screw:"--wsport" usage:"port for the http web socket server"`
	HttpWsStart   bool   `json:"HttpWsStart"`
	HttpJsonPort  int    `screw:"--rpcport" usage:"port for the http json rpc port server"`
	ProfilePort   uint32 `screw:"--profileport" usage:"port for the http profile port rpc server"`
	ProfileHost   string `screw:"--profilehost" usage:"port for the http profile rpc host server"`
	DisableDNS    bool   `screw:"--disabledns" usage:"disable DNS for node"`
	EnableRPC     bool   `screw:"--enablerpc" usage:"enable RPC for node"`
	MaxLogsSize   int64  `json:"MaxLogsSize"`
	MaxPerLogSize int64  `json:"MaxPerLogSize"`
	RestCertPath  string `json:"RestCertPath"`
	RestKeyPath   string `json:"RestKeyPath"`
	// GenesisBlock defines the first block of the chain.
	GenesisBlock *types.Block

	// MaxBlockContextSize is the maximum number of bytes allowed per block context. default value 8000000
	MaxBlockSize uint32 `json:"MaxBlockSize"`
	// MaxBlockHeaderSize is the maximum number of bytes allowed per block header. default value 1000000
	MaxBlockHeaderSize uint32 `json:"MaxBlockHeaderSize"`
	// MaxTxPerBlock is the maximux number of transactions allowed per block. default value 10000
	MaxTxPerBlock uint32 `json:"MaxTxPerBlock"`
	// Show Peers Ip
	ShowPeersIp bool `json:"ShowPeersIp"`
	// Disable transaction filter supports, include bloom filter tx type filter etc.
	DisableTxFilters bool
	// PrintLevel defines the level to print log.
	PrintLevel uint32 `screw:"--printlevel" usage:"level to print log"`
	// NodePort defines the default peer-to-peer port for the network.
	NodePort uint16 `screw:"--nodeport" usage:"default peer-to-peer node port for the network"`
	// Magic defines the magic number of the peer-to-peer network.
	Magic uint32 `screw:"--magic" usage:"magic number for node to initialize p2p connection"`
	// DNSSeeds defines a list of DNS seeds for the network to discover peers.
	DNSSeeds []string `screw:"--dns" usage:"dns seeds for node to initialize p2p connection"`
	// PermanentPeers defines peers seeds for node to initialize p2p connection.
	PermanentPeers []string `json:"PermanentPeers"`
	// The interface/port to listen for connections.
	ListenAddrs []string `json:"ListenAddrs"`
	// MinCrossChainTxFee defines the min fee of cross chain transaction
	MinCrossChainTxFee common.Fixed64 `json:"MinCrossChainTxFee"`
	// MinTransactionFee defines the minimum fee of a transaction.
	MinTransactionFee common.Fixed64 `json:"MinTransactionFee"`
	// DestroyELAAddress defines address which receiving destroyed ELA.
	DestroyELAAddress     string `json:"DestroyELAAddress"`
	DestroyELAProgramHash *common.Uint168
	// OriginDestroyAmount defines the destroyed ELA before voting start height.
	OriginDestroyAmount common.Fixed64 `json:"OriginDestroyAmount"`
	// Foundation defines the foundation address which receiving mining rewards.
	FoundationAddress     string `screw:"--foundation" usage:"specify the foundation address"`
	FoundationProgramHash *common.Uint168
	// Did side chain address
	DIDSideChainAddress string `screw:"--didsidechainaddress" usage:"specify the did sidechain address"`
	//Prohibit transfers to did height
	ProhibitTransferToDIDHeight uint32 `screw:"--prohibittransfertodidheight" usage:"defines the height to prohibit transfer to did"`
	// CheckAddressHeight defines the height begin to check output hash.
	CheckAddressHeight uint32 `screw:"--checkaddressheight" usage:"defines the height begin to check output hash"`
	// VoteStartHeight indicates the height of starting register producer and vote related.
	VoteStartHeight uint32 `screw:"--votestartheight" usage:"ndicates the height of starting register producer and vote related"`
	// CRCOnlyDPOSHeight (H1) indicates the height of DPOS consensus begins with only CRC producers participate in producing blocks.
	CRCOnlyDPOSHeight uint32 `screw:"--crconlydposheight" usage:"(H1) indicates the height of DPOS consensus begins with only CRC producers participate in producing block"`
	// PublicDPOSHeight (H2) indicates the height when public registered and elected producers participate in DPOS consensus.
	PublicDPOSHeight uint32 `screw:"--publicdposheight" usage:"(H2) indicates the height when public registered and elected producers participate in DPOS consensus"`
	// PublicDPOSHeight defines the start height to enable activate illegal producer though activate tx.
	EnableActivateIllegalHeight uint32 `screw:"--enableactivateillegalheight" usage:"defines the start height to enable activate illegal producer though activate tx"`
	// CheckRewardHeight defines the height to check reward in coin base with new check function.
	CheckRewardHeight uint32 `screw:"--checkrewardheight" usage:"defines the height begin to check reward"`
	// VoteStatisticsHeight defines the height to deal with block with vote statistics error.
	VoteStatisticsHeight uint32 `screw:"--votestatisticsheight" usage:"defines the height to fix vote statistics error"`
	// EnableUtxoDB indicate whether to enable utxo database.
	EnableUtxoDB bool `json:"EnableUtxoDB"`
	// Enable cors for http server.
	EnableCORS bool `json:"EnableCORS"`
	// WalletPath defines the wallet path used by DPoS arbiters and CR members.
	WalletPath string `screw:"-w;--walletpath" json:"WalletPath" usage:"defines the keystore file"`
	// RPCServiceLevel defines level of service provide to client.
	RPCServiceLevel string `json:"RPCServiceLevel"`
	// NodeProfileStrategy defines strategy about node profiling.
	MemoryFirst bool `json:"NodeProfileStrategy"`
	// TxCacheVolume defines the default volume of the transaction cache.
	TxCacheVolume uint32 `json:"TxCacheVolume"`
	// MaxNodePerHost defines max nodes that one host can establish.
	MaxNodePerHost uint32 `screw:"--maxnodeperhost" usage:"defines max nodes that one host can establish"`
	// CustomIDProposalStartHeight defines the height to allow custom ID related transaction.
	CustomIDProposalStartHeight uint32 `screw:"--CustomIDProposalStartHeight" usage:"defines the height to allow custom ID related transaction"`
	// MaxReservedCustomIDLength defines the max length of reserved custom id.
	MaxReservedCustomIDLength uint32 `screw:"--maxreservedcustomidlength" usage:"defines the max count of reserved custom iid list per tx"`
	// HalvingRewardHeight represents the height of halving reward
	HalvingRewardHeight uint32 `screw:"--halvingrewardheight" usage:"defines height of having reward"`
	// HalvingRewardInterval represents the interval of halving reward
	HalvingRewardInterval uint32 `screw:"--halvingrewardinterval" usage:"defines interval of having reward"`
	// NewELAIssuanceHeight represents the new issuance ELA amount after proposal #1631
	NewELAIssuanceHeight uint32 `screw:"--newelaissuanceheight" usage:"defines height of using the new ela issuance (2000w)"`
	// SMALLCrossTransferThreshold indicates the minimum amount consider as Small transfer
	SmallCrossTransferThreshold common.Fixed64 `screw:"--smallcrosstransferthreshold" usage:"defines the minimum amount of transfer consider as small cross transfer"`
	// ReturnDepositCoinFee indicates the fee the
	ReturnDepositCoinFee common.Fixed64 `screw:"--returndepositcoinfee" usage:"defines the fee of return cross chain deposit coin"`
	// NewCrossChainStartHeight defines the height of new cross chain transaction started.
	NewCrossChainStartHeight uint32 `screw:"--newcrosschainstartheight" usage:"defines the height to only support TransferCrossChainAsset v1"`
	// ReturnCrossChainCoinStartHeight indicates the start height of ReturnCroossChainDepositCoin transaction
	ReturnCrossChainCoinStartHeight uint32 `screw:"--returncrosschaincoinstartheight" usage:"defines the start height to support ReturnCrossChainDepositCoin transaction"`
	// DPoSV2StartHeight defines the start height of dpos 2.0.
	DPoSV2StartHeight uint32 `screw:"--dposv2startheight" usage:"defines the start height to support DPoSV2 transaction"`
	// multicode support height
	SupportMultiCodeHeight uint32 `screw:"--supportmulticodeheight" usage:"defines the support height of multicode transaction"`
	// DPoSV2EffectiveVotes defines the votes which producer will become a dposV2 effective node
	DPoSV2EffectiveVotes common.Fixed64 `screw:"--dposv2effectivevotes" usage:"defines the minimum votes to active a DposV2 producer"`
	// ExchangeVotes address of votes
	StakePool            string `screw:"--stakepool" usage:"defines DPoSv2 ExchangeVotes address of votes"`
	StakePoolProgramHash *common.Uint168
	// MemoryPoolTxMaximumStayHeight indicates the maximum time of txs can stay in memory pool before rebroadcast again
	MemoryPoolTxMaximumStayHeight uint32 `screw:"--memorypooltxmaximumstayheight" usage:"defines the maximum stay time of memory pool tx before rebroadcast tx again"`
	// SchnorrStartHeight indicates the start height of schnorr withdraw
	SchnorrStartHeight uint32 `screw:"--schnorrstartheight" usage:"defines the start height to support schnorr withdraw transaction"`
	// NormalSchnorrStartHeight indicates the start height of schnorr transfer asset tx
	NormalSchnorrStartHeight uint32 `screw:"--normalschnorrstartheight" usage:"defines the start height to support schnorr transfer asset transaction"`
	// ProducerSchnorrStartHeight indicates the start height of producer related schnorr tx
	ProducerSchnorrStartHeight uint32 `screw:"--producerschnorrstartheight" usage:"defines the start height to support producer related schnorr transaction"`
	// CRSchnorrStartHeight indicates the start height of CR related schnorr tx
	CRSchnorrStartHeight uint32 `screw:"--crschnorrstartheight" usage:"defines the start height to support CR related schnorr transaction"`
	// VotesSchnorrStartHeight indicates the start height of votes related schnorr tx
	VotesSchnorrStartHeight uint32 `screw:"--votesschnorrstartheight" usage:"defines the start height to support votes related schnorr transaction"`
	// MultiExchangeVotesStartHeight indicates the start height of multi-addr exchange votes transaction
	MultiExchangeVotesStartHeight uint32 `screw:"--multiexchangevotesstartheight" usage:"defines the start height to support multi-addr exchange votes transaction"`
	// CrossChainMonitorStartHeight indicates the monitor height of cr cross chain arbitration
	CrossChainMonitorStartHeight uint32 `screw:"--crosschainmonitorstartheight" usage:"defines the start height to monitor cr cross chain transaction"`
	// CrossChainMonitorInterval indicates the interval value of cr cross chain arbitration
	CrossChainMonitorInterval uint32                  `screw:"--crosschainmonitorinterval" usage:"defines the interval cross chain arbitration"`
	CRConfiguration           CRConfiguration         `json:"CRConfiguration"`
	DPoSConfiguration         DPoSConfiguration       `json:"DPoSConfiguration"`
	PowConfiguration          PowConfiguration        `json:"PowConfiguration"`
	RpcConfiguration          RpcConfiguration        `json:"RpcConfiguration"`
	CheckPointConfiguration   CheckPointConfiguration `json:"CheckPointConfiguration"`
}

type CheckPointConfiguration struct {
	// EnableHistory is a switch about recording history of snapshots of checkpoints.
	EnableHistory bool
	// HistoryStartHeight defines the height manager should start to record snapshots of checkpoints.
	HistoryStartHeight uint32
	// DataPath defines root directory path of all checkpoint related files.
	DataPath string
	// NeedSave indicate or not manager should save checkpoints when reached a save point.
	NeedSave bool
}

// DPoSConfiguration defines the DPoS consensus parameters.
type DPoSConfiguration struct {
	EnableArbiter bool `screw:"--arbiter" usage:"indicates where or not to enable DPoS arbiter switch"`
	// Magic defines the magic number used in the DPoS network.
	Magic uint32 `screw:"--dposmagic" usage:"defines the magic number used in the DPoS network"`
	// DPoSIPAddress defines the IP address for the DPoS network.
	IPAddress string `screw:"--dposipaddress" usage:"defines the default IP address for the DPoS network"`
	// DPoSDefaultPort defines the default port for the DPoS network.
	DPoSPort uint16 `screw:"--dposport" usage:"defines the default port for the DPoS network"`
	// ToleranceDuration defines the tolerance duration of the DPoS consensus.
	SignTolerance time.Duration `json:"SignTolerance"`
	// OriginArbiters defines the original arbiters producing the block.
	OriginArbiters []string `screw:"--originarbiters" usage:"defines origin arbiters"`
	// CRCArbiters defines the fixed CRC arbiters producing the block.
	CRCArbiters []string `screw:"--crcarbiters" usage:"defines crc arbiters"`
	// GeneralArbiters defines the number of general(no-CRC) arbiters.
	NormalArbitratorsCount int `screw:"--normalarbitratorscount" usage:"defines the number of general(no-CRC) arbiters"`
	// CandidateArbiters defines the number of needed candidate arbiters.
	CandidatesCount int `screw:"--candidatescount" usage:"defines the number of needed candidate arbiters"`
	// EmergencyInactivePenalty defines the penalty amount the emergency producer takes.
	EmergencyInactivePenalty common.Fixed64 `screw:"--emergencyinactivepenalty" usage:"defines penalty of emergency inactive"`
	// MaxInactiveRounds defines the maximum inactive rounds before producer takes penalty.
	MaxInactiveRounds uint32 `screw:"--maxinactiverounds" usage:"defines the maximum inactive rounds before producer takes penalty"`
	// InactivePenalty defines the penalty amount the producer takes.
	InactivePenalty common.Fixed64 `screw:"--inactivepenalty" usage:"defines penalty of inactive"`
	// InactivePenalty defines the penalty amount the producer takes.
	IllegalPenalty common.Fixed64 `screw:"--illegalpenalty" usage:"defines the num of illegal penalty should be punished "`
	// DPoSV2InactivePenalty defines the penalty amount the producer takes.
	DPoSV2IllegalPenalty common.Fixed64 `screw:"--dposv2illegalpenalty" usage:"defines the num of illegal penalty should be punished"`
	// PreConnectOffset defines the offset blocks to pre-connect to the block producers.
	PreConnectOffset uint32 `screw:"--preconnectoffset" usage:"defines the offset blocks to pre-connect to the block producers"`
	// NoCRCDPOSNodeHeight indicates the height when there is no DPOS node of CRC.
	NoCRCDPOSNodeHeight uint32 `screw:"--nocrcdposnodeheight" usage:"defines the height when there is no DPOS node of CRC"`
	// RandomCandidatePeriod defines the period to get a candidate as DPOS node at random.
	RandomCandidatePeriod uint32 `screw:"--randomcandidateperiod" usage:"defines the period to get a candidate as DPOS node at random"`
	// MaxInactiveRoundsOfRandomNode defines the maximum inactive rounds before the producer at random takes penalty.
	MaxInactiveRoundsOfRandomNode uint32 `screw:"--maxinactiveroundsofrandomnode" usage:"defines the maximum inactive rounds before the random producer takes penalty"`
	// DPOSNodeCrossChainHeight defines the height at which not only CR members are responsible for working across the chain.
	DPOSNodeCrossChainHeight uint32 `screw:"--dposnodecrosschainheight" usage:"defines the height at which not only CR members are responsible for working across the chain"`
	// RevertToPOWInterval defines how long time does it take to revert to POW mode.
	RevertToPOWNoBlockTime int64 `screw:"--reverttopownoblocktime" usage:"defines how long time does it take to revert to POW mode"`
	// StopConfirmBlockTime defines how long time dose it take before stop confirm block.
	StopConfirmBlockTime int64 `screw:"--stopconfirmblocktime" usage:"defines how long time does it take to stop confirm block"`
	// RevertToPOWStartHeight defines the start height to allow to revert to POW mode.
	RevertToPOWStartHeight uint32 `screw:"--reverttopowstartheight" usage:"defines the start height to allow to revert to POW mode"`
	// DPoSV2RewardAccumulateAddress defines the dposv2 reward accumulating address
	DPoSV2RewardAccumulateAddress     string `screw:"--dposv2rewardaccumulateaddress" usage:"defines dposv2 reward accumulate address"`
	DPoSV2RewardAccumulateProgramHash *common.Uint168
	// minimum lock time of DPoS V2 deposit coin
	DPoSV2DepositCoinMinLockTime uint32 `screw:"--dposv2depositcoinminlocktime" usage:"minimum lock time of DPoS V2 deposit coin"`
	// minimum lock time of DPoS V2 votes
	DPoSV2MinVotesLockTime uint32 `screw:"--dposv2minvoteslocktime" usage:"minimum lock time of DPoS V2 votes"`
	// max lock time of DPoS V2 votes
	DPoSV2MaxVotesLockTime uint32 `screw:"--dposv2maxvoteslocktime" usage:"max lock time of DPoS V2 votes"`
	// CRDPoSNodeHotFixHeight indicates the hot fix start height of CR DPoS node
	CRDPoSNodeHotFixHeight uint32 `screw:"--crdposnodehotfixheight" usage:"CRDPoSNodeHotFixHeight indicates the hot fix start height of CR DPoS node"`
	// NFTStartHeight defines the height of NFT started.
	NFTStartHeight uint32 `screw:"--nftstartheight" usage:"the start height of NFT transaction"`
	// NFTV2StartHeight defines the height of NFT 2.0 started, NFT transaction will record the detailed votes information.
	NFTV2StartHeight uint32 `screw:"--NFTV2StartHeight" usage:"the start height of NFT 2.0 transaction"`
	// DexStartHeight defines the height of DEX started.
	DexStartHeight uint32 `screw:"--dexstartheight" usage:"the starting height of Dex support"`
}

type CRConfiguration struct {
	// CheckVoteCRCountHeight defines the height to check count of vote CR.
	CheckVoteCRCountHeight uint32
	// CRMemberCount defines the number of CR committee members
	MemberCount uint32 `screw:"--crmembercount" usage:"defines the number of CR committee members"`
	// CRVotingPeriod defines the duration of voting period which measured by block height
	VotingPeriod uint32 `screw:"--crvotingperiod" usage:"defines the duration of voting period which measured by block height"`
	// CRDutyPeriod defines the duration of a normal duty period which measured by block height
	DutyPeriod uint32 `screw:"--crdutyperiod" usage:"defines the duration of a normal duty period which measured by block height"`
	// ProposalCRVotingPeriod defines the duration of CR voting about a proposal
	ProposalCRVotingPeriod uint32 `screw:"--proposalcrvotingperiod" usage:"defines the duration of CR voting about a proposal"`
	// ProposalPublicVotingPeriod defines the duration of all voters send reject vote about a proposal
	ProposalPublicVotingPeriod uint32 `screw:"--proposalpublicvotingperiod" usage:"defines the duration of all voters send reject vote about a proposal"`
	// CRAgreementCount defines minimum count to let a registered proposal transfer to CRAgreed state.
	CRAgreementCount uint32 `screw:"--cragreementcount" usage:"defines minimum count to let a registered proposal transfer to CRAgreed state"`
	// VoterRejectPercentage defines percentage about voters reject a proposal.
	VoterRejectPercentage float64 `screw:"--voterrejectpercentage" usage:"defines percentage about voters reject a proposal"`
	// CRCAppropriatePercentage defines percentage about CRC appropriation.
	CRCAppropriatePercentage float64 `screw:"--crcappropriatepercentage" usage:"defines percentage about CRC appropriation"`
	// MaxCommitteeProposalCount defines per committee max proposal count
	MaxCommitteeProposalCount uint32 `screw:"--maxcommitteeproposalcount" usage:"defines max count of the proposal that one cr can proposal"`
	// DepositLockupBlocks indicates how many blocks need to wait when cancel producer or CRC was triggered, and can submit return deposit coin request
	DepositLockupBlocks uint32 `screw:"--crdepositlockupblocks" usage:"DepositLockupBlocks indicates how many blocks need to wait when cancel"`
	// SecretaryGeneral defines the secretary general of CR by public key.
	SecretaryGeneral string `screw:"--secretarygeneral" usage:"defines the secretary general of CR"`
	// MaxProposalTrackingCount defines the max count of CRC proposal tracking transaction.
	MaxProposalTrackingCount uint8 `screw:"--maxproposaltrackingcount" usage:"defines the max count of CRC proposal tracking"`
	// RegisterCRByDIDHeight defines the height to support register and update CR by CID and CID.
	RegisterCRByDIDHeight uint32 `screw:"--registercrbydidheight" usage:"defines the height to support register CR by CID"`
	// MaxCRAssetsAddressUTXOCount defines the max UTXOs count of CRFoundation address.
	MaxCRAssetsAddressUTXOCount uint32 `screw:"--maxcrassetsaddressutxocount" usage:"defines the maximum number of utxo cr assets address can have"`
	// MinCRAssetsAddressUTXOCount defines the min UTXOs count of CRFoundation address.
	MinCRAssetsAddressUTXOCount uint32 `screw:"--mincrassetsaddressutxocount" usage:"defines the minimum number of utxo cr assets address can rectify"`
	// CRAssetsRectifyTransactionHeight defines the CR rectify transaction start height
	CRAssetsRectifyTransactionHeight uint32 `screw:"--crassetsrectifytransactionheight" usage:"defines the cr rectify transaction start height"`
	// CRCProposalWithdrawPayloadV1Height defines the CRC proposal withdraw payload height
	CRCProposalWithdrawPayloadV1Height uint32 `screw:"--crcproposalwithdrawpayloadv1height" usage:"defines the crc withdraw proposal payload type v1 accept height"`
	// CRCProposalV1Height defines the height to support ChangeProposalOwner, CloseProposal and SecretaryGeneral proposal.
	CRCProposalV1Height uint32 `screw:"--crcproposalv1height" usage:"defines the ChangeProposalOwner，CloseProposal and SecretaryGeneral crc proposal type accept height"`
	// CRCAddress defines the CRC address which receiving mining rewards.
	CRCAddress     string `screw:"--crcaddress" usage:"crc address of CRC"`
	CRCProgramHash *common.Uint168
	// CRAssetsAddress defines the CR assets address.
	CRAssetsAddress     string `screw:"--crassetsaddress" usage:"defines foundation address of CRC"`
	CRAssetsProgramHash *common.Uint168
	// CRExpensesAddress defines the CR committee address which receiving appropriation from CR assets address.
	CRExpensesAddress     string `screw:"--crexpensesaddress" usage:"defines appropriation address of CRC committee"`
	CRExpensesProgramHash *common.Uint168
	// CRVotingStartHeight defines the height of CR voting started.
	CRVotingStartHeight uint32 `screw:"--crvotingstartheight" usage:"defines the height of CR voting started"`
	// CRCommitteeStartHeight defines the height of CR Committee started.
	CRCommitteeStartHeight uint32 `screw:"--crcommitteestartheight" usage:"defines the height of CR Committee started"`
	// CRClaimDPOSNodeStartHeight defines the height of CR claim DPOS node started.
	CRClaimDPOSNodeStartHeight uint32 `screw:"--crclaimdposnodestartheight" usage:"defines the height of CR claim DPOS node started"`
	// CRClaimDPOSNodePeriod defines the period of CR claim DPOS node.
	CRClaimDPOSNodePeriod uint32 `screw:"--crclaimdposnodeperiod" usage:"defines the period of CR claim DPOS node"`
	// RectifyTxFee defines the fee of cr rectify transaction.
	RectifyTxFee common.Fixed64 `screw:"--rectifytxfee" usage:"defines the fee of cr rectify transaction"`
	// RealWithdrawSingleFee defines the single fee of cr real proposal withdraw transaction.
	RealWithdrawSingleFee common.Fixed64 `screw:"--realwithdrawsinglefee" usage:"defines the single fee of cr real proposal withdraw transaction"`
	// NewP2PProtocolVersionHeight defines the new p2p protocol version message height.
	NewP2PProtocolVersionHeight uint64 `screw:"--newversionheight" usage:"defines the new version message height"`
	// ChangeCommitteeNewCRHeight defines the new arbiter logic after change committee.
	ChangeCommitteeNewCRHeight uint32 `screw:"--changecommitteenewcrheight" usage:"defines the change committee new cr height"`
	// CRCProposalDraftDataStartHeight defines the proposal draft data start height.
	CRCProposalDraftDataStartHeight uint32 `screw:"--crcproposaldraftdatastartheight" usage:"defines the proposal draft data start height"`
	// CRClaimPeriod defines the duration of CR claim DPoS node period which measured by block height
	CRClaimPeriod uint32 `screw:"--crclaimperiod" usage:"defines the duration of CR claim DPoS node"`
}

// PowConfiguration defines the Proof-of-Work parameters.
type PowConfiguration struct {
	PayToAddr    string `screw:"--paytoaddr" usage:"specify the miner reward address"`
	AutoMining   bool   `screw:"--automining" usage:"specify if should open auto mining"`
	MinerInfo    string `json:"MinerInfo"`
	MinTxFee     int    `screw:"--mintxfee" usage:"specify minimum transaction fee"`
	InstantBlock bool   `screw:"--instant" usage:"instant block" usage:"low difficulty to mine block"`
	// powLimit defines the highest allowed proof of work value for a block as a uint256.
	PowLimit *big.Int
	// PowLimitBits defines the highest allowed proof of work value for a block in compact form.
	PowLimitBits uint32
	// TargetTimespan is the desired amount of time that should elapse before the block difficulty requirement
	//is examined to determine how it should be changed in order to maintain the desired block generation rate.
	TargetTimespan time.Duration
	// TargetTimePerBlock is the desired amount of time to generate each block.
	TargetTimePerBlock time.Duration
	// AdjustmentFactor is the adjustment factor used to limit the minimum and maximum amount of adjustment
	//that can occur between difficulty retargets.
	AdjustmentFactor int64
	// RewardPerBlock is the reward amount per block.
	RewardPerBlock common.Fixed64
	// NewRewardPerBlock is the reward amount per block.
	NewRewardPerBlock common.Fixed64
	// CoinbaseMaturity is the number of blocks required before newly mined coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint32
}

// RpcConfiguration defines the JSON-RPC authenticate parameters.
type RpcConfiguration struct {
	User        string   `json:"User"`
	Pass        string   `json:"Pass"`
	WhiteIPList []string `json:"WhiteIPList"`
}

// InstantBlock returns the network parameters for generate instant block.
func (p *Configuration) InstantBlock() *Configuration {
	p.PowConfiguration.PowLimitBits = 0x207fffff
	p.PowConfiguration.TargetTimespan = 10 * time.Second
	p.PowConfiguration.TargetTimePerBlock = 1 * time.Second
	return p
}

func (p *Configuration) Sterilize() *Configuration {
	if p.FoundationAddress != "" {
		p.FoundationProgramHash, _ = common.Uint168FromAddress(
			p.FoundationAddress)
	}
	if p.DestroyELAAddress != "" {
		p.DestroyELAProgramHash, _ = common.Uint168FromAddress(
			p.DestroyELAAddress)
	}
	if p.CRConfiguration.CRCAddress != "" {
		p.CRConfiguration.CRCProgramHash, _ = common.Uint168FromAddress(
			p.CRConfiguration.CRCAddress)
	}
	if p.CRConfiguration.CRAssetsAddress != "" {
		p.CRConfiguration.CRAssetsProgramHash, _ = common.Uint168FromAddress(
			p.CRConfiguration.CRAssetsAddress)
	}
	if p.CRConfiguration.CRExpensesAddress != "" {
		p.CRConfiguration.CRExpensesProgramHash, _ = common.Uint168FromAddress(
			p.CRConfiguration.CRExpensesAddress)
	}
	if p.StakePool != "" {
		p.StakePoolProgramHash, _ = common.Uint168FromAddress(
			p.StakePool)
	}
	if p.DPoSConfiguration.DPoSV2RewardAccumulateAddress != "" {
		p.DPoSConfiguration.DPoSV2RewardAccumulateProgramHash, _ = common.Uint168FromAddress(
			p.DPoSConfiguration.DPoSV2RewardAccumulateAddress)
	}
	p.GenesisBlock = core.GenesisBlock(*p.FoundationProgramHash)
	p.DPoSConfiguration.SignTolerance = p.DPoSConfiguration.SignTolerance * time.Second
	return p
}

// rewardPerBlock calculates the reward for each block by a specified time duration.
func rewardPerBlock(targetTimePerBlock time.Duration) common.Fixed64 {
	blockGenerateInterval := int64(targetTimePerBlock / time.Second)
	generatedBlocksPerYear := 365 * 24 * 60 * 60 / blockGenerateInterval
	return common.Fixed64(float64(inflationPerYear) / float64(generatedBlocksPerYear))
}

func (p *Configuration) GetBlockReward(height uint32) (rewardPerBlock common.Fixed64) {
	if height < p.NewELAIssuanceHeight {
		rewardPerBlock = p.PowConfiguration.RewardPerBlock
	} else {
		rewardPerBlock = p.newRewardPerBlock(2*time.Minute, height)
	}
	return
}

// newRewardPerBlock calculates the reward for each block by a specified time duration.
func (p *Configuration) newRewardPerBlock(targetTimePerBlock time.Duration, height uint32) common.Fixed64 {
	blockGenerateInterval := int64(targetTimePerBlock / time.Second)
	generatedBlocksPerYear := 365 * 24 * 60 * 60 / blockGenerateInterval
	factor := uint32(1)
	if height >= p.HalvingRewardHeight {
		factor = 2 + (height-p.HalvingRewardHeight)/p.HalvingRewardInterval
	}

	return common.Fixed64(float64(newInflationPerYear) / float64(generatedBlocksPerYear) / math.Pow(2, float64(factor-1)))
}

type RPCServiceLevel byte

const (
	ConfigurationPermitted RPCServiceLevel = iota // Allowed  query transaction, and configuration related options.
	MiningPermitted                               // Allowed mining from RPC.
	TransactionPermitted                          // Allowed query and transaction (such as sendrawtransaction) related options.
	WalletPermitted                               // Allowed using wallet related function.
	QueryOnly                                     // Allowed only query related options.
)

func (l RPCServiceLevel) String() string {
	switch l {
	case ConfigurationPermitted:
		return "ConfigurationPermitted"
	case MiningPermitted:
		return "MiningPermitted"
	case TransactionPermitted:
		return "TransactionPermitted"
	case WalletPermitted:
		return "WalletPermitted"
	case QueryOnly:
		return "QueryOnly"
	default:
		return "Unknown"
	}
}

func RPCServiceLevelFromString(str string) RPCServiceLevel {
	switch str {
	case "ConfigurationPermitted":
		return ConfigurationPermitted
	case "MiningPermitted":
		return MiningPermitted
	case "TransactionPermitted":
		return TransactionPermitted
	case "WalletPermitted":
		return WalletPermitted
	case "QueryOnly":
		return QueryOnly
	default:
		return ConfigurationPermitted
	}
}
