// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package common

import (
	"errors"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common/config"
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
	TransactionMemoType = cli.StringFlag{
		Name:  "type",
		Usage: "the poll type `<byte>` from 0 to 256",
	}
	TransactionMemoVotingTime = cli.StringFlag{
		Name:  "votingtime",
		Usage: "the poll voting time `<uint32>` in seconds",
	}
	TransctionMemoDescription = cli.StringFlag{
		Name:  "description",
		Usage: "the poll description `<string>`",
	}
	TransactionMemoChoices = cli.StringSliceFlag{
		Name:  "choices",
		Usage: "the poll choices `<[]string>`, separate choices with comma ',' ",
	}
	TransactionMemoUrl = cli.StringFlag{
		Name:  "url",
		Usage: "the poll url `<string>`",
	}
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
	TransactionClaimAmountFlag = cli.StringFlag{
		Name:  "claimamount",
		Usage: "the amount to claim of dposv2 reward",
		Value: "",
	}
	TransactionReferKeysFlag = cli.StringFlag{
		Name:  "referkeys",
		Usage: "the refer key is the hash of detailed DPoS 2.0 node votes information",
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
	VoteTypeFlag = cli.Uint64Flag{
		Name:  "votetype",
		Usage: "the list of votes",
	}
	CandidatesFlag = cli.StringFlag{
		Name:  "candidates",
		Usage: "the list of candidates",
	}
	VotesFlag = cli.StringFlag{
		Name:  "votes",
		Usage: "the list of votes",
	}
	StakeUntilListFlag = cli.StringFlag{
		Name:  "stakeuntils",
		Usage: "the list of stake until",
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
	RPCUrlFlag = cli.StringFlag{
		Name:  "rpcurl",
		Usage: "JSON-RPC server listening url `<string>`",
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
		Value: config.ConfigFile,
	}
	RegTestFlag = cli.StringFlag{
		Name:  "regtest",
		Usage: "specify network type to reg test net",
		Value: config.ConfigFile,
	}
	ConfigFileFlag = cli.StringFlag{
		Name:  "conf",
		Usage: "config `<file>` path, ",
		Value: config.ConfigFile,
	}
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "block data and logs storage `<path>`",
		Value: config.DataDir,
	}
	EnableDnsFlag = cli.StringFlag{
		Name:  "dnsseed",
		Usage: "enable dns seeds for node to initialize p2p connection",
	}
	PeersFlag = cli.StringFlag{
		Name:  "peers",
		Usage: "peers seeds for node to initialize p2p connection",
	}
	InstantBlockFlag = cli.StringFlag{
		Name:  "instant",
		Usage: "specify if need to generate instant block",
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
	PayloadVersionFlag = cli.Int64Flag{
		Name:  "payloadversion",
		Usage: "payload version",
		Value: 0,
	}

	StakePoolFlag = cli.StringFlag{
		Name:  "stakepool",
		Usage: "defines the stake address of DPoS v2 votes",
	}
)

// MoveRPCFlags finds the rpc argument and moves it to the front
// of the argument array.
func MoveRPCFlags(args []string) ([]string, error) {
	newArgs := args[:1]
	cacheArgs := make([]string, 0)

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--rpcurl":
			fallthrough
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
