# config.json explanation

## Configuration free
- `config.json` file is optional, you can run a `ela` node without a `config.json` file.

## Change active network
Just modify the `ActiveNet` parameter in the `config.json` file.
```json
{
  "Configuration": {
    "ActiveNet": "testnet"
  }
}
```
Default config for `testnet`
- Peer-to-Peer network connect to ELA `testnet`.

## Inline Explanation

```json5
{
  "Configuration": {
    "ActiveNet": "mainnet",       // Network type. Choices: mainnet testnet and regnet
    "Magic": 2017001,             // Magic Number：Segregation for different subnet. No matter the port number, as long as the magic number not matching, nodes cannot talk to each others
    "DNSSeeds": [                 // DNSSeeds. DNSSeeds defines a list of DNS seeds for the network that are used to discover peers.
      "node-mainnet-001.elastos.org:20338"
    ],
    "DisableDNS": false,          // DisableDNS. Disable the DNS seeding function.
    "PermanentPeers": [           // PermanentPeers. Other nodes will look up this seed list to connect to any of those seed in order to get all nodes addresses, if lost connection will try to connect again
      "127.0.0.1:20338"
    ],
    "HttpInfoPort": 20333,        // Local web portal port number. User can go to http://127.0.0.1:10333/info to access the web UI
    "HttpInfoStart": true,        // Whether to enable the HTTPInfo service
    "HttpRestPort": 20334,        // Restful port number
    "HttpRestStart": true,        // Whether to enable the REST service
    "HttpWsPort": 20335,          // Websocket port number
    "HttpWsStart": true,          // Whether to enable the WebSocket service
    "HttpJsonPort": 20336,        // RPC port number
    "EnableRPC": true,            // Enable the RPC service
    "NodePort": 20338,            // P2P port number
    "PrintLevel": 0,              // Log level. Level 0 is the highest, 5 is the lowest
    "MaxLogsSize": 0,             // Max total logs size in MB
    "MaxPerLogSize": 0,           // Max per log file size in MB
    "MinCrossChainTxFee": 10000,  // Minimal cross-chain transaction fee
    "PowConfiguration": {
      "PayToAddr": "",            // Pay bonus to this address. Cannot be empty if AutoMining set to "true"
      "AutoMining": true,         // Start mining automatically? true or false
      "MinerInfo": "ELA",         // No need to change
      "MinTxFee": 100,            // Minimal mining fee
      "InstantBlock": false       // false: high difficulty to mine block  true: low difficulty to mine block
    },
    "RpcConfiguration": {
      "User": "ElaUser",          // Check the username when use rpc interface, null will not check
      "Pass": "Ela123",           // Check the password when use rpc interface, null will not check
      "WhiteIPList": [            // Check if ip in list when use rpc interface, "0.0.0.0" will not check
        "127.0.0.1"
      ]
    },
    "CheckAddressHeight": 88812,   // Before the height will not check that if address is ela address
    "VoteStartHeight": 88812,      // Starting height of statistical voting
    "CRCOnlyDPoSHeight": 1008812,  // The height start DPoS by CRC producers
    "PublicDPoSHeight": 1108812,   // The height start DPoS by CRCProducers and voted producers
    "EnableActivateIllegalHeight": 439000, // The start height to enable activate illegal producer though activate tx
    "EnableUtxoDB": true,          // Whether the db is enabled to store the UTXO
    "EnableCORS": true,            // Enable Cross-Origin Resource Sharing (CORS) is an HTTP-header
    "MaxNodePerHost": 72,          // Limit on the number of node connections
    "TxCacheVolume": 100000,       // Transaction cache size
    "CheckVoteCRCountHeight": 658930,           // Vote to check CR height
    "CustomIDProposalStartHeight": 932530,      // Customize proposal start height
    "MaxReservedCustomIDLength": 255,           // Max Reserved Custom ID Length
    "HalvingRewardHeight": 1051200,             // Halving Reward Height
    "HalvingRewardInterval": 1051200,           // Halving Reward Interval
    "NewELAIssuanceHeight": 919800,             // New ELA Issuance Height
    "SmallCrossTransferThreshold": 100000000,   // Small cross chain transaction threshold
    "ReturnDepositCoinFee": 100,                // Return Deposit Fee
    "NewCrossChainStartHeight": 1032840,        // New Cross Chain Start Height
    "ReturnCrossChainCoinStartHeight": 1032840, // Return Cross Chain Coin Start Height
    "ProhibitTransferToDIDHeight": 1032840,     // Prohibit Transfer To DID Height
    "CrossChainMonitorStartHeight": 2000000,    // Cross Chain Monitor Start Height
    "CrossChainMonitorInterval": 100,           // Cross Chain Monitor Interval
    "DPoSV2StartHeight": 2000000,               // Second edition Dpos start height
    "DPoSV2EffectiveVotes": 8000000000000,      // Minimum valid number of votes
    "StakePool": "",                            // Stake Pool Address
    "SchnorrStartHeight": 2000000,              // Schnorr consensus Start Height
    "DPoSConfiguration": {
      "EnableArbiter": false,                   // EnableArbiter enables the arbiter service.
      "Magic": 2019000,                         // The magic number of DPoS network
      "IPAddress": "192.168.0.1",               // The public network IP address of the node.
      "DPoSPort": 20339,                        // The node prot of DPoS network
      "SignTolerance": 5,                       // The time interval of consensus in seconds
      "OriginArbiters": [                       // The publickey list of arbiters before CRCOnlyDPoSHeight
        "02f3876d0973210d5af7eb44cc11029eb63a102e424f0dc235c60adb80265e426e",
        "03c96f2469b43dd8d0e6fa3041a6cee727e0a3a6658a9c28d91e547d11ba8014a1",
        "036d25d54fb7a40bc7c3e836a26c9e30d5294bc46f6918ad61d0937960f13307bc",
        "0248ddc9ac60f1e5b9e9a26719a8a20e1447e6f2bbb0d31597646f1feb9704f291",
        "02e34e47a06955ef1ec0d325c9edada34a0df6e519530344cc85f5942d061223b3"
      ],
      "CRCArbiters": [                          // The crc arbiters after CRCOnlyDPoSHeight
        "02eae9164bd143eb988fcd4b7a3c9c04a44eb9a009f73e7615e80a5e8ce1e748b8",
        "0294d85959f746b8e6e579458b41eea05afeae50f5a37a037de601673cb24133d9",
        "03b0a3a16edfba8d9c1fed9094431c9f24c78b8ceb04b4b6eeb7706f1686b83499",
        "0222461ae6c9671cad288f10469f9fd759912f257c64524367dc12c40c2bb4046d"
      ],
      "NormalArbitratorsCount": 24,             // The count of voted arbiters
      "CandidatesCount": 72,                    // The count of candidates
      "EmergencyInactivePenalty": 50000000000,  // EmergencyInactivePenalty defines the penalty amount the emergency producer takes.
      "MaxInactiveRounds": 1440,                // MaxInactiveRounds defines the maximum inactive rounds before producer takes penalty.
      "InactivePenalty": 10000000000,           // InactivePenalty defines the penalty amount the producer takes.
      "PreConnectOffset": 360,                  // PreConnectOffset defines the offset blocks to pre-connect to the block producers.
      "IllegalPenalty": 0,                      // DPoS V1 Illegal Penalty
      "DPoSV2RewardAccumulateAddress": "",      // Stake Reward Address
      "DPoSV2DepositCoinMinLockTime": 7200,     // DPoS V2 Deposit Coin Min LockTime
      "DPoSV2MinVotesLockTime": 7200,           // DPoS V2 Min Votes LockTime
      "DPoSV2MaxVotesLockTime": 720000,         // DPoS V2 Max Votes LockTime
      "DPoSV2IllegalPenalty": 20000000000,      // DPoS V2 fine for doing evil
      "CRDPoSNodeHotFixHeight": 0,              // CR DPoS Node HotFix Height
      "NoCRCDPoSNodeHeight": 932530,            // No CRC DPoS Node Height
      "RevertToPOWStartHeight": 932530,         // Revert To POW Start Height
      "RandomCandidatePeriod": 360,             // Random Candidate Period
      "MaxInactiveRoundsOfRandomNode": 288,     // Max Inactive Rounds Of Random Node
      "DPoSNodeCrossChainHeight": 2000000,      // DPoS Node Cross Chain Height
      "RevertToPOWNoBlockTime": 43200,          // Revert To POW Time
      "StopConfirmBlockTime": 39600,            // Block stop confirmation time
    },
    "CRConfiguration": {
      "MemberCount": 12,                        // The count of CR committee members
      "VotingPeriod": 21600,                    // CR Voting StartHeight defines the height of CR voting started
      "DutyPeriod": 262800,                     // CR DutyPeriod defines the duration of a normal duty period which measured by block height
      "CRClaimPeriod": 10080,                   // CR Claim Period
      "DepositLockupBlocks": 2160,              // Deposit Lockup Blocks
      "ProposalCRVotingPeriod": 5040,           // Proposal CR Voting Period
      "ProposalPublicVotingPeriod": 5040,       // Proposal Public Voting Period
      "CRAgreementCount": 8,                    // CR Agreement Count
      "VoterRejectPercentage": 10,              // Voter Reject Percentage
      "CRCAppropriatePercentage": 10,           // CRC Appropriate Percentage
      "MaxCommitteeProposalCount": 128,         // Max Committee Proposal Count
      "SecretaryGeneral": "",                   // Secretary public key
      "MaxProposalTrackingCount": 128,          // Maximum number of proposal traces
      "RegisterCRByDIDHeight": 483500,          // Register CR by DID Height
      "CRAssetsAddress": "",                    // CR Assets Address
      "CRExpensesAddress": "",                  // CR Expenses Address
      "CRClaimDPoSNodePeriod": 10080,           // CR Claim DPoS Node Period
      "CRVotingStartHeight": 1800000,           // CR Voting Start Height defines the height of CR voting started
      "CRCProposalV1Height": 751400,            // Version 1 Height for CR Proposal
      "CRClaimDPoSNodeStartHeight": 751400,     // CR claim node start height
      "NewP2PProtocolVersionHeight": 751400,    // New P2P Protocol Version Height
      "ChangeCommitteeNewCRHeight": 932530,     // The height of the change of members
      "CRCProposalDraftDataStartHeight": 1056600,  // CRC Proposal raft Data Start Height
      "CRCAddress": "",                         // CRC Address
      "MaxCRAssetsAddressUTXOCount": 800,       // Max CR Assets Address UTXO Count
      "MinCRAssetsAddressUTXOCount": 720,       // Min CR Assets Address UTXO Count
      "CRAssetsRectifyTransactionHeight": 751400,  // CR asset trading adjusted height
      "CRCProposalWithdrawPayloadV1Height": 751400,// Version 1 Withdraw Height for CR Proposal
      "CRCommitteeStartHeight": 2000000,        // CR Committee Start Height defines the height of CR Committee started
      "RectifyTxFee":  10000,                   // Rectify transaction Fee
      "RealWithdrawSingleFee": 10000            // Single transaction withdrawal fee
    },
  }
}
```
