// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package common

import (
	"errors"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/utils"

	"github.com/urfave/cli"
)

var (
	// Account flags
	AccountWalletFlag = cli.StringFlag{
		Name:  "wallet, w",
		Usage: "wallet `<file>` path",
		Value: account.KeystoreFileName,
	}
	AccountPasswordFlag = cli.StringFlag{
		Name:  "password, p",
		Usage: "wallet password",
	}
	AccountMultiMFlag = cli.IntFlag{
		Name:  "m",
		Usage: "min signature `<number>` of multi signature address",
	}
	AccountMultiPubKeyFlag = cli.StringFlag{
		Name:  "pubkeys, pks",
		Usage: "public key list of multi signature address, separate public keys with comma `,`",
	}

	// Transaction flags
	TransactionFromFlag = cli.StringFlag{
		Name:  "from",
		Usage: "the sender `<address>` of the transaction",
	}
	TransactionToFlag = cli.StringFlag{
		Name:  "to",
		Usage: "the recipient `<address>` of the transaction",
	}
	TransactionToManyFlag = cli.StringFlag{
		Name:  "tomany",
		Usage: "the `<file>` path that contains multi-recipients and amount",
	}
	TransactionAmountFlag = cli.StringFlag{
		Name:  "amount",
		Usage: "the transfer `<amount>` of the transaction",
	}
	TransactionClaimAmountFlag = cli.Int64Flag{
		Name:  "claimamount",
		Usage: "the amount to claim of dposv2 reward",
		Value: 0,
	}
	TransactionFeeFlag = cli.StringFlag{
		Name:  "fee",
		Usage: "the transfer `<fee>` of the transaction",
	}
	TransactionOutputLockFlag = cli.StringFlag{
		Name:  "outputlock",
		Usage: "the `<lock height>` to specify when the received asset can be spent",
	}
	TransactionTxLockFlag = cli.StringFlag{
		Name:  "txlock",
		Usage: "the `<lock height>` to specify when the transaction can be packaged",
	}
	TransactionHexFlag = cli.StringFlag{
		Name:  "hex",
		Usage: "the transaction content in hex string format to be sign or send",
	}
	TransactionFileFlag = cli.StringFlag{
		Name:  "file, f",
		Usage: "the file path to specify a transaction file path with the hex string content to be sign",
	}
	TransactionNodePublicKeyFlag = cli.StringFlag{
		Name:  "nodepublickey",
		Usage: "the node public key of an arbitrator which have been inactivated (default: same as owner public key)",
	}
	TransactionForFlag = cli.StringFlag{
		Name:  "for",
		Usage: "the `<file>` path that holds the list of candidates",
	}
	TransactionSAddressFlag = cli.StringFlag{
		Name:  "saddress",
		Usage: "the locked `<address>` on main chain represents one side chain",
	}
	TransactionNickNameFlag = cli.StringFlag{
		Name:  "nickname",
		Usage: "the nick name of producer or cr council member",
	}
	TransactionUrlFlag = cli.StringFlag{
		Name:  "url",
		Usage: "the url of producer or cr council member",
	}
	TransactionLocationFlag = cli.Uint64Flag{
		Name:  "location",
		Usage: "localtion code of producer or cr council member",
	}
	TransactionNetAddressFlag = cli.StringFlag{
		Name:  "netaddress",
		Usage: "ip address of producer",
	}
	TransactionStakeUntilFlag = cli.UintFlag{
		Name:  "stakeuntil",
		Usage: "stake until this block height",
	}
	TransactionPayloadFlag = cli.StringFlag{
		Name:  "payload",
		Usage: "proposal payload",
	}
	TransactionCategoryDataFlag = cli.StringFlag{
		Name:  "category",
		Usage: "proposal category data",
	}
	TransactionDraftHashFlag = cli.StringFlag{
		Name:  "drafthash",
		Usage: "proposal draft hash",
	}
	TransactionDraftDataFlag = cli.StringFlag{
		Name:  "draftdata",
		Usage: "proposal draft data",
	}
	TransactionBudgetsFlag = cli.StringFlag{
		Name:  "budgets",
		Usage: "proposal budgets, eg: --budgets \"type1,stage1,amount1|type2,stage2,amount2\"",
	}
	TransactionRecipientFlag = cli.StringFlag{
		Name:  "recipient",
		Usage: "proposal recipient address",
	}
	TransactionTargetProposalHashFlag = cli.StringFlag{
		Name:  "targetproposalhash",
		Usage: "proposal target proposal hash",
	}
	TransactionReservedCustomIDListFlag = cli.StringFlag{
		Name:  "reservedcustomidlist",
		Usage: "proposal reserved custom id list, eg: --reservedcustomidlist \"id1|id2|id3\"",
	}
	TransactionReceivedCustomIDListFlag = cli.StringFlag{
		Name:  "receivedcustomidlist",
		Usage: "proposal received custom id list, eg: --receivedcustomidlist \"id1|id2|id3\"",
	}
	TransactionReceiverDIDFlag = cli.StringFlag{
		Name:  "receiverdid",
		Usage: "proposal receiver did",
	}
	TransactionCustomIDFeeRateInfoFlag = cli.StringFlag{
		Name:  "customidfeerate",
		Usage: "proposal custom id fee rate info, eg: --customidfeerate \"rate|height\"",
	}
	TransactionNewRecipientFlag = cli.StringFlag{
		Name:  "newrecipient",
		Usage: "proposal new recipient",
	}
	TransactionOwnerPublicKeyFlag = cli.StringFlag{
		Name:  "ownerpubkey",
		Usage: "proposal owner public key",
	}
	TransactionNewOwnerPublicKeyFlag = cli.StringFlag{
		Name:  "newownerpubkey",
		Usage: "proposal new owner public key",
	}
	TransactionSecretaryPublicKeyFlag = cli.StringFlag{
		Name:  "secretarypublickey",
		Usage: "proposal secretary public key",
	}
	TransactionSecretaryDIDFlag = cli.StringFlag{
		Name:  "secretarydid",
		Usage: "proposal secretary did",
	}
	TransactionSignatureFlag = cli.StringFlag{
		Name:  "signature",
		Usage: "signature hex-string",
	}
	TransactionOwnerSignatureFlag = cli.StringFlag{
		Name:  "ownersignature",
		Usage: "proposal owner signature",
	}
	TransactionNewOwnerSignatureFlag = cli.StringFlag{
		Name:  "newownersignature",
		Usage: "proposal new owner signature",
	}
	TransactionCRCouncilMemberDIDFlag = cli.StringFlag{
		Name:  "crcmemberdid",
		Usage: "proposal cr council member did",
	}
	TransactionCRCouncilMemberSignatureFlag = cli.StringFlag{
		Name:  "crcmembersignature",
		Usage: "proposal cr council member signature",
	}
	TransactionRegisterSideChainFlag = cli.StringFlag{
		Name:  "sidechaininfo",
		Usage: "proposal register side chain info, eg: --sidechaininfo \"name|magic|genesisHash|exchangeRate|effectiveHeight|resourcePath\"",
	}
	TransactionProposalHashFlag = cli.StringFlag{
		Name:  "proposalhash",
		Usage: "proposal hash",
	}
	TransactionVoteResultFlag = cli.StringFlag{
		Name:  "voteresult",
		Usage: "vote result, eg: --voteresult=`<result>`, `<result>` can be 0:approve 1:reject 2:abstain",
	}
	TransactionOpinionHashFlag = cli.StringFlag{
		Name:  "opinionhash",
		Usage: "opinion hash",
	}
	TransactionOpinionDataFlag = cli.StringFlag{
		Name:  "opiniondata",
		Usage: "opinion data",
	}
	TransactionDIDFlag = cli.StringFlag{
		Name:  "did",
		Usage: "did string",
	}
	TransactionMessageHashFlag = cli.StringFlag{
		Name:  "messagehash",
		Usage: "message hash",
	}
	TransactionMessageDataFlag = cli.StringFlag{
		Name:  "messagedata",
		Usage: "message data",
	}
	TransactionProposalTrackingTypeFlag = cli.StringFlag{
		Name:  "type",
		Usage: "proposal tracking type, eg: --type=`<type>`, `<type>` can be 0:common 1:progress 2:rejected 3:terminated 4:changeOwner 5:finalized",
	}
	TransactionSecretaryGeneralOpinionHashFlag = cli.StringFlag{
		Name:  "secretarygeneralopinionhash",
		Usage: "opinion hash of secretary general",
	}
	TransactionSecretaryGeneralOpinionDataFlag = cli.StringFlag{
		Name:  "secretarygeneralopiniondata",
		Usage: "opinion data of secretary general",
	}
	TransactionSecretaryGeneralSignatureFlag = cli.StringFlag{
		Name:  "secretarygeneralsignature",
		Usage: "secretary general signature",
	}
	TransactionDigestFlag = cli.StringFlag{
		Name:  "digest",
		Usage: "digest hex-string",
	}

	// RPC flags
	RPCUserFlag = cli.StringFlag{
		Name:  "rpcuser",
		Usage: "username for JSON-RPC connections",
	}
	RPCPasswordFlag = cli.StringFlag{
		Name:  "rpcpassword",
		Usage: "password for JSON-RPC connections",
	}
	RPCPortFlag = cli.StringFlag{
		Name:  "rpcport",
		Usage: "JSON-RPC server listening port `<number>`",
	}
	RPCIpFlag = cli.StringFlag{
		Name:  "rpcip",
		Usage: "JSON-RPC server listening ip `<string>`",
	}
	EnableRPCFlag = cli.StringFlag{
		Name:  "server",
		Usage: "decide if open JSON-RPC server or not",
	}
	RPCAllowedIPsFlag = cli.StringFlag{
		Name:  "rpcips",
		Usage: "white IP list allowed to access RPC server",
	}

	// Info flags
	InfoStartFlag = cli.IntFlag{
		Name:  "start",
		Usage: "the start index of producers",
		Value: 0,
	}
	InfoLimitFlag = cli.Int64Flag{
		Name:  "limit",
		Usage: "the limit count of producers",
		Value: -1,
	}
	InfoProducerStateFlag = cli.StringFlag{
		Name:  "state",
		Usage: "the producer state you want",
	}

	// Config flags
	TestNetFlag = cli.StringFlag{
		Name:  "testnet",
		Usage: "specify network type to test net",
		Value: defaultConfigPath,
	}
	RegTestFlag = cli.StringFlag{
		Name:  "regtest",
		Usage: "specify network type to reg test net",
		Value: defaultConfigPath,
	}
	ConfigFileFlag = cli.StringFlag{
		Name:  "conf",
		Usage: "config `<file>` path, ",
		Value: defaultConfigPath,
	}
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "block data and logs storage `<path>`",
		Value: defaultDataDir,
	}
	MagicFlag = cli.StringFlag{
		Name:  "magic",
		Usage: "magic number for node to initialize p2p connection",
	}
	PrintLevelFlag = cli.StringFlag{
		Name:  "printlevel",
		Usage: "level to print log",
	}
	EnableDnsFlag = cli.StringFlag{
		Name:  "dnsseed",
		Usage: "enable dns seeds for node to initialize p2p connection",
	}
	DnsSeedFlag = cli.StringFlag{
		Name:  "dns",
		Usage: "dns seeds for node to initialize p2p connection",
	}
	PeersFlag = cli.StringFlag{
		Name:  "peers",
		Usage: "peers seeds for node to initialize p2p connection",
	}
	PortFlag = cli.StringFlag{
		Name:  "port",
		Usage: "default peer-to-peer port for the network",
	}
	InfoPortFlag = cli.StringFlag{
		Name:  "infoport",
		Usage: "port for the http info server",
	}
	RestPortFlag = cli.StringFlag{
		Name:  "restport",
		Usage: "port for the http restful server",
	}
	WsPortFlag = cli.StringFlag{
		Name:  "wsport",
		Usage: "port for the http web socket server",
	}
	InstantBlockFlag = cli.StringFlag{
		Name:  "instant",
		Usage: "specify if need to generate instant block",
	}
	FoundationAddrFlag = cli.StringFlag{
		Name:  "foundation",
		Usage: "specify the foundation address",
	}
	DIDSideChainAddressFlag = cli.StringFlag{
		Name:  "didsidechainaddress",
		Usage: "specify the did sidechain address",
	}
	PayToAddrFlag = cli.StringFlag{
		Name:  "paytoaddr",
		Usage: "specify the miner reward address",
	}
	AutoMiningFlag = cli.StringFlag{
		Name:  "automining",
		Usage: "specify if should open auto mining",
	}
	MinTxFeeFlag = cli.StringFlag{
		Name:  "mintxfee",
		Usage: "specify minimum transaction fee",
	}
	VoteStartHeightFlag = cli.StringFlag{
		Name: "votestartheight",
		Usage: "indicates the height of starting register producer and " +
			"vote related",
	}
	CheckAddressHeightFlag = cli.StringFlag{
		Name:  "checkaddressheight",
		Usage: "defines the height begin to check output hash",
	}
	CheckRewardHeightFlag = cli.StringFlag{
		Name:  "checkrewardheight",
		Usage: "defines the height begin to check reward",
	}
	EnableArbiterFlag = cli.StringFlag{
		Name:  "arbiter",
		Usage: "indicates where or not to enable DPoS arbiter switch",
	}
	CRCOnlyDPOSHeightFlag = cli.StringFlag{
		Name: "crconlydposheight",
		Usage: "(H1) indicates the height of DPOS consensus begins with only " +
			"CRC producers participate in producing block",
	}
	PublicDPOSHeightFlag = cli.StringFlag{
		Name: "publicdposheight",
		Usage: "(H2) indicates the height when public registered and elected " +
			"producers participate in DPOS consensus",
	}
	IllegalPenaltyFlag = cli.StringFlag{
		Name:  "illegalpenalty",
		Usage: "defines the num of illegal penalty should be punished ",
	}
	CRCommitteeStartHeightFlag = cli.StringFlag{
		Name:  "crcommitteestartheight",
		Usage: "defines the height of CR Committee started",
	}
	CRClaimDPOSNodeStartHeightFlag = cli.StringFlag{
		Name:  "crclaimdposnodestartheight",
		Usage: "defines the height of CR claim DPOS node started",
	}
	CRClaimDPOSNodePeriodFlag = cli.StringFlag{
		Name:  "crclaimdposnodeperiod",
		Usage: "defines the period of CR claim DPOS node",
	}
	CRVotingStartHeightFlag = cli.StringFlag{
		Name:  "crvotingstartheight",
		Usage: "defines the height of CR voting started",
	}
	MaxCommitteeProposalCount = cli.StringFlag{
		Name:  "maxcommitteeproposalcount",
		Usage: "defines max count of the proposal that one cr can proposal",
	}
	MaxNodePerHost = cli.StringFlag{
		Name:  "maxnodeperhost",
		Usage: "defines max nodes that one host can establish",
	}
	VoteStatisticsHeightFlag = cli.StringFlag{
		Name:  "votestatisticsheight",
		Usage: "defines the height to fix vote statistics error",
	}
	EnableActivateIllegalHeightFlag = cli.StringFlag{
		Name: "enableactivateillegalheight",
		Usage: "defines the start height to enable activate illegal producer" +
			" though activate tx",
	}
	DPoSMagicFlag = cli.StringFlag{
		Name:  "dposmagic",
		Usage: "defines the magic number used in the DPoS network",
	}
	DPoSIPAddressFlag = cli.StringFlag{
		Name:  "dposipaddress",
		Usage: "defines the default IP address for the DPoS network",
	}
	DPoSPortFlag = cli.StringFlag{
		Name:  "dposport",
		Usage: "defines the default port for the DPoS network",
	}
	SecretaryGeneralFlag = cli.StringFlag{
		Name:  "secretarygeneral",
		Usage: "defines the secretary general of CR",
	}
	MaxProposalTrackingCountFlag = cli.StringFlag{
		Name:  "maxproposaltrackingcount",
		Usage: "defines the max count of CRC proposal tracking",
	}
	OriginArbitersFlag = cli.StringFlag{
		Name:  "originarbiters",
		Usage: "defines origin arbiters",
	}
	CRCArbitersFlag = cli.StringFlag{
		Name:  "crcarbiters",
		Usage: "defines crc arbiters",
	}
	PreConnectOffsetFlag = cli.StringFlag{
		Name:  "preconnectoffset",
		Usage: "defines the offset blocks to pre-connect to the block producers",
	}
	NormalArbitratorsCountFlag = cli.StringFlag{
		Name:  "normalarbitratorscount",
		Usage: "defines the number of general(no-CRC) arbiters",
	}
	CandidatesCountFlag = cli.StringFlag{
		Name:  "candidatescount",
		Usage: "defines the number of needed candidate arbiters",
	}
	MaxInactiveRoundsFlag = cli.StringFlag{
		Name:  "maxinactiverounds",
		Usage: "defines the maximum inactive rounds before producer takes penalty",
	}
	InactivePenaltyFlag = cli.StringFlag{
		Name:  "inactivepenalty",
		Usage: "defines penalty of inactive",
	}
	EmergencyInactivePenaltyFlag = cli.StringFlag{
		Name:  "emergencyinactivepenalty",
		Usage: "defines penalty of emergency inactive",
	}
	DPoSV2MinVotesLockTimeFlag = cli.StringFlag{
		Name:  "dposv2minvoteslocktime",
		Usage: "minimum lock time of DPoS V2 votes",
	}
	DPoSV2MaxVotesLockTimeFlag = cli.StringFlag{
		Name:  "dposv2maxvoteslocktime",
		Usage: "max lock time of DPoS V2 votes",
	}
	CRMemberCountFlag = cli.StringFlag{
		Name:  "crmembercount",
		Usage: "defines the number of CR committee members",
	}
	CRDutyPeriodFlag = cli.StringFlag{
		Name: "crdutyperiod",
		Usage: "defines the duration of a normal duty period which measured " +
			"by block height",
	}
	CRDepositLockupBlocksFlag = cli.StringFlag{
		Name:  "crdepositlockupblocks",
		Usage: "DepositLockupBlocks indicates how many blocks need to wait when cancel",
	}
	CRVotingPeriodFlag = cli.StringFlag{
		Name: "crvotingperiod",
		Usage: "defines the duration of voting period which measured by " +
			"block height",
	}
	ProposalCRVotingPeriodFlag = cli.StringFlag{
		Name:  "proposalcrvotingperiod",
		Usage: "defines the duration of CR voting about a proposal",
	}
	ProposalPublicVotingPeriodFlag = cli.StringFlag{
		Name: "proposalpublicvotingperiod",
		Usage: "defines the duration of all voters send reject vote about " +
			"a proposal",
	}
	CRAgreementCountFlag = cli.StringFlag{
		Name: "cragreementcount",
		Usage: "defines minimum count to let a registered proposal transfer " +
			"to CRAgreed state",
	}
	VoterRejectPercentageFlag = cli.StringFlag{
		Name:  "voterrejectpercentage",
		Usage: "defines percentage about voters reject a proposal",
	}
	CRCProposalHashFlag = cli.StringFlag{
		Name:  "proposalhash",
		Usage: "the `<proposalhash>` of the transaction",
	}
	CRCProposalStageFlag = cli.StringFlag{
		Name:  "stage",
		Usage: "the  `<stage>` of the proposal",
	}
	CRCCommiteeAddrFlag = cli.StringFlag{
		Name:  "crccommiteeaddr",
		Usage: "the  `<crccommiteeaddr>`",
	}
	CRCAppropriatePercentageFlag = cli.StringFlag{
		Name:  "crcappropriatepercentage",
		Usage: "defines percentage about CRC appropriation",
	}
	CRAssetsAddressFlag = cli.StringFlag{
		Name:  "crassetsaddress",
		Usage: "defines foundation address of CRC",
	}
	CRExpensesAddressFlag = cli.StringFlag{
		Name:  "crexpensesaddress",
		Usage: "defines appropriation address of CRC committee",
	}
	RegisterCRByDIDHeightFlag = cli.StringFlag{
		Name:  "registercrbydidheight",
		Usage: "defines the height to support register CR by CID",
	}
	ProhibitTransferToDIDHeightFlag = cli.StringFlag{
		Name:  "prohibittransfertodidheight",
		Usage: "defines the height to prohibit transfer to did",
	}
	MaxCRAssetsAddressUTXOCount = cli.StringFlag{
		Name:  "maxcrassetsaddressutxocount",
		Usage: "defines the maximum number of utxo cr assets address can have ",
	}
	MinCRAssetsAddressUTXOCount = cli.StringFlag{
		Name:  "mincrassetsaddressutxocount",
		Usage: "defines the minimum number of utxo cr assets address can rectify",
	}
	CRAssetsRectifyTransactionHeight = cli.StringFlag{
		Name:  "crassetsrectifytransactionheight",
		Usage: "defines the cr rectify transaction start height",
	}
	CRCProposalWithdrawPayloadV1Height = cli.StringFlag{
		Name:  "crcproposalwithdrawpayloadv1height",
		Usage: "defines the crc withdraw proposal payload type v1 accept height",
	}
	CRCProposalV1Height = cli.StringFlag{
		Name: "crcproposalv1height",
		Usage: "defines the ChangeProposalOwner，CloseProposal and " +
			"SecretaryGeneral crc proposal type accept height",
	}
	RectifyTxFee = cli.StringFlag{
		Name:  "rectifytxfee",
		Usage: "defines the fee of cr rectify transaction",
	}
	RealWithdrawSingleFee = cli.StringFlag{
		Name:  "realwithdrawsinglefee",
		Usage: "defines the single fee of cr real proposal withdraw transaction",
	}
	NewVersionHeight = cli.StringFlag{
		Name:  "newversionheight",
		Usage: "defines the new version message height",
	}

	ChangeCommitteeNewCRHeight = cli.StringFlag{
		Name:  "changecommitteenewcrheight",
		Usage: "defines the change committee new cr height",
	}

	CRCProposalDraftDataStartHeight = cli.StringFlag{
		Name:  "crcproposaldraftdatastartheight",
		Usage: "defines the proposal draft data start height",
	}

	CRClaimPeriodFlag = cli.StringFlag{
		Name:  "crclaimperiod",
		Usage: "defines the duration of CR claim DPoS node",
	}

	CustomIDProposalStartHeight = cli.StringFlag{
		Name:  "CustomIDProposalStartHeight",
		Usage: "defines the height to allow custom ID related transaction",
	}

	MaxReservedCustomIDLength = cli.StringFlag{
		Name:  "maxreservedcustomidlength",
		Usage: "defines the max count of reserved custom iid list per tx",
	}

	NoCRCDPOSNodeHeight = cli.StringFlag{
		Name:  "nocrcdposnodeheight",
		Usage: "defines the height when there is no DPOS node of CRC",
	}

	RandomCandidatePeriod = cli.StringFlag{
		Name:  "randomcandidateperiod",
		Usage: "defines the period to get a candidate as DPOS node at random",
	}

	MaxInactiveRoundsOfRandomNode = cli.StringFlag{
		Name:  "maxinactiveroundsofrandomnode",
		Usage: "defines the maximum inactive rounds before the random producer takes penalty",
	}

	DPOSNodeCrossChainHeight = cli.StringFlag{
		Name:  "dposnodecrosschainheight",
		Usage: "defines the height at which not only CR members are responsible for working across the chain",
	}

	RevertToPOWNoBlockTimeFlag = cli.StringFlag{
		Name:  "reverttopownoblocktime",
		Usage: "defines how long time does it take to revert to POW mode",
	}

	StopConfirmBlockTimeFlag = cli.StringFlag{
		Name:  "stopconfirmblocktime",
		Usage: "defines how long time does it take to stop confirm block",
	}

	RevertToPOWStartHeightFlag = cli.StringFlag{
		Name:  "reverttopowstartheight",
		Usage: "defines the start height to allow to revert to POW mode",
	}

	HalvingRewardHeightFlag = cli.StringFlag{
		Name:  "halvingrewardheight",
		Usage: "defines height of having reward",
	}

	HalvingRewardIntervalFlag = cli.StringFlag{
		Name:  "halvingrewardinterval",
		Usage: "defines interval of having reward",
	}

	NewELAIssuanceHeightFlag = cli.StringFlag{
		Name:  "newelaissuanceheight",
		Usage: "defines height of using the new ela issuance (2000w)",
	}

	SmallCrossTransferThreshold = cli.StringFlag{
		Name:  "smallcrosstransferthreshold",
		Usage: "defines the minimum amount of transfer consider as small cross transfer",
	}

	ReturnDepositCoinFeeFlag = cli.StringFlag{
		Name:  "returndepositcoinfee",
		Usage: "defines the fee of return cross chain deposit coin",
	}

	NewCrossChainStartHeightFlag = cli.StringFlag{
		Name:  "newcrosschainstartheight",
		Usage: "defines the height to only support TransferCrossChainAsset v1",
	}

	ReturnCrossChainCoinStartHeightFlag = cli.StringFlag{
		Name:  "returncrosschaincoinstartheight",
		Usage: "defines the start height to support ReturnCrossChainDepositCoin transaction",
	}

	DposV2StartHeightFlag = cli.StringFlag{
		Name:  "dposv2startheight",
		Usage: "defines the start height to support DposV2 transaction",
	}

	DposV2EffectiveVotesFlag = cli.StringFlag{
		Name:  "dposv2effectivevotes",
		Usage: "defines the minimum votes to active a DposV2 producer",
	}

	DposV2RewardAccumulateAddressFlag = cli.StringFlag{
		Name:  "dposv2rewardaccumulateaddress",
		Usage: "defines dposv2 reward accumulate address",
	}

	StakePoolFlag = cli.StringFlag{
		Name:  "stakepool",
		Usage: "defines the stake address of DPoS v2 votes",
	}

	SchnorrStartHeightFlag = cli.StringFlag{
		Name:  "schnorrstartheight",
		Usage: "defines the start height to support schnorr transaction",
	}

	CRDPoSNodeHotFixHeightFlag = cli.StringFlag{
		Name:  "crdposnodehotfixheight",
		Usage: "CRDPoSNodeHotFixHeight indicates the hot fix start height of CR DPoS node",
	}

	CrossChainMonitorStartHeightFlag = cli.StringFlag{
		Name:  "crosschainmonitorstartheight",
		Usage: "defines the start height to monitor cr cross chain transaction",
	}

	CrossChainMonitorIntervalFlag = cli.StringFlag{
		Name:  "crosschainmonitorinterval",
		Usage: "defines the interval cross chain arbitration",
	}
)

// MoveRPCFlags finds the rpc argument and moves it to the front
// of the argument array.
func MoveRPCFlags(args []string) ([]string, error) {
	newArgs := args[:1]
	cacheArgs := make([]string, 0)

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--rpcip":
			fallthrough
		case "--rpcport":
			fallthrough
		case "--rpcuser":
			fallthrough
		case "--rpcpassword":
			newArgs = append(newArgs, args[i])
			if i == len(args)-1 {
				return nil, errors.New("invalid flag " + args[i])
			}
			newArgs = append(newArgs, args[i+1])
			i++
		default:
			cacheArgs = append(cacheArgs, args[i])
		}
	}

	newArgs = append(newArgs, cacheArgs...)
	return newArgs, nil
}

// GetFlagPassword gets node's wallet password from command line or user input
func GetFlagPassword(c *cli.Context) ([]byte, error) {
	flagPassword := c.String("password")
	password := []byte(flagPassword)
	if flagPassword == "" {
		return utils.GetPassword()
	}

	return password, nil
}
