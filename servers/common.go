// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package servers

import (
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const TlsPort = 443

type AttributeInfo struct {
	Usage common2.AttributeUsage `json:"usage"`
	Data  string                 `json:"data"`
}

type InputInfo struct {
	TxID     string `json:"txid"`
	VOut     uint16 `json:"vout"`
	Sequence uint32 `json:"sequence"`
}

type RpcOutputInfo struct {
	Value         string            `json:"value"`
	Index         uint32            `json:"n"`
	Address       string            `json:"address"`
	AssetID       string            `json:"assetid"`
	OutputLock    uint32            `json:"outputlock"`
	OutputType    uint32            `json:"type"`
	OutputPayload OutputPayloadInfo `json:"payload"`
}

type OutputPayloadInfo interface{}

type DefaultOutputInfo struct{}

type CrossChainOutputInfo struct {
	Version       byte   `json:"Version"`
	TargetAddress string `json:"TargetAddress"`
	TargetAmount  string `json:"TargetAmount"`
	TargetData    string `json:"TargetData"`
}

type WithdrawInfo struct {
	Version                  byte   `json:"Version"`
	GenesisBlockAddress      string `json:"GenesisBlockAddress"`
	SideChainTransactionHash string `json:"SideChainTransactionHash"`
	TargetData               string `json:"TargetData"`
}

type ReturnSideChainDepositInfo struct {
	Version                byte   `json:"Version"`
	GenesisBlockAddress    string `json:"GenesisBlockAddress"`
	DepositTransactionHash string `json:"DepositTransactionHash"`
}

type ExchangeVotesOutputInfo struct {
	Version      byte   `json:"Version"`
	StakeAddress string `json:"StakeAddress"`
}

type CandidateVotes struct {
	Candidate string `json:"candidate"`
	Votes     string `json:"votes"`
}

type VoteContentInfo struct {
	VoteType       outputpayload.VoteType `json:"votetype"`
	CandidatesInfo []CandidateVotes       `json:"candidates"`
}

type VoteOutputInfo struct {
	Version  byte              `json:"version"`
	Contents []VoteContentInfo `json:"contents"`
}

type ProgramInfo struct {
	Code      string `json:"code"`
	Parameter string `json:"parameter"`
}

type TransactionInfo struct {
	TxID           string                     `json:"txid"`
	Hash           string                     `json:"hash"`
	Size           uint32                     `json:"size"`
	VSize          uint32                     `json:"vsize"`
	Version        common2.TransactionVersion `json:"version"`
	TxType         common2.TxType             `json:"type"`
	PayloadVersion byte                       `json:"payloadversion"`
	Payload        PayloadInfo                `json:"payload"`
	Attributes     []AttributeInfo            `json:"attributes"`
	Inputs         []InputInfo                `json:"vin"`
	Outputs        []RpcOutputInfo            `json:"vout"`
	LockTime       uint32                     `json:"locktime"`
	Programs       []ProgramInfo              `json:"programs"`
}

type TransactionContextInfo struct {
	*TransactionInfo
	BlockHash     string `json:"blockhash"`
	Confirmations uint32 `json:"confirmations"`
	Time          uint32 `json:"time"`
	BlockTime     uint32 `json:"blocktime"`
}

type BlockInfo struct {
	Hash              string        `json:"hash"`
	Confirmations     uint32        `json:"confirmations"`
	StrippedSize      uint32        `json:"strippedsize"`
	Size              uint32        `json:"size"`
	Weight            uint32        `json:"weight"`
	Height            uint32        `json:"height"`
	Version           uint32        `json:"version"`
	VersionHex        string        `json:"versionhex"`
	MerkleRoot        string        `json:"merkleroot"`
	Tx                []interface{} `json:"tx"`
	Time              uint32        `json:"time"`
	MedianTime        uint32        `json:"mediantime"`
	Nonce             uint32        `json:"nonce"`
	Bits              uint32        `json:"bits"`
	Difficulty        string        `json:"difficulty"`
	ChainWork         string        `json:"chainwork"`
	PreviousBlockHash string        `json:"previousblockhash"`
	NextBlockHash     string        `json:"nextblockhash"`
	AuxPow            string        `json:"auxpow"`
	MinerInfo         string        `json:"minerinfo"`
}

type VoteInfo struct {
	Signer string `json:"signer"`
	Accept bool   `json:"accept"`
}

type ConfirmInfo struct {
	BlockHash  string     `json:"blockhash"`
	Sponsor    string     `json:"sponsor"`
	ViewOffset uint32     `json:"viewoffset"`
	Votes      []VoteInfo `json:"votes"`
}

type ServerInfo struct {
	Compile   string      `json:"compile"`   // The compile version of this server node
	Height    uint32      `json:"height"`    // The ServerNode latest block height
	Version   uint32      `json:"version"`   // The network protocol the ServerNode used
	Services  string      `json:"services"`  // The services the server supports
	Port      uint16      `json:"port"`      // The nodes's port
	RPCPort   uint16      `json:"rpcport"`   // The RPC service port
	RestPort  uint16      `json:"restport"`  // The RESTful service port
	WSPort    uint16      `json:"wsport"`    // The webservcie port
	Neighbors []*PeerInfo `json:"neighbors"` // The connected neighbor peers.
}

type PeerInfo struct {
	NetAddress     string `json:"netaddress"`
	Services       string `json:"services"`
	RelayTx        bool   `json:"relaytx"`
	LastSend       string `json:"lastsend"`
	LastRecv       string `json:"lastrecv"`
	ConnTime       string `json:"conntime"`
	TimeOffset     int64  `json:"timeoffset"`
	Version        uint32 `json:"version"`
	Inbound        bool   `json:"inbound"`
	StartingHeight uint32 `json:"startingheight"`
	LastBlock      uint32 `json:"lastblock"`
	LastPingTime   string `json:"lastpingtime"`
	LastPingMicros int64  `json:"lastpingmicros"`
	NodeVersion    string `json:"nodeversion"`
}

type ArbitratorGroupInfo struct {
	OnDutyArbitratorIndex int      `json:"ondutyarbitratorindex"`
	Arbitrators           []string `json:"arbitrators"`
}

type PayloadInfo interface{}

type CoinbaseInfo struct {
	CoinbaseData string `json:"coinbasedata"`
}

type RegisterAssetInfo struct {
	Asset      payload.Asset `json:"asset"`
	Amount     string        `json:"amount"`
	Controller string        `json:"controller"`
}

type SideChainPowInfo struct {
	BlockHeight     uint32 `json:"blockheight"`
	SideBlockHash   string `json:"sideblockhash"`
	SideGenesisHash string `json:"sidegenesishash"`
	Signature       string `json:"signature"`
}

type TransferCrossChainAssetInfo struct {
	CrossChainAddresses []string         `json:"crosschainaddresses"`
	OutputIndexes       []uint64         `json:"outputindexes"`
	CrossChainAmounts   []common.Fixed64 `json:"crosschainamounts"`
}

type WithdrawFromSideChainInfo struct {
	BlockHeight                uint32   `json:"blockheight"`
	GenesisBlockAddress        string   `json:"genesisblockaddress"`
	SideChainTransactionHashes []string `json:"sidechaintransactionhashes"`
}

type SchnorrWithdrawFromSideChainInfo struct {
	Signers []uint32 `json:"Signers"`
}

type ProducerInfo struct {
	OwnerPublicKey string `json:"ownerpublickey"`
	NodePublicKey  string `json:"nodepublickey"`
	NickName       string `json:"nickname"`
	Url            string `json:"url"`
	Location       uint64 `json:"location"`
	NetAddress     string `json:"netaddress"`
	StakeUntil     uint32 `json:"stakeuntil"`
	Signature      string `json:"signature"`
}

type CancelProducerInfo struct {
	OwnerPublicKey string `json:"ownerpublickey"`
	Signature      string `json:"signature"`
}

type InactiveArbitratorsInfo struct {
	Sponsor     string   `json:"sponsor"`
	Arbitrators []string `json:"arbitrators"`
	BlockHeight uint32   `json:"blockheight"`
}

type RevertToDPOSInfo struct {
	WorkHeightInterval     uint32
	RevertToPOWBlockHeight uint32
}

type RevertToPOWInfo struct {
	Type          string
	WorkingHeight uint32
}

type ActivateProducerInfo struct {
	NodePublicKey string `json:"nodepublickey"`
	Signature     string `json:"signature"`
}

type UpdateVersionInfo struct {
	StartHeight uint32 `json:"startheight"`
	EndHeight   uint32 `json:"endheight"`
}

type CRInfo struct {
	Code      string `json:"code"`
	CID       string `json:"cid"`
	DID       string `json:"did"`
	NickName  string `json:"nickname"`
	Url       string `json:"url"`
	Location  uint64 `json:"location"`
	Signature string `json:"signature"`
}

type MultiCRInfo struct {
	CID      string `json:"cid"`
	DID      string `json:"did"`
	NickName string `json:"nickname"`
	Url      string `json:"url"`
	Location uint64 `json:"location"`
}

type UnregisterCRInfo struct {
	CID       string `json:"cid"`
	Signature string `json:"signature"`
}

type BudgetBaseInfo struct {
	Type   string `json:"type"`
	Stage  uint8  `json:"stage"`
	Amount string `json:"amount"`
}

type BudgetInfo struct {
	Type   string `json:"type"`
	Stage  uint8  `json:"stage"`
	Amount string `json:"amount"`
	Status string `json:"status"`
}

type CRCProposalInfo struct {
	ProposalType             string           `json:"proposaltype"`
	CategoryData             string           `json:"categorydata"`
	OwnerPublicKey           string           `json:"ownerpublickey"`
	DraftHash                string           `json:"drafthash"`
	Budgets                  []BudgetBaseInfo `json:"budgets"`
	Recipient                string           `json:"recipient"`
	Signature                string           `json:"signature"`
	CRCouncilMemberDID       string           `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string           `json:"crcouncilmembersignature"`
	Hash                     string           `json:"hash"`
}

type CRCChangeProposalOwnerInfo struct {
	ProposalType             string `json:"proposaltype"`
	CategoryData             string `json:"categorydata"`
	OwnerPublicKey           string `json:"ownerpublickey"`
	DraftHash                string `json:"drafthash"`
	TargetProposalHash       string `json:"targetproposalhash"`
	NewRecipient             string `json:"newrecipient"`
	NewOwnerPublicKey        string `json:"newownerpublickey"`
	Signature                string `json:"signature"`
	NewOwnerSignature        string `json:"newownersignature"`
	CRCouncilMemberDID       string `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string `json:"crcouncilmembersignature"`
	Hash                     string `json:"hash"`
}

type CRCCloseProposalInfo struct {
	ProposalType             string `json:"proposaltype"`
	CategoryData             string `json:"categorydata"`
	OwnerPublicKey           string `json:"ownerpublickey"`
	DraftHash                string `json:"drafthash"`
	TargetProposalHash       string `json:"targetproposalhash"`
	Signature                string `json:"signature"`
	CRCouncilMemberDID       string `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string `json:"crcouncilmembersignature"`
	Hash                     string `json:"hash"`
}

type CRCReservedCustomIDProposalInfo struct {
	ProposalType             string   `json:"proposaltype"`
	CategoryData             string   `json:"categorydata"`
	OwnerPublicKey           string   `json:"ownerpublickey"`
	DraftHash                string   `json:"drafthash"`
	ReservedCustomIDList     []string `json:"reservedcustomidlist"`
	Signature                string   `json:"signature"`
	CRCouncilMemberDID       string   `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string   `json:"crcouncilmembersignature"`
	Hash                     string   `json:"hash"`
}

type CRCChangeCustomIDFeeInfo struct {
	ProposalType             string `json:"proposaltype"`
	CategoryData             string `json:"categorydata"`
	OwnerPublicKey           string `json:"ownerpublickey"`
	DraftHash                string `json:"drafthash"`
	FeeRate                  int64  `json:"feerate"`
	EIDEffectiveHeight       uint32 `json:"eideffectiveheight"`
	Signature                string `json:"signature"`
	CRCouncilMemberDID       string `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string `json:"crcouncilmembersignature"`
	Hash                     string `json:"hash"`
}

type CRCReceivedCustomIDProposalInfo struct {
	ProposalType             string   `json:"proposaltype"`
	CategoryData             string   `json:"categorydata"`
	OwnerPublicKey           string   `json:"ownerpublickey"`
	DraftHash                string   `json:"drafthash"`
	ReceiveCustomIDList      []string `json:"receivecustomidlist"`
	ReceiverDID              string   `json:"receiverdid"`
	Signature                string   `json:"signature"`
	CRCouncilMemberDID       string   `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string   `json:"crcouncilmembersignature"`
	Hash                     string   `json:"hash"`
}

type CRCSecretaryGeneralProposalInfo struct {
	ProposalType              string `json:"proposaltype"`
	CategoryData              string `json:"categorydata"`
	OwnerPublicKey            string `json:"ownerpublickey"`
	DraftHash                 string `json:"drafthash"`
	SecretaryGeneralPublicKey string `json:"secretarygeneralpublickey"`
	SecretaryGeneralDID       string `json:"secretarygeneraldid"`
	Signature                 string `json:"signature"`
	SecretaryGeneraSignature  string `json:"secretarygenerasignature"`
	CRCouncilMemberDID        string `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature  string `json:"crcouncilmembersignature"`
	Hash                      string `json:"hash"`
}

type CRCRegisterSideChainProposalInfo struct {
	ProposalType             string         `json:"proposaltype"`
	CategoryData             string         `json:"categorydata"`
	OwnerPublicKey           string         `json:"ownerpublickey"`
	DraftHash                string         `json:"drafthash"`
	SideChainName            string         `json:"sidechainname"`
	MagicNumber              uint32         `json:"magicnumber"`
	GenesisHash              string         `json:"genesishash"`
	ExchangeRate             common.Fixed64 `json:"exchangerate"`
	EffectiveHeight          uint32         `json:"effectiveheight"`
	ResourcePath             string         `json:"resourcepath"`
	Signature                string         `json:"signature"`
	CRCouncilMemberDID       string         `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string         `json:"crcouncilmembersignature"`
	Hash                     string         `json:"hash"`
}

type CRCProposalReviewInfo struct {
	ProposalHash string `json:"proposalhash"`
	VoteResult   string `json:"voteresult"`
	OpinionHash  string `json:"opinionhash"`
	DID          string `json:"did"`
	Sign         string `json:"sign"`
}

type CRCCustomIDProposalResultInfo struct {
	ProposalResults []ProposalResultInfo `json:"proposalresults"`
}

type ProposalResultInfo struct {
	ProposalHash string `json:"proposalhash"`
	ProposalType string `json:"proposaltype"`
	Result       bool   `json:"result"`
}

type CRCProposalTrackingInfo struct {
	ProposalTrackingType        string `json:"proposaltrackingtype"`
	ProposalHash                string `json:"proposalhash"`
	MessageHash                 string `json:"messagehash"`
	Stage                       uint8  `json:"stage"`
	OwnerPublicKey              string `json:"ownerpublickey"`
	NewOwnerPublicKey           string `json:"newownerpublickey"`
	OwnerSignature              string `json:"ownersignature"`
	NewOwnerSignature           string `json:"newownersignature"`
	SecretaryGeneralOpinionHash string `json:"secretarygeneralopinionhash"`
	SecretaryGeneralSignature   string `json:"secretarygeneralsignature"`
}

type CRCProposalWithdrawInfo struct {
	ProposalHash   string `json:"proposalhash"`
	OwnerPublicKey string `json:"ownerpublickey"`
	Recipient      string `json:"recipient,omitempty"`
	Amount         string `json:"amount,omitempty"`
	Signature      string `json:"signature"`
}

type CRCouncilMemberClaimNodeInfo struct {
	NodePublicKey            string `json:"nodepublickey"`
	CRCouncilMemberDID       string `json:"crcouncilmemberdid"`
	CRCouncilMemberSignature string `json:"crcouncilmembersignature"`
}

type NextTurnDPOSPayloadInfo struct {
	WorkingHeight  uint32   `json:"workingheight"`
	CRPublickeys   []string `json:"crpublickeys"`
	DPOSPublicKeys []string `json:"dpospublickeys"`
}

type NextTurnDPOSPayloadInfoV2 struct {
	WorkingHeight        uint32   `json:"workingheight"`
	CRPublicKeys         []string `json:"crpublickeys"`
	DPOSPublicKeys       []string `json:"dpospublickeys"`
	CompleteCRPublicKeys []string `json:"CompleteCRPublicKeys"`
}

type CRCProposalRealWithdrawInfo struct {
	WithdrawTransactionHashes []string `json:"withdrawtransactionhashes"`
}

type DPOSProposalInfo struct {
	Sponsor    string `json:"sponsor"`
	BlockHash  string `json:"blockhash"`
	ViewOffset uint32 `json:"viewoffset"`
	Sign       string `json:"sign"`
	Hash       string `json:"hash"`
}

type BlockEvidenceInfo struct {
	Header       string   `json:"header"`
	BlockConfirm string   `json:"blockconfirm"`
	Signers      []string `json:"signers"`

	Hash string `json:"hash"`
}

type DPOSIllegalBlocksInfo struct {
	CoinType        uint32            `json:"cointype"`
	BlockHeight     uint32            `json:"blockheight"`
	Evidence        BlockEvidenceInfo `json:"evidence"`
	CompareEvidence BlockEvidenceInfo `json:"compareevidence"`

	Hash string `json:"hash"`
}

type ProposalEvidenceInfo struct {
	Proposal    DPOSProposalInfo `json:"proposal"`
	BlockHeight uint32           `json:"blockheight"`
}

type DPOSIllegalProposalsInfo struct {
	Evidence        ProposalEvidenceInfo `json:"evidence"`
	CompareEvidence ProposalEvidenceInfo `json:"compareevidence"`
	Hash            string               `json:"hash"`
}

type DPOSProposalVoteInfo struct {
	ProposalHash string `json:"proposalhash"`
	Signer       string `json:"signer"`
	Accept       bool   `json:"accept"`
	Sign         string `json:"sign"`
	Hash         string `json:"hash"`
}

type VoteEvidenceInfo struct {
	ProposalEvidenceInfo
	Vote DPOSProposalVoteInfo `json:"vote"`
}

type DPOSIllegalVotesInfo struct {
	Evidence        VoteEvidenceInfo `json:"evidence"`
	CompareEvidence VoteEvidenceInfo `json:"compareevidence"`
	Hash            string           `json:"hash"`
}

type UTXOInfo struct {
	TxType        byte   `json:"txtype"`
	TxID          string `json:"txid"`
	AssetID       string `json:"assetid"`
	VOut          uint16 `json:"vout"`
	Address       string `json:"address"`
	Amount        string `json:"amount"`
	OutputLock    uint32 `json:"outputlock"`
	Confirmations uint32 `json:"confirmations"`
}

type SidechainIllegalDataInfo struct {
	IllegalType         uint8    `json:"illegaltype"`
	Height              uint32   `json:"height"`
	IllegalSigner       string   `json:"illegalsigner"`
	Evidence            string   `json:"evidence"`
	CompareEvidence     string   `json:"compareevidence"`
	GenesisBlockAddress string   `json:"genesisblockaddress"`
	Signs               []string `json:"signs"`
}

type RsInfo struct {
	SideChainName   string         `json:"sidechainname"`
	MagicNumber     uint32         `json:"magicnumber"`
	GenesisHash     string         `json:"genesishash"`
	ExchangeRate    common.Fixed64 `json:"exchangerate"`
	ResourcePath    string         `json:"resourcepath"`
	TxHash          string         `json:"txhash"`
	Height          uint32         `json:"height"`
	EffectiveHeight uint32         `json:"effectiveheight"`
}

type VotingInfo struct {
	Contents        []VotesContentInfo        `json:"contents"`
	RenewalContents []RenewalVotesContentInfo `json:"renewalcontents"`
}

type VotesContentInfo struct {
	VoteType  byte                    `json:"votetype"`
	VotesInfo []VotesWithLockTimeInfo `json:"votesinfo"`
}

type VotesWithLockTimeInfo struct {
	Candidate string `json:"candidate"`
	Votes     string `json:"votes"`
	LockTime  uint32 `json:"locktime"`
}

type RenewalVotesContentInfo struct {
	ReferKey  string                `json:"referkey"`
	VotesInfo VotesWithLockTimeInfo `json:"votesinfo"`
}

type ExchangeVotesInfo struct {
}

type ReturnVotesInfo struct {
	// target or to address
	ToAddr string `json:"toaddr"`
	// code
	Code string `json:"code,omitempty"`
	// return votes value
	Value string `json:"value"`
	// signature
	Signature string `json:"signature,omitempty"`
}

type RealReturnVotesInfo struct {
	ReturnVotesTXHash string `json:"returnvotestxhash"`
	StakeAddress      string `json:"stakeaddress"`
	Value             string `json:"value"`
}
type RealVotesWithdrawInfo struct {
	RealReturnVotes []RealReturnVotesInfo `json:"realReturnVotes"`
}

type DposV2ClaimRewardInfo struct {
	// target or to address
	ToAddr string `json:"toaddr"`
	// code
	Code string `json:"code,omitempty"`
	// reward value
	Value string `json:"value"`
	// signature
	Signature string `json:"signature,omitempty"`
}

type DposV2ClaimRewardRealWithdrawInfo struct {
	// Hash of the proposal to withdrawal ela.
	WithdrawTransactionHashes []string `json:"withdrawtransactionhashes"`
}

type CreateNFTInfo struct {
	// NFT ID
	ID string
	// hash of detailed vote information.
	ReferKey string
	// side chain format address.
	StakeAddress string
	// side chain genesis block address
	GenesisBlockHash string
}

type CreateNFTInfoV2 struct {
	// NFT ID
	ID string
	// hash of detailed vote information.
	ReferKey string
	// side chain format address.
	StakeAddress string
	// side chain genesis block address
	GenesisBlockHash string
	// the start height of votes
	StartHeight uint32
	// the end height of votes: start height + lock time.
	EndHeight uint32
	// the DPoS 2.0 votes.
	Votes string
	// the DPoS 2.0 vote rights.
	VoteRights string
	// the votes to the producer, and TargetOwnerPublicKey is the producer's
	// owner key.
	TargetOwnerKey string
}

type DestroyNFTInfo struct {
	// detail votes info referkey
	IDs []string
	// owner OwnerStakeAddress
	OwnerStakeAddresses []string
	// genesis block hash of side chain
	GenesisBlockHash string
}

type DetailedVoteInfo struct {
	StakeAddress    string
	TransactionHash string
	BlockHeight     uint32
	PayloadVersion  byte
	VoteType        uint32
	Info            []VotesWithLockTimeInfo
}
