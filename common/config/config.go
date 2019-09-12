// Copyright (c) 2017-2019 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
// 

package config

import (
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/utils/elalog"
)

var (
	Parameters *Configuration
)

// PowConfiguration defines the Proof-of-Work parameters.
type PowConfiguration struct {
	PayToAddr    string `json:"PayToAddr"`
	AutoMining   bool   `json:"AutoMining"`
	MinerInfo    string `json:"MinerInfo"`
	MinTxFee     int    `json:"MinTxFee"`
	InstantBlock bool   `json:"InstantBlock"`
}

// RpcConfiguration defines the JSON-RPC authenticate parameters.
type RpcConfiguration struct {
	User        string   `json:"User"`
	Pass        string   `json:"Pass"`
	WhiteIPList []string `json:"WhiteIPList"`
}

// Configuration defines the configurable parameters to run a ELA node.
type Configuration struct {
	ActiveNet                   string            `json:"ActiveNet"`
	Magic                       uint32            `json:"Magic"`
	DNSSeeds                    []string          `json:"DNSSeeds"`
	DisableDNS                  bool              `json:"DisableDNS"`
	PermanentPeers              []string          `json:"PermanentPeers"`
	HttpInfoPort                uint16            `json:"HttpInfoPort"`
	HttpInfoStart               bool              `json:"HttpInfoStart"`
	HttpRestPort                int               `json:"HttpRestPort"`
	HttpRestStart               bool              `json:"HttpRestStart"`
	HttpWsPort                  int               `json:"HttpWsPort"`
	HttpWsStart                 bool              `json:"HttpWsStart"`
	HttpJsonPort                int               `json:"HttpJsonPort"`
	EnableRPC                   bool              `json:"EnableRPC"`
	NodePort                    uint16            `json:"NodePort"`
	PrintLevel                  elalog.Level      `json:"PrintLevel"`
	MaxLogsSize                 int64             `json:"MaxLogsSize"`
	MaxPerLogSize               int64             `json:"MaxPerLogSize"`
	RestCertPath                string            `json:"RestCertPath"`
	RestKeyPath                 string            `json:"RestKeyPath"`
	MinCrossChainTxFee          common.Fixed64    `json:"MinCrossChainTxFee"`
	FoundationAddress           string            `json:"FoundationAddress"`
	CRCAddress                  string            `json:"CRCAddress"`
	PowConfiguration            PowConfiguration  `json:"PowConfiguration"`
	RpcConfiguration            RpcConfiguration  `json:"RpcConfiguration"`
	DPoSConfiguration           DPoSConfiguration `json:"DPoSConfiguration"`
	CRConfiguration             CRConfiguration   `json:"CRConfiguration"`
	CheckAddressHeight          uint32            `json:"CheckAddressHeight"`
	VoteStartHeight             uint32            `json:"VoteStartHeight"`
	CRCOnlyDPOSHeight           uint32            `json:"CRCOnlyDPOSHeight"`
	PublicDPOSHeight            uint32            `json:"PublicDPOSHeight"`
	EnableActivateIllegalHeight uint32            `json:"EnableActivateIllegalHeight"`
	CRVotingStartHeight         uint32            `json:"CRVotingStartHeight"`
	CRCommitteeStartHeight      uint32            `json:"CRCommitteeStartHeight"`
	CheckRewardHeight           uint32            `json:"CheckRewardHeight"`
	ProfilePort                 uint32            `json:"ProfilePort"`
	MaxBlockSize                uint32            `json"MaxBlockSize"`
	EnableHistory               bool              `json:"EnableHistory"`
	HistoryStartHeight          uint32            `json:"HistoryStartHeight"`
	EnableUtxoDB                bool              `json:"EnableUtxoDB"`
}

// DPoSConfiguration defines the DPoS consensus parameters.
type DPoSConfiguration struct {
	EnableArbiter            bool           `json:"EnableArbiter"`
	Magic                    uint32         `json:"Magic"`
	IPAddress                string         `json:"IPAddress"`
	DPoSPort                 uint16         `json:"DPoSPort"`
	SignTolerance            time.Duration  `json:"SignTolerance"`
	OriginArbiters           []string       `json:"OriginArbiters"`
	CRCArbiters              []string       `json:"CRCArbiters"`
	NormalArbitratorsCount   int            `json:"NormalArbitratorsCount"`
	CandidatesCount          int            `json:"CandidatesCount"`
	EmergencyInactivePenalty common.Fixed64 `json:"EmergencyInactivePenalty"`
	MaxInactiveRounds        uint32         `json:"MaxInactiveRounds"`
	InactivePenalty          common.Fixed64 `json:"InactivePenalty"`
	PreConnectOffset         uint32         `json:"PreConnectOffset"`
}

type CRConfiguration struct {
	MemberCount                uint32  `json:"MemberCount"`
	VotingPeriod               uint32  `json:"VotingPeriod"`
	DutyPeriod                 uint32  `json:"DutyPeriod"`
	ProposalCRVotingPeriod     uint32  `json:"ProposalCRVotingPeriod"`
	ProposalPublicVotingPeriod uint32  `json:"ProposalPublicVotingPeriod"`
	CRAgreementCount           uint32  `json:"CRAgreementCount"`
	VoterRejectPercentage      float64 `json:"VoterRejectPercentage"`
}
