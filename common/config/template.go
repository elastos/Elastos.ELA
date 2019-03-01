package config

import "time"

var Template = Configuration{
	ActiveNet:          "mainnet",
	Magic:              7630401,
	SeedList:           []string{"127.0.0.1:30338"},
	HttpInfoPort:       20333,
	HttpInfoStart:      true,
	HttpRestPort:       20334,
	HttpWsPort:         20335,
	HttpJsonPort:       20336,
	NodePort:           20338,
	NodeOpenPort:       20866,
	OpenService:        true,
	PrintLevel:         0,
	MaxLogsSize:        0,
	MaxPerLogSize:      0,
	CertPath:           "./sample-cert.pem",
	KeyPath:            "./sample-cert-key.pem",
	CAPath:             "./sample-ca.pem",
	MaxTxsInBlock:      10000,
	MinCrossChainTxFee: 10000,
	PowConfiguration: PowConfiguration{
		PayToAddr:    "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
		AutoMining:   false,
		MinerInfo:    "ELA",
		MinTxFee:     100,
		InstantBlock: false,
	},
	EnableArbiter: false,
	ArbiterConfiguration: ArbiterConfiguration{
		PublicKey:                "023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
		Magic:                    7630403,
		NodePort:                 30338,
		ProtocolVersion:          0,
		Services:                 0,
		PrintLevel:               1,
		SignTolerance:            5,
		MaxLogsSize:              0,
		MaxPerLogSize:            0,
		MaxConnections:           100,
		NormalArbitratorsCount:   5,
		CandidatesCount:          0,
		EmergencyDuration:        uint32((time.Hour * 24 * 7) / time.Second),
		EmergencyInactivePenalty: 500 * 100000000,
		MaxInactiveRounds:        3,
		InactivePenalty:          100 * 100000000,
		InactiveEliminateCount:   12,
		EnableEventRecord:        false,
	},
	RpcConfiguration: RpcConfiguration{
		User:        "",
		Pass:        "",
		WhiteIPList: []string{"127.0.0.1"},
	},
	HeightVersions: []uint32{
		0,
		88812,
		1008812, //fixme edit height later
		1108812, //fixme edit height later
	},
}
