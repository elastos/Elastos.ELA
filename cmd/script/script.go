// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package script

import (
	"fmt"
	"os"
	"strings"

	"github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/cmd/script/api"

	"github.com/urfave/cli"
	lua "github.com/yuin/gopher-lua"
)

func registerParams(c *cli.Context, L *lua.LState) {
	wallet := c.String("wallet")
	addresses := c.String("addresses")
	password := c.String("password")
	code := c.String("code")
	publicKey := c.String("publickey")
	privateKeys := c.String("privatekeys")
	publicKeys := c.String("publickeys")
	depositAddr := c.String("depositaddr")
	nickname := c.String("nickname")
	url := c.String("url")
	location := c.Int64("location")
	depositAmount := c.Float64("depositamount")
	stakeUntil := c.Int64("stakeuntil")
	amount := c.Float64("amount")
	fee := c.Float64("fee")
	votes := c.Float64("votes")
	toAddr := c.String("to")
	amounts := c.String("amounts")
	ownPublicKey := c.String("ownerpublickey")
	ownPrivateKey := c.String("ownerprivatekey")
	nodePubkey := c.String("nodepublickey")
	host := c.String("host")
	candidates := c.String("candidates")
	candidateVotes := c.String("candidateVotes")
	draftHash := c.String("drafthash")

	// CRCProposal Related Params
	proposalType := c.Int64("proposaltype")
	proposalHash := c.String("proposalhash")
	draftData := c.String("draftdata")
	budgets := c.String("budgets")
	voteResult := c.Int("voteresult")
	proposalTrackingType := c.Int64("proposaltrackingtype")
	MessageHash := c.String("messagehash")
	crOpinionHash := c.String("cropinionhash")
	crOpinionData := c.String("cropiniondata")
	SecretaryGeneralOpinionHash := c.String("secretarygeneralopinionhash")
	stage := c.Int64("stage")

	newOwnerPublicKey := c.String("newownerpublickey")
	newOwnerPrivateKey := c.String("newownerprivatekey")
	secretaryGeneralPrivkey := c.String("secretarygeneralprivatekey")
	secretaryGeneralPublickey := c.String("secretarygeneralpublickey")
	recipient := c.String("recipient")
	targetHash := c.String("targethash")
	closeProposalHash := c.String("closeproposalhash")
	reservedCustomIDList := c.String("reservedcustomidlist")
	receivedCustomIDList := c.String("receivedcustomidlist")
	customidrate := c.String("customidrate")
	receiverDID := c.String("receiverdid")
	CRExpensesAddress := c.String("crccommiteeaddr")
	payloadVersion := c.Int64("payloadversion")

	crManagementPublicKey := c.String("crmanagementpublickey")
	crDPOSPrivateKey := c.String("crdposprivatekey")
	crCommitteeDID := c.String("crcommitteedid")

	targetData := c.String("targetdata")

	// Register SideChain
	sideChainName := c.String("sidechainname")
	magicNumber := c.Uint("magicnumber")
	genesisHash := c.String("genesishash")
	exchangeRate := c.String("exchangerate")
	effectiveHeight := c.Uint("effectiveheight")
	resourcePath := c.String("resourcepath")

	// dposv2
	referKey := c.String("referkey")
	voteType := c.Uint("votetype")

	// nft
	nftID := c.String("nftid")
	stakeAddress := c.String("stakeaddr")

	getWallet := func(L *lua.LState) int {
		L.Push(lua.LString(wallet))
		return 1
	}
	//get_addresses
	getAddresses := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("addresses"))
		cs := strings.Split(addresses, ",")
		for _, c := range cs {
			table.Append(lua.LString(c))
		}
		L.Push(table)
		return 1
	}
	getPassword := func(L *lua.LState) int {
		L.Push(lua.LString(password))
		return 1
	}
	getDepositAddr := func(L *lua.LState) int {
		L.Push(lua.LString(depositAddr))
		return 1
	}
	getPublicKey := func(L *lua.LState) int {
		L.Push(lua.LString(publicKey))
		return 1
	}
	getPrivateKeys := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("privatekeys"))
		cs := strings.Split(privateKeys, ",")
		for _, c := range cs {
			table.Append(lua.LString(c))
		}
		L.Push(table)
		return 1
	}
	getPublicKeys := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("publickeys"))
		cs := strings.Split(publicKeys, ",")
		for _, c := range cs {
			table.Append(lua.LString(c))
		}
		L.Push(table)
		return 1
	}
	getCode := func(L *lua.LState) int {
		L.Push(lua.LString(code))
		return 1
	}
	getNickName := func(L *lua.LState) int {
		L.Push(lua.LString(nickname))
		return 1
	}
	getUrl := func(L *lua.LState) int {
		L.Push(lua.LString(url))
		return 1
	}
	getLocation := func(L *lua.LState) int {
		L.Push(lua.LNumber(location))
		return 1
	}
	getDepositAmount := func(L *lua.LState) int {
		L.Push(lua.LNumber(depositAmount))
		return 1
	}
	getStakeUntil := func(L *lua.LState) int {
		L.Push(lua.LNumber(stakeUntil))
		return 1
	}
	getAmount := func(L *lua.LState) int {
		L.Push(lua.LNumber(amount))
		return 1
	}
	getFee := func(L *lua.LState) int {
		L.Push(lua.LNumber(fee))
		return 1
	}
	getVotes := func(L *lua.LState) int {
		L.Push(lua.LNumber(votes))
		return 1
	}
	getToAddr := func(L *lua.LState) int {
		L.Push(lua.LString(toAddr))
		return 1
	}
	getAmounts := func(L *lua.LState) int {
		L.Push(lua.LString(amounts))
		return 1
	}
	getOwnerPublicKey := func(L *lua.LState) int {
		L.Push(lua.LString(ownPublicKey))
		return 1
	}
	getOwnerPrivateKey := func(L *lua.LState) int {
		L.Push(lua.LString(ownPrivateKey))
		return 1
	}
	getNodePublicKey := func(L *lua.LState) int {
		L.Push(lua.LString(nodePubkey))
		return 1
	}
	getHostAddr := func(L *lua.LState) int {
		L.Push(lua.LString(host))
		return 1
	}
	getCandidates := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("candidates"))
		cs := strings.Split(candidates, ",")
		for _, c := range cs {
			table.Append(lua.LString(c))
		}
		L.Push(table)
		return 1
	}
	getCandidateVotes := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("candidateVotes"))
		votes := strings.Split(candidateVotes, ",")
		for _, cv := range votes {
			table.Append(lua.LString(cv))
		}
		L.Push(table)
		return 1
	}
	getDraftHash := func(L *lua.LState) int {
		L.Push(lua.LString(draftHash))
		return 1
	}

	getProposalType := func(L *lua.LState) int {
		L.Push(lua.LNumber(proposalType))
		return 1
	}
	getProposalHash := func(L *lua.LState) int {
		L.Push(lua.LString(proposalHash))
		return 1
	}
	getDraftData := func(L *lua.LState) int {
		L.Push(lua.LString(draftData))
		return 1
	}
	getBudgets := func(L *lua.LState) int {
		table := L.NewTable()
		L.SetMetatable(table, L.GetTypeMetatable("budgets"))
		bs := strings.Split(budgets, ",")
		for _, budget := range bs {
			table.Append(lua.LString(budget))
		}
		L.Push(table)
		return 1
	}
	getVoteResult := func(L *lua.LState) int {
		L.Push(lua.LNumber(voteResult))
		return 1
	}
	getProposalTrackingType := func(L *lua.LState) int {
		L.Push(lua.LNumber(proposalTrackingType))
		return 1
	}
	getMessageHash := func(L *lua.LState) int {
		L.Push(lua.LString(MessageHash))
		return 1
	}
	getCROpinionHash := func(L *lua.LState) int {
		L.Push(lua.LString(crOpinionHash))
		return 1
	}
	getCROpinionData := func(L *lua.LState) int {
		L.Push(lua.LString(crOpinionData))
		return 1
	}

	getSecretaryGeneralOpinionHash := func(L *lua.LState) int {
		L.Push(lua.LString(SecretaryGeneralOpinionHash))
		return 1
	}
	getStage := func(L *lua.LState) int {
		L.Push(lua.LNumber(stage))
		return 1
	}
	getLeaderPrivkey := func(L *lua.LState) int {
		L.Push(lua.LString(ownPrivateKey))
		return 1
	}
	getNewLeaderPrivkey := func(L *lua.LState) int {
		L.Push(lua.LString(newOwnerPrivateKey))
		return 1
	}
	getSecretaryGeneralPrivkey := func(L *lua.LState) int {
		L.Push(lua.LString(secretaryGeneralPrivkey))
		return 1
	}
	getSecretaryGeneralPublickey := func(L *lua.LState) int {
		L.Push(lua.LString(secretaryGeneralPublickey))
		return 1
	}
	getRecipient := func(L *lua.LState) int {
		L.Push(lua.LString(recipient))
		return 1
	}
	getNewOwnerPublicKey := func(L *lua.LState) int {
		L.Push(lua.LString(newOwnerPublicKey))
		return 1
	}
	getNewOwnerPrivateKey := func(L *lua.LState) int {
		L.Push(lua.LString(newOwnerPrivateKey))
		return 1
	}
	getTargetHash := func(L *lua.LState) int {
		L.Push(lua.LString(targetHash))
		return 1
	}
	getCloseProposalHash := func(L *lua.LState) int {
		L.Push(lua.LString(closeProposalHash))
		return 1
	}
	getReservedCustomIDList := func(L *lua.LState) int {
		L.Push(lua.LString(reservedCustomIDList))
		return 1
	}
	getReceivedCustomIDList := func(L *lua.LState) int {
		L.Push(lua.LString(receivedCustomIDList))
		return 1
	}
	getRateOfCustomIDFee := func(L *lua.LState) int {
		L.Push(lua.LString(customidrate))
		return 1
	}
	getReceiverDID := func(L *lua.LState) int {
		L.Push(lua.LString(receiverDID))
		return 1
	}
	getCRExpensesAddress := func(L *lua.LState) int {
		L.Push(lua.LString(CRExpensesAddress))
		return 1
	}
	getPayloadVersion := func(L *lua.LState) int {
		L.Push(lua.LNumber(payloadVersion))
		return 1
	}
	getCRManagementPublicKey := func(L *lua.LState) int {
		L.Push(lua.LString(crManagementPublicKey))
		return 1
	}
	getCRDPOSPrivateKey := func(L *lua.LState) int {
		L.Push(lua.LString(crDPOSPrivateKey))
		return 1
	}
	getCRCommitteeDID := func(L *lua.LState) int {
		L.Push(lua.LString(crCommitteeDID))
		return 1
	}
	getTargetData := func(L *lua.LState) int {
		L.Push(lua.LString(targetData))
		return 1
	}

	// Register SideChain
	getSideChainName := func(L *lua.LState) int {
		L.Push(lua.LString(sideChainName))
		return 1
	}

	getMagicNumber := func(L *lua.LState) int {
		L.Push(lua.LNumber(magicNumber))
		return 1
	}

	getGenesisHash := func(L *lua.LState) int {
		L.Push(lua.LString(genesisHash))
		return 1
	}

	getExchangeRate := func(L *lua.LState) int {
		L.Push(lua.LString(exchangeRate))
		return 1
	}

	getEffectiveHeight := func(L *lua.LState) int {
		L.Push(lua.LString(string(rune(effectiveHeight))))
		return 1
	}

	getResourcePath := func(L *lua.LState) int {
		L.Push(lua.LString(resourcePath))
		return 1
	}

	getReferKey := func(L *lua.LState) int {
		L.Push(lua.LString(referKey))
		return 1
	}

	getVoteType := func(L *lua.LState) int {
		L.Push(lua.LNumber(voteType))
		return 1
	}

	getNFTID := func(L *lua.LState) int {
		L.Push(lua.LString(nftID))
		return 1
	}

	getStakeAddress := func(L *lua.LState) int {
		L.Push(lua.LString(stakeAddress))
		return 1
	}
	L.Register("getAddresses", getAddresses)
	L.Register("getWallet", getWallet)
	L.Register("getPassword", getPassword)
	L.Register("getDepositAddr", getDepositAddr)
	L.Register("getPublicKey", getPublicKey)
	L.Register("getPrivateKeys", getPrivateKeys)
	L.Register("getPublicKeys", getPublicKeys)
	L.Register("getCode", getCode)
	L.Register("getNickName", getNickName)
	L.Register("getUrl", getUrl)
	L.Register("getLocation", getLocation)
	L.Register("getDepositAmount", getDepositAmount)
	L.Register("getStakeUntil", getStakeUntil)
	L.Register("getAmount", getAmount)
	L.Register("getFee", getFee)
	L.Register("getVotes", getVotes)
	L.Register("getToAddr", getToAddr)
	L.Register("getAmounts", getAmounts)
	L.Register("getNodePublicKey", getNodePublicKey)
	L.Register("getHostAddr", getHostAddr)
	L.Register("getCandidates", getCandidates)
	L.Register("getCandidateVotes", getCandidateVotes)
	L.Register("getDraftHash", getDraftHash)
	L.Register("getOwnerPublicKey", getOwnerPublicKey)
	L.Register("getOwnerPrivateKey", getOwnerPrivateKey)

	L.Register("getProposalType", getProposalType)
	L.Register("getDraftData", getDraftData)
	L.Register("getBudgets", getBudgets)
	L.Register("getProposalHash", getProposalHash)
	L.Register("getVoteResult", getVoteResult)
	L.Register("getProposalTrackingType", getProposalTrackingType)
	L.Register("getMessageHash", getMessageHash)
	L.Register("getCROpinionHash", getCROpinionHash)
	L.Register("getCROpinionData", getCROpinionData)

	L.Register("getSecretaryGeneralOpinionHash", getSecretaryGeneralOpinionHash)
	L.Register("getStage", getStage)

	L.Register("getLeaderPrivkey", getLeaderPrivkey)
	L.Register("getNewLeaderPrivkey", getNewLeaderPrivkey)
	L.Register("getSecretaryGeneralPrivkey", getSecretaryGeneralPrivkey)
	L.Register("getSecretaryGeneralPublickey", getSecretaryGeneralPublickey)
	L.Register("getRecipient", getRecipient)
	L.Register("getNewOwnerPublicKey", getNewOwnerPublicKey)
	L.Register("getNewOwnerPrivateKey", getNewOwnerPrivateKey)
	L.Register("getTargetHash", getTargetHash)
	L.Register("getCloseProposalHash", getCloseProposalHash)
	L.Register("getReservedCustomIDList", getReservedCustomIDList)
	L.Register("getReceivedCustomIDList", getReceivedCustomIDList)
	L.Register("getRateOfCustomIDFee", getRateOfCustomIDFee)
	L.Register("getReceiverDID", getReceiverDID)
	L.Register("getCRExpensesAddress", getCRExpensesAddress)
	L.Register("getPayloadVersion", getPayloadVersion)
	L.Register("getCRManagementPublicKey", getCRManagementPublicKey)
	L.Register("getCRDPOSPrivateKey", getCRDPOSPrivateKey)
	L.Register("getCRCommitteeDID", getCRCommitteeDID)
	L.Register("getTargetData", getTargetData)

	//Register SideChain
	L.Register("getSideChainName", getSideChainName)
	L.Register("getMagicNumber", getMagicNumber)
	L.Register("getGenesisHash", getGenesisHash)
	L.Register("getExchangeRate", getExchangeRate)
	L.Register("getEffectiveHeight", getEffectiveHeight)
	L.Register("getResourcePath", getResourcePath)

	L.Register("getReferKey", getReferKey)
	L.Register("getVoteType", getVoteType)
	L.Register("getNFTID", getNFTID)
	L.Register("getStakeAddr", getStakeAddress)
}

func scriptAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	fileContent := c.String("file")
	strContent := c.String("str")
	testContent := c.String("test")

	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("api", api.Loader)
	api.RegisterDataType(L)

	if strContent != "" {
		if err := L.DoString(strContent); err != nil {
			panic(err)
		}
	}

	if fileContent != "" {
		registerParams(c, L)
		if err := L.DoFile(fileContent); err != nil {
			panic(err)
		}
	}

	if testContent != "" {
		fmt.Println("begin white box")
		if err := L.DoFile(testContent); err != nil {
			println(err.Error())
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "script",
		Usage:       "Test the blockchain via lua script",
		Description: "With ela-cli test, you could test blockchain.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			common.AccountWalletFlag,
			common.AccountPasswordFlag,
			cli.StringFlag{
				Name:  "file, f",
				Usage: "test file",
			},
			cli.StringFlag{
				Name:  "str, s",
				Usage: "test string",
			},
			cli.StringFlag{
				Name:  "test, t",
				Usage: "white box test",
			},
			cli.StringFlag{
				Name:  "publickey, pk",
				Usage: "set the public key",
			},
			cli.StringFlag{
				Name:  "privatekeys, priks",
				Usage: "set the private key",
			},
			cli.StringFlag{
				Name:  "addresses",
				Usage: "set the addresses",
			},
			cli.StringFlag{
				Name:  "publickeys, pubs",
				Usage: "set the pub keys",
			},
			cli.StringFlag{
				Name:  "depositaddr, daddr",
				Usage: "set the deposit addr",
			},
			cli.Float64Flag{
				Name:  "depositamount, damount",
				Usage: "set the amount",
			},
			cli.StringFlag{
				Name:  "nickname, nn",
				Usage: "set the nick name",
			},
			cli.StringFlag{
				Name:  "url, u",
				Usage: "set the url",
			},
			cli.Int64Flag{
				Name:  "location, l",
				Usage: "set the location",
			},
			cli.Float64Flag{
				Name:  "amount",
				Usage: "set the amount",
			},
			cli.Int64Flag{
				Name:  "stakeuntil",
				Usage: "set the stakeuntil height",
			},
			cli.Float64Flag{
				Name:  "fee",
				Usage: "set the fee",
			},
			cli.StringFlag{
				Name:  "code, c",
				Usage: "set the code",
			},
			cli.Float64Flag{
				Name:  "votes, v",
				Usage: "set the votes",
			},
			cli.StringFlag{
				Name:  "to",
				Usage: "set the output address",
			},
			cli.StringFlag{
				Name:  "amounts",
				Usage: "set a list of amounts",
			},
			cli.StringFlag{
				Name:  "ownerpublickey, opk",
				Usage: "set the node public key",
			},
			cli.StringFlag{
				Name:  "ownerprivatekey, oprk",
				Usage: "set the node owner private key",
			},
			cli.StringFlag{
				Name:  "nodepublickey, npk",
				Usage: "set the owner public key",
			},
			cli.StringFlag{
				Name:  "host",
				Usage: "set the host address",
			},
			cli.StringFlag{
				Name:  "candidates, cds",
				Usage: "set the candidates public key",
			},
			cli.StringFlag{
				Name:  "candidateVotes, cvs",
				Usage: "set the candidateVotes values",
			},
			cli.Int64Flag{
				Name:  "proposaltype",
				Usage: "set the proposal type",
			},
			cli.StringFlag{
				Name:  "drafthash",
				Usage: "set the draft proposal hash",
			},
			cli.StringFlag{
				Name:  "draftdata",
				Usage: "set the draft data",
			},
			cli.StringFlag{
				Name:  "targethash",
				Usage: "set the target proposal hash",
			},
			cli.StringFlag{
				Name:  "recipient",
				Usage: "set the recipient address",
			},
			cli.StringFlag{
				Name:  "closeproposalhash",
				Usage: "set the close proposal hash",
			},
			cli.StringFlag{
				Name:  "reservedcustomidlist",
				Usage: "reserved custom id list",
			},
			cli.StringFlag{
				Name:  "receivedcustomidlist",
				Usage: "received custom id list",
			},
			cli.StringFlag{
				Name:  "receiverdid",
				Usage: "receiver did",
			},
			cli.StringFlag{
				Name:  "voteresult, votres",
				Usage: "set the owner public key",
			},
			cli.StringFlag{
				Name:  "budgets",
				Usage: "set the budgets",
			},
			cli.StringFlag{
				Name:  "proposalhash, prophash",
				Usage: "set the owner public key",
			},
			cli.StringFlag{
				Name:  "votecontenttype, votconttype",
				Usage: "set the owner public key",
			},
			cli.Int64Flag{
				Name:  "proposaltrackingtype",
				Usage: "set the type of proposal tracking transaction",
			},
			cli.StringFlag{
				Name:  "messagehash",
				Usage: "set the hash of proposal tracking document",
			},
			cli.StringFlag{
				Name:  "cropinionhash",
				Usage: "set the hash of proposal opinion",
			},
			cli.StringFlag{
				Name:  "cropiniondata",
				Usage: "set the data of cr opinion",
			},
			cli.StringFlag{
				Name:  "secretarygeneralopinionhash",
				Usage: "set the hash of proposal tracking opinion",
			},
			cli.Int64Flag{
				Name:  "stage",
				Usage: "set the stage of proposal",
			},
			cli.StringFlag{
				Name:  "newownerpublickey",
				Usage: "set the public key of new proposal leader",
			},
			cli.StringFlag{
				Name:  "newownerprivatekey",
				Usage: "set the private key of new proposal leader",
			},
			cli.StringFlag{
				Name:  "secretarygeneralprivatekey",
				Usage: "set the private key of secretary general",
			},
			cli.StringFlag{
				Name:  "secretarygeneralpublickey",
				Usage: "set the public key of secretary general",
			},
			cli.StringFlag{
				Name:  "crccommiteeaddr",
				Usage: "set the crccommiteeaddress",
			},
			cli.Int64Flag{
				Name:  "payloadversion",
				Usage: "set the version of payload",
			},
			cli.StringFlag{
				Name:  "crmanagementpublickey",
				Usage: "set the public key of crmanagement",
			},
			cli.StringFlag{
				Name:  "crdposprivatekey",
				Usage: "set the private key of crmanagement",
			},
			cli.StringFlag{
				Name:  "crcommitteedid",
				Usage: "set the crcommittee did",
			},
			cli.StringFlag{
				Name:  "customidrate",
				Usage: "set the rate of custom id",
			},
			cli.StringFlag{
				Name:  "targetdata",
				Usage: "set the target data of proposal",
			},
			// Register SideChain
			cli.StringFlag{
				Name:  "sidechainname",
				Usage: "set the sidechain name ",
			},
			cli.Int64Flag{
				Name:  "magicnumber",
				Usage: "set magic number ",
			},
			cli.StringFlag{
				Name:  "dnsseeds",
				Usage: "set dns seeds ",
			},
			cli.Int64Flag{
				Name:  "nodeport",
				Usage: "set node port ",
			},
			cli.StringFlag{
				Name:  "genesishash",
				Usage: "set genesis hash ",
			},
			cli.Int64Flag{
				Name:  "genesistimestamp",
				Usage: "set genesis timestamp ",
			},
			cli.StringFlag{
				Name:  "genesisblockdifficulty",
				Usage: "set genesis block difficulty ",
			},
			cli.StringFlag{
				Name:  "referkey",
				Usage: "set refer key of related votes",
			},
			cli.Int64Flag{
				Name:  "votetype",
				Usage: "set vote type of related votes",
			},
			cli.StringFlag{
				Name:  "nftid",
				Usage: "set id of NFT, id is the hash of detailed vote information",
			},
			cli.StringFlag{
				Name:  "stakeaddr",
				Usage: "set stake address of NFT",
			},
		},
		Action: scriptAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			common.PrintError(c, err, "script")
			return cli.NewExitError("", 1)
		},
	}
}
