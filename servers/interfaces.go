// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package servers

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elastos/Elastos.ELA/account"
	aux "github.com/elastos/Elastos.ELA/auxpow"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	. "github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/pow"
	. "github.com/elastos/Elastos.ELA/servers/errors"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/wallet"

	"github.com/tidwall/gjson"
)

var (
	Compile     string
	ChainParams *config.Configuration
	Chain       *blockchain.BlockChain
	Store       blockchain.IChainStore
	TxMemPool   *mempool.TxPool
	Pow         *pow.Service
	Server      elanet.Server
	Arbiter     *dpos.Arbitrator
	Arbiters    state.Arbitrators
	Wallet      *wallet.Wallet
	emptyHash   = common.Uint168{}
)

func GetTransactionInfo(tx interfaces.Transaction) *TransactionInfo {
	inputs := make([]InputInfo, len(tx.Inputs()))
	for i, v := range tx.Inputs() {
		inputs[i].TxID = common.ToReversedString(v.Previous.TxID)
		inputs[i].VOut = v.Previous.Index
		inputs[i].Sequence = v.Sequence
	}

	outputs := make([]RpcOutputInfo, len(tx.Outputs()))
	for i, v := range tx.Outputs() {
		outputs[i].Value = v.Value.String()
		outputs[i].Index = uint32(i)
		address, _ := v.ProgramHash.ToAddress()
		outputs[i].Address = address
		outputs[i].AssetID = common.ToReversedString(v.AssetID)
		outputs[i].OutputLock = v.OutputLock
		outputs[i].OutputType = uint32(v.Type)
		outputs[i].OutputPayload = getOutputPayloadInfo(v.Payload)
	}

	attributes := make([]AttributeInfo, len(tx.Attributes()))
	for i, v := range tx.Attributes() {
		attributes[i].Usage = v.Usage
		attributes[i].Data = common.BytesToHexString(v.Data)
	}

	programs := make([]ProgramInfo, len(tx.Programs()))
	for i, v := range tx.Programs() {
		programs[i].Code = common.BytesToHexString(v.Code)
		programs[i].Parameter = common.BytesToHexString(v.Parameter)
	}

	var txHash = tx.Hash()
	var txHashStr = common.ToReversedString(txHash)
	var size = uint32(tx.GetSize())
	return &TransactionInfo{
		TxID:           txHashStr,
		Hash:           txHashStr,
		Size:           size,
		VSize:          size,
		Version:        tx.Version(),
		TxType:         tx.TxType(),
		PayloadVersion: tx.PayloadVersion(),
		Payload:        getPayloadInfo(tx, tx.PayloadVersion()),
		Attributes:     attributes,
		Inputs:         inputs,
		Outputs:        outputs,
		LockTime:       tx.LockTime(),
		Programs:       programs,
	}
}

func GetTransactionContextInfo(header *common2.Header, tx interfaces.Transaction) *TransactionContextInfo {
	var blockHash string
	var confirmations uint32
	var time uint32
	var blockTime uint32
	if header != nil {
		confirmations = Store.GetHeight() - header.Height + 1
		blockHash = common.ToReversedString(header.Hash())
		time = header.Timestamp
		blockTime = header.Timestamp
	}

	txInfo := GetTransactionInfo(tx)

	return &TransactionContextInfo{
		TransactionInfo: txInfo,
		BlockHash:       blockHash,
		Confirmations:   confirmations,
		Time:            time,
		BlockTime:       blockTime,
	}
}

// common2.Input JSON string examples for getblock method as following:
func GetRawTransaction(param Params) map[string]interface{} {
	str, ok := param.String("txid")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	hex, err := common.FromReversedString(str)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	var hash common.Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		return ResponsePack(InvalidTransaction, "")
	}

	var header *common2.Header
	tx, height, err := Store.GetTransaction(hash)
	if err != nil {
		//try to find transaction in transaction pool.
		tx = TxMemPool.GetTransaction(hash)
		if tx == nil {
			return ResponsePack(UnknownTransaction,
				"cannot find transaction in blockchain and transactionpool")
		}
	} else {
		hash, err := Chain.GetBlockHash(height)
		if err != nil {
			return ResponsePack(UnknownTransaction, "")
		}
		header, err = Chain.GetHeader(hash)
		if err != nil {
			return ResponsePack(UnknownTransaction, "")
		}
	}

	verbose, _ := param.Bool("verbose")
	if verbose {
		return ResponsePack(Success, GetTransactionContextInfo(header, tx))
	} else {
		buf := new(bytes.Buffer)
		tx.Serialize(buf)
		return ResponsePack(Success, common.BytesToHexString(buf.Bytes()))
	}
}

func GetNeighbors(param Params) map[string]interface{} {
	peers := Server.ConnectedPeers()
	neighborAddrs := make([]string, 0, len(peers))
	for _, peer := range peers {
		neighborAddrs = append(neighborAddrs, peer.ToPeer().String())
	}
	return ResponsePack(Success, neighborAddrs)
}

func GetNodeState(param Params) map[string]interface{} {
	peers := Server.ConnectedPeers()
	states := make([]*PeerInfo, 0, len(peers))
	for _, peer := range peers {
		snap := peer.ToPeer().StatsSnapshot()
		states = append(states, &PeerInfo{
			NetAddress:     snap.Addr,
			Services:       pact.ServiceFlag(snap.Services).String(),
			RelayTx:        snap.RelayTx != 0,
			LastSend:       snap.LastSend.String(),
			LastRecv:       snap.LastRecv.String(),
			ConnTime:       snap.ConnTime.String(),
			TimeOffset:     snap.TimeOffset,
			Version:        snap.Version,
			Inbound:        snap.Inbound,
			StartingHeight: snap.StartingHeight,
			LastBlock:      snap.LastBlock,
			LastPingTime:   snap.LastPingTime.String(),
			LastPingMicros: snap.LastPingMicros,
			NodeVersion:    snap.NodeVersion,
		})
	}
	height := Chain.GetHeight()
	ver := pact.DPOSStartVersion
	if height > uint32(ChainParams.CRConfiguration.NewP2PProtocolVersionHeight) {
		ver = pact.CRProposalVersion
	}
	return ResponsePack(Success, ServerInfo{
		Compile:   Compile,
		Height:    height,
		Version:   ver,
		Services:  Server.Services().String(),
		Port:      ChainParams.NodePort,
		RPCPort:   uint16(ChainParams.HttpJsonPort),
		RestPort:  uint16(ChainParams.HttpRestPort),
		WSPort:    uint16(ChainParams.HttpWsPort),
		Neighbors: states,
	})
}

func GetSupply(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()
	burnAmount := crCommittee.DestroyedAmount
	crAssetBalance := crCommittee.CRCFoundationBalance
	crCouncilExpenses := crCommittee.CRCCommitteeBalance

	height := Chain.GetHeight()
	circulationAmount := common.Fixed64(config.OriginIssuanceAmount) +
		crCommittee.CalculateTotalELAByHeight(height) - burnAmount
	availableSupply := circulationAmount - crAssetBalance - crCouncilExpenses

	return ResponsePack(Success, SupplyInfo{
		TotaySupply:       circulationAmount.String(),
		AvailableSupply:   availableSupply.String(),
		BurnAmount:        burnAmount.String(),
		CRAssets:          crAssetBalance.String(),
		CRCouncilExpenses: crCouncilExpenses.String(),
	})
}

func SetLogLevel(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.ConfigurationPermitted); rtn != nil {
		return rtn
	}

	level, ok := param.Int("level")
	if !ok || level < 0 {
		return ResponsePack(InvalidParams, "level must be an integer in 0-6")
	}

	log.SetPrintLevel(uint8(level))
	return ResponsePack(Success, fmt.Sprint("log level has been set to ", level))
}

func CreateAuxBlock(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.MiningPermitted); rtn != nil {
		return rtn
	}

	payToAddr, ok := param.String("paytoaddress")
	if !ok {
		return ResponsePack(InvalidParams, "parameter paytoaddress not found")
	}

	block, err := Pow.CreateAuxBlock(payToAddr)
	if err != nil {
		return ResponsePack(InternalError, "generate block failed")
	}

	type AuxBlock struct {
		ChainID           int            `json:"chainid"`
		Height            uint32         `json:"height"`
		CoinBaseValue     common.Fixed64 `json:"coinbasevalue"`
		Bits              string         `json:"bits"`
		Hash              string         `json:"hash"`
		PreviousBlockHash string         `json:"previousblockhash"`
	}

	SendToAux := AuxBlock{
		ChainID:           aux.AuxPowChainID,
		Height:            Chain.GetHeight(),
		CoinBaseValue:     block.Transactions[0].Outputs()[1].Value,
		Bits:              fmt.Sprintf("%x", block.Header.Bits),
		Hash:              block.Hash().String(),
		PreviousBlockHash: Chain.GetCurrentBlockHash().String(),
	}
	return ResponsePack(Success, &SendToAux)
}

func SubmitAuxBlock(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.MiningPermitted); rtn != nil {
		return rtn
	}

	blockHashHex, ok := param.String("blockhash")
	if !ok {
		return ResponsePack(InvalidParams, "parameter blockhash not found")
	}
	blockHash, err := common.Uint256FromHexString(blockHashHex)
	if err != nil {
		return ResponsePack(InvalidParams, "bad blockhash")
	}

	auxPow, ok := param.String("auxpow")
	if !ok {
		return ResponsePack(InvalidParams, "parameter auxpow not found")
	}
	var aux aux.AuxPow
	buf, _ := common.HexStringToBytes(auxPow)
	if err := aux.Deserialize(bytes.NewReader(buf)); err != nil {
		log.Debug("[json-rpc:SubmitAuxBlock] auxpow deserialization failed", auxPow)
		return ResponsePack(InternalError, "auxpow deserialization failed")
	}

	err = Pow.SubmitAuxBlock(blockHash, &aux)
	if err != nil {
		log.Debug(err)
		return ResponsePack(InternalError, "adding block failed")
	}

	log.Debug("AddBlock called finished and Pow.MsgBlock.MapNewBlock has been deleted completely")
	log.Info(auxPow, blockHash)
	return ResponsePack(Success, true)
}

func SubmitSidechainIllegalData(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.TransactionPermitted); rtn != nil {
		return rtn
	}

	if Arbiter == nil {
		return ResponsePack(InternalError, "arbiter disabled")
	}

	rawHex, ok := param.String("illegaldata")
	if !ok {
		return ResponsePack(InvalidParams, "parameter illegaldata not found")
	}

	var data payload.SidechainIllegalData
	buf, _ := common.HexStringToBytes(rawHex)
	if err := data.DeserializeUnsigned(bytes.NewReader(buf),
		payload.SidechainIllegalDataVersion); err != nil {
		log.Debug("[json-rpc:SubmitSidechainIllegalData] illegaldata deserialization failed", rawHex)
		return ResponsePack(InternalError, "illegaldata deserialization failed")
	}

	Arbiter.OnSidechainIllegalEvidenceReceived(&data)

	return ResponsePack(Success, true)
}

func GetSmallCrossTransferTxs(params Params) map[string]interface{} {
	type SmallCrossTransferTx struct {
		Txs []string `json:"txs"`
	}
	txs, err := Store.GetSmallCrossTransferTx()
	if err != nil {
		return ResponsePack(InternalError, "internal error fail to get small crosschain transfer txs")
	}

	result := SmallCrossTransferTx{
		Txs: txs,
	}

	return ResponsePack(Success, result)
}

func GetCrossChainPeersInfo(params Params) map[string]interface{} {
	if Arbiter == nil {
		return ResponsePack(InternalError, "arbiter disabled")
	}

	peers := Arbiter.GetCurrentArbitrators()
	type peerInfo struct {
		NodePublicKeys []string `json:"nodepublickeys"`
	}
	var result peerInfo
	result.NodePublicKeys = make([]string, 0)
	peersMap := make(map[string]struct{})
	for _, p := range peers {
		if !p.IsNormal {
			continue
		}
		pk := common.BytesToHexString(p.NodePublicKey)
		peersMap[pk] = struct{}{}
		result.NodePublicKeys = append(result.NodePublicKeys, pk)
	}

	nextPeers := Arbiter.GetNextArbitrators()
	for _, p := range nextPeers {
		if !p.IsNormal {
			continue
		}
		pk := common.BytesToHexString(p.NodePublicKey)
		if _, ok := peersMap[pk]; ok {
			continue
		}
		result.NodePublicKeys = append(result.NodePublicKeys, pk)
	}

	sort.Slice(result.NodePublicKeys, func(i, j int) bool {
		return result.NodePublicKeys[i] < result.NodePublicKeys[j]
	})

	return ResponsePack(Success, result)
}

func GetCRCPeersInfo(params Params) map[string]interface{} {
	if Arbiter == nil {
		return ResponsePack(InternalError, "arbiter disabled")
	}

	peers := Arbiter.GetCurrentCRCs()
	type peerInfo struct {
		NodePublicKeys []string `json:"nodepublickeys"`
	}
	var result peerInfo
	result.NodePublicKeys = make([]string, 0)
	peersMap := make(map[string]struct{})
	for _, p := range peers {
		if !p.IsNormal {
			continue
		}
		pk := common.BytesToHexString(p.NodePublicKey)
		peersMap[pk] = struct{}{}
		result.NodePublicKeys = append(result.NodePublicKeys, pk)
	}

	nextPeers := Arbiter.GetNextCRCs()
	for _, p := range nextPeers {
		pk := common.BytesToHexString(p)
		if _, ok := peersMap[pk]; ok {
			continue
		}
		result.NodePublicKeys = append(result.NodePublicKeys, pk)
	}

	sort.Slice(result.NodePublicKeys, func(i, j int) bool {
		return result.NodePublicKeys[i] < result.NodePublicKeys[j]
	})

	return ResponsePack(Success, result)
}

func GetArbiterPeersInfo(params Params) map[string]interface{} {
	if Arbiter == nil {
		return ResponsePack(InternalError, "arbiter disabled")
	}

	type peerInfo struct {
		OwnerPublicKey string `json:"ownerpublickey"`
		NodePublicKey  string `json:"nodepublickey"`
		IP             string `json:"ip,omitempty"`
		ConnState      string `json:"connstate"`
		NodeVersion    string `json:"nodeversion"`
	}

	peers := Arbiter.GetArbiterPeersInfo()
	ip := config.Parameters.ShowPeersIp
	result := make([]peerInfo, 0)
	for _, p := range peers {
		producer := Arbiters.GetConnectedProducer(p.PID[:])
		if producer == nil {
			continue
		}
		if !ip {
			p.Addr = ""
		}
		result = append(result, peerInfo{
			OwnerPublicKey: common.BytesToHexString(
				producer.GetOwnerPublicKey()),
			NodePublicKey: common.BytesToHexString(
				producer.GetNodePublicKey()),
			IP:          p.Addr,
			ConnState:   p.State.String(),
			NodeVersion: p.NodeVersion,
		})
	}
	return ResponsePack(Success, result)
}

// if have params stakeAddress  get stakeAddress all dposv2 votes
// else get all dposv2 votes
func GetAllDetailedDPoSV2Votes(params Params) map[string]interface{} {
	start, _ := params.Int("start")
	if start < 0 {
		start = 0
	}
	limit, ok := params.Int("limit")
	if !ok {
		limit = -1
	}

	stakeAddress, _ := params.String("stakeaddress")
	type detailedVoteInfo struct {
		ProducerOwnerKey string                `json:"producerownerkey"`
		ProducerNodeKey  string                `json:"producernodekey"`
		ReferKey         string                `json:"referkey"`
		StakeAddress     string                `json:"stakeaddress"`
		TransactionHash  string                `json:"transactionhash"`
		BlockHeight      uint32                `json:"blockheight"`
		PayloadVersion   byte                  `json:"payloadversion"`
		VoteType         byte                  `json:"votetype"`
		Info             VotesWithLockTimeInfo `json:"info"`
		DPoSV2VoteRights string                `json:"DPoSV2VoteRights"`
	}

	var result []*detailedVoteInfo
	ps := Chain.GetState().GetAllProducers()
	for _, p := range ps {
		dposv2Votes := p.GetAllDetailedDPoSV2Votes()
		if len(dposv2Votes) == 0 {
			continue
		}
		for voterProgramHash, v := range dposv2Votes {
			for k1, v1 := range v {
				voterAddress, _ := voterProgramHash.ToAddress()
				//get stakeAddress all dposv2 votes
				if stakeAddress != "" && stakeAddress != voterAddress {
					continue
				}
				info := &detailedVoteInfo{
					ProducerOwnerKey: hex.EncodeToString(p.OwnerPublicKey()),
					ProducerNodeKey:  hex.EncodeToString(p.NodePublicKey()),
					ReferKey:         common.ToReversedString(k1),
					StakeAddress:     voterAddress,
					TransactionHash:  common.ToReversedString(v1.TransactionHash),
					BlockHeight:      v1.BlockHeight,
					PayloadVersion:   v1.PayloadVersion,
					VoteType:         byte(v1.VoteType),
					Info: VotesWithLockTimeInfo{
						Candidate: hex.EncodeToString(v1.Info[0].Candidate),
						Votes:     v1.Info[0].Votes.String(),
						LockTime:  v1.Info[0].LockTime,
					},
					DPoSV2VoteRights: v1.VoteRights().String(),
				}
				result = append(result, info)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.Compare(result[i].ReferKey, result[j].ReferKey) >= 0
	})

	count := int64(len(result))
	if limit < 0 {
		limit = count
	}
	var dvi []*detailedVoteInfo
	if start < count {
		end := start
		if start+limit <= count {
			end = start + limit
		} else {
			end = count
		}
		dvi = append(dvi, result[start:end]...)
	}

	return ResponsePack(Success, dvi)
}

// GetProducerInfo
func GetProducerInfo(params Params) map[string]interface{} {
	publicKey, ok := params.String("publickey")
	if !ok {
		return ResponsePack(InvalidParams, "public key not found")
	}
	publicKeyBytes, err := common.HexStringToBytes(publicKey)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid public key")
	}
	p := Chain.GetState().GetProducer(publicKeyBytes)
	if p == nil {
		return ResponsePack(InvalidParams, "unknown producer public key")
	}

	producerInfo := RPCProducerInfo{
		OwnerPublicKey: hex.EncodeToString(p.Info().OwnerKey),
		NodePublicKey:  hex.EncodeToString(p.Info().NodePublicKey),
		Nickname:       p.Info().NickName,
		Url:            p.Info().Url,
		Location:       p.Info().Location,
		StakeUntil:     p.Info().StakeUntil,
		Active:         p.State() == state.Active,
		Votes:          p.Votes().String(),
		DPoSV2Votes:    common.Fixed64(p.GetTotalDPoSV2VoteRights()).String(),
		State:          p.State().String(),
		Identity:       p.Identity().String(),
		RegisterHeight: p.RegisterHeight(),
		CancelHeight:   p.CancelHeight(),
		InactiveHeight: p.InactiveSince(),
		IllegalHeight:  p.IllegalHeight(),
		Index:          0,
	}
	return ResponsePack(Success, producerInfo)
}

func GetNFTInfo(params Params) map[string]interface{} {
	idParam, ok := params.String("id")
	if !ok {
		return ResponsePack(InvalidParams, "need string id ")
	}

	idBytes, err := common.HexStringToBytes(idParam)
	if err != nil {
		return ResponsePack(InvalidParams, "id HexStringToBytes error")
	}
	nftID, err := common.Uint256FromBytes(idBytes)
	if err != nil {
		return ResponsePack(InvalidParams, "idbytes to hash error")
	}

	type nftInfo struct {
		ID          string `json:"ID"`
		StartHeight uint32 `json:"startheight"`
		EndHeight   uint32 `json:"endheight"`
		Votes       string `json:"votes"`
		VotesRight  string `json:"votesright"`
		Rewards     string `json:"rewards"`
	}

	producers := Chain.GetState().GetAllProducers()

	fillNFTINFO := func(nftID common.Uint256, detailVoteInfo payload.DetailedVoteInfo) (info nftInfo) {
		ct, _ := contract.CreateStakeContractByCode(nftID.Bytes())
		nftStakeAddress, _ := ct.ToProgramHash().ToAddress()
		info.StartHeight = detailVoteInfo.BlockHeight
		info.EndHeight = detailVoteInfo.Info[0].LockTime
		info.Votes = detailVoteInfo.Info[0].Votes.String()
		info.Rewards = Chain.GetState().DPoSV2RewardInfo[nftStakeAddress].String()
		return
	}
	nftReferKey, err := Chain.GetState().GetNFTReferKey(*nftID)
	if err != nil {
		return ResponsePack(InvalidParams, "wrong nft id, not found it!")
	}
	//todo referkey recreate problem
	for _, producer := range producers {
		for _, votesInfo := range producer.GetAllDetailedDPoSV2Votes() {
			for referKey, detailVoteInfo := range votesInfo {
				if referKey.IsEqual(nftReferKey) {
					info := fillNFTINFO(*nftID, detailVoteInfo)
					info.ID = idParam
					info.VotesRight = common.Fixed64(producer.GetNFTVotesRight(referKey)).String()
					return ResponsePack(Success, info)
				}
			}
		}
		for _, expiredVotesInfo := range producer.GetExpiredNFTVotes() {
			if expiredVotesInfo.ReferKey().IsEqual(nftReferKey) {
				info := fillNFTINFO(*nftID, expiredVotesInfo)
				info.ID = idParam
				info.VotesRight = "0"
				return ResponsePack(Success, info)
			}
		}
	}

	return ResponsePack(InvalidParams, "wrong nft id, not found it!")
}

func GetReferKeyInfo(params Params) map[string]interface{} {
	idParam, ok := params.String("id")
	if !ok {
		return ResponsePack(InvalidParams, "need string id ")
	}

	idBytes, err := common.HexStringToBytes(idParam)
	if err != nil {
		return ResponsePack(InvalidParams, "id HexStringToBytes error")
	}
	nftID, err := common.Uint256FromBytes(idBytes)
	if err != nil {
		return ResponsePack(InvalidParams, "idbytes to hash error")
	}

	type VotesWithLockTime struct {
		Candidate string `json:"candiate"`
		Votes     int64  `json:"votes"`
		LockTime  uint32 `json:"lockTime"`
	}

	type ReferKeyInfo struct {
		Hash            string              `json:"hash"`
		TransactionHash string              `json:"transactionHash"`
		BlockHeight     uint32              `json:"blockHeight"`
		PayloadVersion  byte                `json:"payloadVersion"`
		VoteType        byte                `json:"voteType"`
		Info            []VotesWithLockTime `json:"info"`
	}

	fillReferKeyINFO := func(detailVoteInfo payload.DetailedVoteInfo) (info ReferKeyInfo) {
		info.TransactionHash = detailVoteInfo.TransactionHash.ReversedString()
		info.BlockHeight = detailVoteInfo.BlockHeight
		info.PayloadVersion = detailVoteInfo.PayloadVersion
		info.VoteType = byte(detailVoteInfo.VoteType)
		info.Info = []VotesWithLockTime{
			{
				Candidate: common.BytesToHexString(detailVoteInfo.Info[0].Candidate),
				Votes:     detailVoteInfo.Info[0].Votes.IntValue(),
				LockTime:  detailVoteInfo.Info[0].LockTime,
			},
		}
		return
	}

	nftReferKey, err := Chain.GetState().GetNFTReferKey(*nftID)
	if err != nil {
		return ResponsePack(InvalidParams, "wrong nft id, not found it!")
	}
	producers := Chain.GetState().GetAllProducers()
	for _, producer := range producers {
		for _, votesInfo := range producer.GetAllDetailedDPoSV2Votes() {
			for referKey, detailVoteInfo := range votesInfo {
				if referKey.IsEqual(nftReferKey) {
					info := fillReferKeyINFO(detailVoteInfo)
					info.Hash = nftReferKey.ReversedString()
					return ResponsePack(Success, info)
				}
			}
		}
		for _, expiredVotesInfo := range producer.GetExpiredNFTVotes() {
			if expiredVotesInfo.ReferKey().IsEqual(nftReferKey) {
				info := fillReferKeyINFO(expiredVotesInfo)
				info.Hash = nftReferKey.ReversedString()
				return ResponsePack(Success, info)
			}
		}
	}

	return ResponsePack(InvalidParams, "wrong nft id, not found it!")
}

func GetCanDestroynftIDs(params Params) map[string]interface{} {
	idsParam, ok := params.ArrayString("ids")
	if !ok {
		return ResponsePack(InvalidParams, "need ids in an array!")
	}
	genesisBlockStr, ok := params.String("genesisblockhash")
	if !ok {
		return ResponsePack(InvalidParams, "genesisblockhash not found")
	}
	genesisBlockhash, err := common.Uint256FromHexString(genesisBlockStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesisblockhash ")
	}

	var IDs []common.Uint256
	for i := 0; i < len(idsParam); i++ {
		idBytes, err := common.HexStringToBytes(idsParam[i])
		if err != nil {
			return ResponsePack(InvalidParams, "HexStringToBytes idsParam[i] error")
		}
		id, err := common.Uint256FromBytes(idBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "Uint256FromBytes error")
		}
		IDs = append(IDs, *id)
	}
	state := Chain.GetState()
	canDestroyIDs := state.CanNFTDestroy(IDs)
	var destoryIDs []string

	for _, id := range canDestroyIDs {
		if state.IsNFTIDBelongToSideChain(id, *genesisBlockhash) {
			destoryIDs = append(destoryIDs, id.String())
		}
	}

	return ResponsePack(Success, destoryIDs)
}

// by s address.
func GetVoteRights(params Params) map[string]interface{} {
	addresses, ok := params.ArrayString("stakeaddresses")
	if !ok {
		return ResponsePack(InvalidParams, "need stakeaddresses in an array!")
	}
	currentHeight := Chain.GetHeight()
	dposV2 := Arbiters.IsDPoSV2Run(currentHeight)

	type usedVoteRightDetailInfo struct {
		UsedDPoSVotes           []VotesWithLockTimeInfo `json:"useddposvotes"`
		UsedCRVotes             []VotesWithLockTimeInfo `json:"usedcrvotes"`
		UsedCRCProposalVotes    []VotesWithLockTimeInfo `json:"usedcrcproposalvotes"`
		UsdedCRImpeachmentVotes []VotesWithLockTimeInfo `json:"usdedcrimpeachmentvotes"`
		UsedDPoSV2Votes         []DetailedVoteInfo      `json:"useddposv2votes"`
	}

	type detailedVoteRight struct {
		StakeAddress    string                  `json:"stakeaddress"`
		TotalVotesRight string                  `json:"totalvotesright"`
		UsedVotesInfo   usedVoteRightDetailInfo `json:"usedvotesinfo"`
		RemainVoteRight []string                `json:"remainvoteright"` //index is same to VoteType
	}
	var result []*detailedVoteRight
	state := Chain.GetState()
	crstate := Chain.GetCRCommittee().GetState()
	for _, address := range addresses {
		if !strings.HasPrefix(address, "S") {
			return ResponsePack(InvalidParams, "invalid stake address need prefix s")
		}
		programhash, err := common.Uint168FromAddress(address)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid stake address")
		}
		voteRights := state.DposV2VoteRights
		stakeProgramHash := *programhash
		//get totalVotes
		totalVotesRight := voteRights[stakeProgramHash]
		vote := &detailedVoteRight{
			StakeAddress:    address,
			TotalVotesRight: totalVotesRight.String(),
			UsedVotesInfo: usedVoteRightDetailInfo{
				UsedDPoSV2Votes:         []DetailedVoteInfo{},
				UsedDPoSVotes:           []VotesWithLockTimeInfo{},
				UsedCRVotes:             []VotesWithLockTimeInfo{},
				UsdedCRImpeachmentVotes: []VotesWithLockTimeInfo{},
				UsedCRCProposalVotes:    []VotesWithLockTimeInfo{},
			},
			RemainVoteRight: make([]string, 5),
		}
		// dposv1

		if udv := state.UsedDposVotes[stakeProgramHash]; !dposV2 && udv != nil {
			for _, v := range udv {
				vote.UsedVotesInfo.UsedDPoSVotes = append(vote.UsedVotesInfo.UsedDPoSVotes, VotesWithLockTimeInfo{
					Candidate: hex.EncodeToString(v.Candidate),
					Votes:     v.Votes.String(),
					LockTime:  v.LockTime,
				})
			}
		}
		// crc
		if ucv := crstate.UsedCRVotes[stakeProgramHash]; ucv != nil {
			for _, v := range ucv {
				c, _ := common.Uint168FromBytes(v.Candidate)
				candidate, _ := c.ToAddress()
				vote.UsedVotesInfo.UsedCRVotes = append(vote.UsedVotesInfo.UsedCRVotes, VotesWithLockTimeInfo{
					Candidate: candidate,
					Votes:     v.Votes.String(),
					LockTime:  v.LockTime,
				})
			}
		}
		// cr Impeachment
		if uciv := crstate.UsedCRImpeachmentVotes[stakeProgramHash]; uciv != nil {
			for _, v := range uciv {
				c, _ := common.Uint168FromBytes(v.Candidate)
				candidate, _ := c.ToAddress()
				vote.UsedVotesInfo.UsdedCRImpeachmentVotes = append(vote.UsedVotesInfo.UsdedCRImpeachmentVotes, VotesWithLockTimeInfo{
					Candidate: candidate,
					Votes:     v.Votes.String(),
					LockTime:  v.LockTime,
				})
			}
		}

		// cr Proposal
		if ucpv := crstate.UsedCRCProposalVotes[stakeProgramHash]; ucpv != nil {
			for _, v := range ucpv {
				proposalHash, _ := common.Uint256FromBytes(v.Candidate)
				vote.UsedVotesInfo.UsedCRCProposalVotes = append(vote.UsedVotesInfo.UsedCRCProposalVotes, VotesWithLockTimeInfo{
					Candidate: common.ToReversedString(*proposalHash),
					Votes:     v.Votes.String(),
					LockTime:  v.LockTime,
				})
			}
		}

		// dposv2
		if dpv2 := state.GetDetailedDPoSV2Votes(&stakeProgramHash); dpv2 != nil {
			for i, v := range dpv2 {
				address, _ := v.StakeProgramHash.ToAddress()
				vote.UsedVotesInfo.UsedDPoSV2Votes = append(vote.UsedVotesInfo.UsedDPoSV2Votes, DetailedVoteInfo{
					StakeAddress:    address,
					TransactionHash: common.ToReversedString(v.TransactionHash),
					BlockHeight:     v.BlockHeight,
					PayloadVersion:  v.PayloadVersion,
					VoteType:        uint32(v.VoteType),
				})

				if v.Info != nil {
					for _, v := range v.Info {
						vote.UsedVotesInfo.UsedDPoSV2Votes[i].Info = append(vote.UsedVotesInfo.UsedDPoSV2Votes[i].Info, VotesWithLockTimeInfo{
							Candidate: hex.EncodeToString(v.Candidate),
							Votes:     v.Votes.String(),
							LockTime:  v.LockTime,
						})
					}
				}
			}
		}

		//fill RemainVoteRight
		for i := outputpayload.Delegate; i <= outputpayload.DposV2; i++ {
			usedVoteRight, _ := GetUsedVoteRight(i, &stakeProgramHash)
			remainRoteRight := totalVotesRight - usedVoteRight
			if dposV2 && i == outputpayload.Delegate {
				vote.RemainVoteRight[i] = common.Fixed64(0).String()
			} else {
				vote.RemainVoteRight[i] = remainRoteRight.String()
			}
		}
		result = append(result, vote)
	}
	return ResponsePack(Success, result)
}

func GetUsedVoteRight(voteType outputpayload.VoteType, stakeProgramHash *common.Uint168) (common.Fixed64, error) {
	state := Chain.GetState()
	crstate := Chain.GetCRCommittee().GetState()
	usedDposVote := common.Fixed64(0)
	switch voteType {
	case outputpayload.Delegate:
		if Chain.GetHeight() >= Chain.GetState().DPoSV2ActiveHeight {
			usedDposVote = 0
		} else {
			if dposVotes, ok := state.UsedDposVotes[*stakeProgramHash]; ok {
				maxVotes := common.Fixed64(0)
				for _, votesInfo := range dposVotes {
					if votesInfo.Votes > maxVotes {
						maxVotes = votesInfo.Votes
					}
					usedDposVote = maxVotes
				}
			}
		}
	case outputpayload.CRC:
		if usedCRVoteRights, ok := crstate.UsedCRVotes[*stakeProgramHash]; ok {
			for _, votesInfo := range usedCRVoteRights {
				usedDposVote += votesInfo.Votes
			}
		}
	case outputpayload.CRCProposal:
		if usedCRCProposalVoteRights, ok := crstate.UsedCRCProposalVotes[*stakeProgramHash]; ok {
			maxVotes := common.Fixed64(0)
			for _, votesInfo := range usedCRCProposalVoteRights {
				if votesInfo.Votes > maxVotes {
					maxVotes = votesInfo.Votes
				}
			}
			usedDposVote = maxVotes
		}

	case outputpayload.CRCImpeachment:
		if usedCRImpeachmentVoteRights, ok := crstate.UsedCRImpeachmentVotes[*stakeProgramHash]; ok {
			for _, votesInfo := range usedCRImpeachmentVoteRights {
				usedDposVote += votesInfo.Votes
			}
		}
	case outputpayload.DposV2:
		addr, _ := stakeProgramHash.ToAddress()
		fmt.Println("addr", addr)
		usedDposVote = state.UsedDposV2Votes[*stakeProgramHash]
	default:
		return 0, errors.New("unsupport vote type")
	}
	return usedDposVote, nil
}

func GetArbitersInfo(params Params) map[string]interface{} {
	type arbitersInfo struct {
		Arbiters               []string `json:"arbiters"`
		Candidates             []string `json:"candidates"`
		NextArbiters           []string `json:"nextarbiters"`
		NextCandidates         []string `json:"nextcandidates"`
		OnDutyArbiter          string   `json:"ondutyarbiter"`
		CurrentTurnStartHeight int      `json:"currentturnstartheight"`
		NextTurnStartHeight    int      `json:"nextturnstartheight"`
	}

	dutyIndex := Arbiters.GetDutyIndex()
	result := &arbitersInfo{
		Arbiters:       make([]string, 0),
		Candidates:     make([]string, 0),
		NextArbiters:   make([]string, 0),
		NextCandidates: make([]string, 0),
		OnDutyArbiter:  common.BytesToHexString(Arbiters.GetOnDutyArbitrator()),

		CurrentTurnStartHeight: int(Chain.GetHeight()) - dutyIndex,
		NextTurnStartHeight: int(Chain.GetHeight()) +
			Arbiters.GetArbitersCount() - dutyIndex,
	}
	for _, v := range Arbiters.GetArbitrators() {
		var nodePK string
		if v.IsNormal {
			nodePK = common.BytesToHexString(v.NodePublicKey)
		}
		result.Arbiters = append(result.Arbiters, nodePK)
	}
	for _, v := range Arbiters.GetCandidates() {
		result.Candidates = append(result.Candidates, common.BytesToHexString(v))
	}
	for _, v := range Arbiters.GetNextArbitrators() {
		var nodePK string
		if v.IsNormal {
			nodePK = common.BytesToHexString(v.NodePublicKey)
		}
		result.NextArbiters = append(result.NextArbiters, nodePK)
	}
	for _, v := range Arbiters.GetNextCandidates() {
		result.NextCandidates = append(result.NextCandidates,
			common.BytesToHexString(v))
	}
	return ResponsePack(Success, result)
}

func GetInfo(param Params) map[string]interface{} {
	RetVal := struct {
		Version       uint32 `json:"version"`
		Balance       int    `json:"balance"`
		Blocks        uint32 `json:"blocks"`
		Timeoffset    int    `json:"timeoffset"`
		Connections   int32  `json:"connections"`
		Testnet       bool   `json:"testnet"`
		Keypoololdest int    `json:"keypoololdest"`
		Keypoolsize   int    `json:"keypoolsize"`
		UnlockedUntil int    `json:"unlocked_until"`
		Paytxfee      int    `json:"paytxfee"`
		Relayfee      int    `json:"relayfee"`
		Errors        string `json:"errors"`
	}{
		Version:       pact.DPOSStartVersion,
		Balance:       0,
		Blocks:        Chain.GetHeight(),
		Timeoffset:    0,
		Connections:   Server.ConnectedCount(),
		Keypoololdest: 0,
		Keypoolsize:   0,
		UnlockedUntil: 0,
		Paytxfee:      0,
		Relayfee:      0,
		Errors:        "Tobe written"}
	return ResponsePack(Success, &RetVal)
}

func AuxHelp(param Params) map[string]interface{} {
	return ResponsePack(Success, "createauxblock==submitauxblock")
}

func GetMiningInfo(param Params) map[string]interface{} {
	block, err := Chain.GetBlockByHash(Chain.GetCurrentBlockHash())
	if err != nil {
		return ResponsePack(InternalError, "get tip block failed")
	}

	miningInfo := struct {
		Blocks         uint32 `json:"blocks"`
		CurrentBlockTx uint32 `json:"currentblocktx"`
		Difficulty     string `json:"difficulty"`
		NetWorkHashPS  string `json:"networkhashps"`
		PooledTx       uint32 `json:"pooledtx"`
		Chain          string `json:"chain"`
	}{
		Blocks:         Chain.GetHeight() + 1,
		CurrentBlockTx: uint32(len(block.Transactions)),
		Difficulty:     Chain.CalcCurrentDifficulty(block.Bits),
		NetWorkHashPS:  Chain.GetNetworkHashPS().String(),
		PooledTx:       uint32(len(TxMemPool.GetTxsInPool())),
		Chain:          ChainParams.ActiveNet,
	}

	return ResponsePack(Success, miningInfo)
}

func ToggleMining(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.ConfigurationPermitted); rtn != nil {
		return rtn
	}

	mining, ok := param.Bool("mining")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	var message string
	if mining {
		go Pow.Start()
		message = "mining started"
	} else {
		go Pow.Halt()
		message = "mining stopped"
	}

	return ResponsePack(Success, message)
}

func DiscreteMining(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.MiningPermitted); rtn != nil {
		return rtn
	}

	if Pow == nil {
		return ResponsePack(PowServiceNotStarted, "")
	}
	count, ok := param.Uint("count")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	ret := make([]string, 0)

	blockHashes, err := Pow.DiscreteMining(uint32(count))
	if err != nil {
		return ResponsePack(Error, err.Error())
	}

	for _, hash := range blockHashes {
		retStr := common.ToReversedString(*hash)
		ret = append(ret, retStr)
	}

	return ResponsePack(Success, ret)
}

func GetConnectionCount(param Params) map[string]interface{} {
	return ResponsePack(Success, Server.ConnectedCount())
}

func GetTransactionPool(param Params) map[string]interface{} {
	str, ok := param.String("state")
	if ok {
		switch str {
		case "all":
			txs := make([]*TransactionContextInfo, 0)
			for _, tx := range TxMemPool.GetTxsInPool() {
				txs = append(txs, GetTransactionContextInfo(nil, tx))
			}
			return ResponsePack(Success, txs)
		}
	}

	txs := make([]string, 0)
	for _, tx := range TxMemPool.GetTxsInPool() {
		txs = append(txs, common.ToReversedString(tx.Hash()))
	}
	return ResponsePack(Success, txs)

}

func GetBlockInfo(block *Block, verbose bool) BlockInfo {
	var txs []interface{}
	if verbose {
		for _, tx := range block.Transactions {
			txs = append(txs, GetTransactionContextInfo(&block.Header, tx))
		}
	} else {
		for _, tx := range block.Transactions {
			txs = append(txs, common.ToReversedString(tx.Hash()))
		}
	}
	var versionBytes [4]byte
	binary.BigEndian.PutUint32(versionBytes[:], block.Header.Version)

	var chainWork [4]byte
	binary.BigEndian.PutUint32(chainWork[:], Chain.GetHeight()-block.Header.Height)

	nextBlockHash, _ := Chain.GetBlockHash(block.Header.Height + 1)

	auxPow := new(bytes.Buffer)
	block.Header.AuxPow.Serialize(auxPow)

	return BlockInfo{
		Hash:              common.ToReversedString(block.Hash()),
		Confirmations:     Chain.GetHeight() - block.Header.Height + 1,
		StrippedSize:      uint32(block.GetSize()),
		Size:              uint32(block.GetSize()),
		Weight:            uint32(block.GetSize() * 4),
		Height:            block.Header.Height,
		Version:           block.Header.Version,
		VersionHex:        common.BytesToHexString(versionBytes[:]),
		MerkleRoot:        common.ToReversedString(block.Header.MerkleRoot),
		Tx:                txs,
		Time:              block.Header.Timestamp,
		MedianTime:        block.Header.Timestamp,
		Nonce:             block.Header.Nonce,
		Bits:              block.Header.Bits,
		Difficulty:        Chain.CalcCurrentDifficulty(block.Header.Bits),
		ChainWork:         common.BytesToHexString(chainWork[:]),
		PreviousBlockHash: common.ToReversedString(block.Header.Previous),
		NextBlockHash:     common.ToReversedString(nextBlockHash),
		AuxPow:            common.BytesToHexString(auxPow.Bytes()),
		MinerInfo:         string(block.Transactions[0].Payload().(*payload.CoinBase).Content[:]),
	}
}

func GetConfirmInfo(confirm *payload.Confirm) ConfirmInfo {
	votes := make([]VoteInfo, 0)
	for _, vote := range confirm.Votes {
		votes = append(votes, VoteInfo{
			Signer: common.BytesToHexString(vote.Signer),
			Accept: vote.Accept,
		})
	}

	return ConfirmInfo{
		BlockHash:  common.ToReversedString(confirm.Proposal.BlockHash),
		Sponsor:    common.BytesToHexString(confirm.Proposal.Sponsor),
		ViewOffset: confirm.Proposal.ViewOffset,
		Votes:      votes,
	}
}

func getBlock(hash common.Uint256, verbose uint32) (interface{}, ServerErrCode) {
	block, err := Chain.GetBlockByHash(hash)
	if err != nil {
		return "", UnknownBlock
	}
	switch verbose {
	case 0:
		w := new(bytes.Buffer)
		block.Serialize(w)
		return common.BytesToHexString(w.Bytes()), Success
	case 2:
		return GetBlockInfo(block, true), Success
	}
	return GetBlockInfo(block, false), Success
}

func getConfirm(hash common.Uint256, verbose uint32) (interface{}, ServerErrCode) {
	block, _ := Store.GetFFLDB().GetBlock(hash)
	if block == nil {
		return "", UnknownBlock
	} else if !block.HaveConfirm {
		return "", UnknownConfirm
	}
	if verbose == 0 {
		w := new(bytes.Buffer)
		block.Confirm.Serialize(w)
		return common.BytesToHexString(w.Bytes()), Success
	}

	return GetConfirmInfo(block.Confirm), Success
}

func GetBlockByHash(param Params) map[string]interface{} {
	str, ok := param.String("blockhash")
	if !ok {
		return ResponsePack(InvalidParams, "block hash not found")
	}

	var hash common.Uint256
	hashBytes, err := common.FromReversedString(str)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid block hash")
	}
	if err := hash.Deserialize(bytes.NewReader(hashBytes)); err != nil {
		ResponsePack(InvalidParams, "invalid block hash")
	}

	verbosity, ok := param.Uint("verbosity")
	if !ok {
		verbosity = 1
	}

	result, error := getBlock(hash, verbosity)

	return ResponsePack(error, result)
}

func GetConfirmByHeight(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}

	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, err.Error())
	}

	verbosity, ok := param.Uint("verbosity")
	if !ok {
		verbosity = 1
	}

	result, errCode := getConfirm(hash, verbosity)
	return ResponsePack(errCode, result)
}

func GetConfirmByHash(param Params) map[string]interface{} {
	str, ok := param.String("blockhash")
	if !ok {
		return ResponsePack(InvalidParams, "block hash not found")
	}

	var hash common.Uint256
	hashBytes, err := common.FromReversedString(str)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid block hash")
	}
	if err := hash.Deserialize(bytes.NewReader(hashBytes)); err != nil {
		ResponsePack(InvalidParams, "invalid block hash")
	}

	verbosity, ok := param.Uint("verbosity")
	if !ok {
		verbosity = 1
	}

	result, error := getConfirm(hash, verbosity)
	return ResponsePack(error, result)
}

func SendRawTransaction(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.TransactionPermitted); rtn != nil {
		return rtn
	}

	str, ok := param.String("data")
	if !ok {
		return ResponsePack(InvalidParams, "need a string parameter named data")
	}
	log.Info("RawTx Received by SendRawTransaction:", str)
	bys, err := common.HexStringToBytes(str)
	if err != nil {
		return ResponsePack(InvalidParams, "hex string to bytes error")
	}

	r := bytes.NewReader(bys)
	txn, err := functions.GetTransactionByBytes(r)
	if err != nil {
		return ResponsePack(InvalidTransaction, "invalid transaction")
	}
	if err := txn.Deserialize(r); err != nil {
		return ResponsePack(InvalidTransaction, err.Error())
	}

	if err := VerifyAndSendTx(txn); err != nil {
		return ResponsePack(InvalidTransaction, err.Error())
	}

	return ResponsePack(Success, common.ToReversedString(txn.Hash()))
}

func GetBlockHeight(param Params) map[string]interface{} {
	return ResponsePack(Success, Chain.GetHeight())
}

func GetBestBlockHash(param Params) map[string]interface{} {
	hash, err := Chain.GetBlockHash(Chain.GetHeight())
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	return ResponsePack(Success, common.ToReversedString(hash))
}

func GetBlockCount(param Params) map[string]interface{} {
	return ResponsePack(Success, Chain.GetHeight()+1)
}

func GetBlockHash(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}

	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	return ResponsePack(Success, common.ToReversedString(hash))
}

func GetBlockTransactions(block *Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = common.ToReversedString(block.Transactions[i].Hash())
	}
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         common.ToReversedString(block.Hash()),
		Height:       block.Header.Height,
		Transactions: trans,
	}
	return b
}

func GetTransactionsByHeight(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}

	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, "")

	}
	block, err := Chain.GetBlockByHash(hash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	return ResponsePack(Success, GetBlockTransactions(block))
}

func GetBlockByHeight(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}

	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, err.Error())
	}

	result, errCode := getBlock(hash, 2)

	return ResponsePack(errCode, result)
}

func GetArbitratorGroupByHeight(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}

	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, "not found block hash at given height")
	}

	block, _ := Chain.GetBlockByHash(hash)
	if block == nil {
		return ResponsePack(InternalError, "not found block at given height")
	}

	result := ArbitratorGroupInfo{}
	if height < ChainParams.DPoSConfiguration.DPOSNodeCrossChainHeight {
		crcArbiters := Arbiters.GetCRCArbiters()
		sort.Slice(crcArbiters, func(i, j int) bool {
			return bytes.Compare(crcArbiters[i].NodePublicKey, crcArbiters[j].NodePublicKey) < 0
		})
		var arbitrators []string
		for _, a := range crcArbiters {
			if !a.IsNormal {
				arbitrators = append(arbitrators, "")
			} else {
				arbitrators = append(arbitrators, common.BytesToHexString(a.NodePublicKey))
			}
		}

		result = ArbitratorGroupInfo{
			OnDutyArbitratorIndex: Arbiters.GetDutyIndexByHeight(height),
			Arbitrators:           arbitrators,
		}
	} else {
		arbiters := Arbiters.GetArbitrators()
		arbitrators := make([]string, 0)
		for _, a := range arbiters {
			if !a.IsNormal {
				arbitrators = append(arbitrators, "")
			} else {
				arbitrators = append(arbitrators, common.BytesToHexString(a.NodePublicKey))
			}
		}

		result = ArbitratorGroupInfo{
			OnDutyArbitratorIndex: Arbiters.GetDutyIndexByHeight(height),
			Arbitrators:           arbitrators,
		}
	}

	return ResponsePack(Success, result)
}

// GetAssetByHash always return ELA asset
// Deprecated: It may be removed in the next version
func GetAssetByHash(param Params) map[string]interface{} {
	asset := payload.RegisterAsset{
		Asset: payload.Asset{
			Name:      "ELA",
			Precision: core.ELAPrecision,
			AssetType: 0x00,
		},
		Amount:     0 * 100000000,
		Controller: common.Uint168{},
	}

	return ResponsePack(Success, asset)
}

func GetBalanceByAddr(param Params) map[string]interface{} {
	address, ok := param.String("addr")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}
	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	var balance common.Fixed64 = 0
	for _, u := range utxos {
		balance = balance + u.Value
	}

	return ResponsePack(Success, balance.String())
}

// Deprecated: May be removed in the next version
func GetBalanceByAsset(param Params) map[string]interface{} {
	address, ok := param.String("addr")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	var balance common.Fixed64 = 0
	for _, u := range utxos {
		balance = balance + u.Value
	}
	return ResponsePack(Success, balance.String())
}

func Getallregistertransactions(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()
	rs := crCommittee.GetAllRegisteredSideChain()
	var result []RsInfo
	for k, v := range rs {
		for k1, v1 := range v {
			result = append(result, RsInfo{
				SideChainName:   v1.SideChainName,
				MagicNumber:     v1.MagicNumber,
				GenesisHash:     common.ToReversedString(v1.GenesisHash),
				ExchangeRate:    v1.ExchangeRate,
				TxHash:          common.ToReversedString(k1),
				Height:          k,
				EffectiveHeight: v1.EffectiveHeight,
				ResourcePath:    v1.ResourcePath,
			})
		}
	}
	return ResponsePack(Success, result)
}
func Getregistertransactionsbyheight(param Params) map[string]interface{} {
	height, ok := param.Uint("height")
	if !ok {
		return ResponsePack(InvalidParams, "height parameter should be a positive integer")
	}
	crCommittee := Chain.GetCRCommittee()

	rs := crCommittee.GetRegisteredSideChainByHeight(height)
	var result []RsInfo
	for k, v := range rs {
		result = append(result, RsInfo{
			SideChainName:   v.SideChainName,
			MagicNumber:     v.MagicNumber,
			GenesisHash:     common.ToReversedString(v.GenesisHash),
			ExchangeRate:    v.ExchangeRate,
			EffectiveHeight: v.EffectiveHeight,
			ResourcePath:    v.ResourcePath,
			TxHash:          common.ToReversedString(k),
			Height:          height,
		})
	}
	return ResponsePack(Success, result)
}

func GetReceivedByAddress(param Params) map[string]interface{} {
	address, ok := param.String("address")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named address")
	}
	spendable := false
	if s, ok := param.Bool("spendable"); ok {
		spendable = s
	}
	bestHeight := Chain.GetHeight()
	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	var balance common.Fixed64 = 0
	for _, u := range utxos {
		tx, height, err := Store.GetTransaction(u.TxID)
		if err != nil {
			return ResponsePack(InternalError, "unknown transaction "+
				u.TxID.String()+" from persisted utxo")
		}
		if spendable && tx.IsCoinBaseTx() {
			if bestHeight-height < ChainParams.PowConfiguration.CoinbaseMaturity {
				continue
			}
		}
		balance = balance + u.Value
	}

	return ResponsePack(Success, balance.String())
}

func GetUTXOsByAmount(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	bestHeight := Chain.GetHeight()

	result := make([]UTXOInfo, 0)
	address, ok := param.String("address")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named address!")
	}
	amountStr, ok := param.String("amount")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named amount!")
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid amount!")
	}
	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	utxoType := "mixed"
	if t, ok := param.String("utxotype"); ok {
		switch t {
		case "mixed", "vote", "normal", "unused":
			utxoType = t
		default:
			return ResponsePack(InvalidParams, "invalid utxotype")
		}
	}

	if utxoType == "unused" {
		var unusedUTXOs []*common2.UTXO
		usedUTXOs := TxMemPool.GetUsedUTXOs()
		for _, u := range utxos {
			outPoint := common2.OutPoint{TxID: u.TxID, Index: u.Index}
			referKey := outPoint.ReferKey()
			if _, ok := usedUTXOs[referKey]; !ok {
				unusedUTXOs = append(unusedUTXOs, u)
			}
		}
		utxos = unusedUTXOs
	}

	totalAmount := common.Fixed64(0)
	for _, utxo := range utxos {
		if totalAmount >= *amount {
			break
		}
		tx, height, err := Store.GetTransaction(utxo.TxID)
		if err != nil {
			return ResponsePack(InternalError, "unknown transaction "+
				utxo.TxID.String()+" from persisted utxo")
		}
		if utxoType == "vote" && (tx.Version() < common2.TxVersion09 ||
			tx.Version() >= common2.TxVersion09 && tx.Outputs()[utxo.Index].Type != common2.OTVote) {
			continue
		}
		if utxoType == "normal" && tx.Version() >= common2.TxVersion09 &&
			tx.Outputs()[utxo.Index].Type == common2.OTVote {
			continue
		}
		if tx.TxType() == common2.CoinBase && bestHeight-height < ChainParams.PowConfiguration.CoinbaseMaturity {
			continue
		}
		totalAmount += utxo.Value
		result = append(result, UTXOInfo{
			TxType:        byte(tx.TxType()),
			TxID:          common.ToReversedString(utxo.TxID),
			AssetID:       common.ToReversedString(core.ELAAssetID),
			VOut:          utxo.Index,
			Amount:        utxo.Value.String(),
			Address:       address,
			OutputLock:    tx.Outputs()[utxo.Index].OutputLock,
			Confirmations: bestHeight - height + 1,
		})
	}

	if totalAmount < *amount {
		return ResponsePack(InternalError, "not enough utxo")
	}

	return ResponsePack(Success, result)
}

func GetAmountByInputs(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	inputStr, ok := param.String("inputs")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named inputs!")
	}

	inputBytes, _ := common.HexStringToBytes(inputStr)
	r := bytes.NewReader(inputBytes)
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid inputs")
	}

	amount := common.Fixed64(0)
	for i := uint64(0); i < count; i++ {
		input := new(common2.Input)
		if err := input.Deserialize(r); err != nil {
			return ResponsePack(InvalidParams, "invalid inputs")
		}
		tx, _, err := Store.GetTransaction(input.Previous.TxID)
		if err != nil {
			return ResponsePack(InternalError, "unknown transaction "+
				input.Previous.TxID.String()+" from persisted utxo")
		}
		amount += tx.Outputs()[input.Previous.Index].Value
	}

	return ResponsePack(Success, amount.String())
}

func ListUnspent(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	bestHeight := Chain.GetHeight()

	var result []UTXOInfo
	addresses, ok := param.ArrayString("addresses")
	if !ok {
		return ResponsePack(InvalidParams, "need addresses in an array!")
	}
	utxoType := "mixed"
	if t, ok := param.String("utxotype"); ok {
		switch t {
		case "mixed", "vote", "normal":
			utxoType = t
		default:
			return ResponsePack(InvalidParams, "invalid utxotype")
		}
	}
	spendable := false
	if s, ok := param.Bool("spendable"); ok {
		spendable = s
	}
	for _, address := range addresses {
		programHash, err := common.Uint168FromAddress(address)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid address, "+err.Error())
		}
		utxos, err := Store.GetFFLDB().GetUTXO(programHash)
		if err != nil {
			return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
		}
		for _, utxo := range utxos {
			tx, height, err := Store.GetTransaction(utxo.TxID)
			if err != nil {
				return ResponsePack(InternalError,
					"unknown transaction "+utxo.TxID.String()+" from persisted utxo")
			}
			if utxoType == "vote" && (tx.Version() < common2.TxVersion09 ||
				tx.Version() >= common2.TxVersion09 && tx.Outputs()[utxo.Index].Type != common2.OTVote) {
				continue
			}
			if utxoType == "normal" && tx.Version() >= common2.TxVersion09 && tx.Outputs()[utxo.Index].Type == common2.OTVote {
				continue
			}
			if spendable && tx.IsCoinBaseTx() {
				if bestHeight-height < ChainParams.PowConfiguration.CoinbaseMaturity {
					continue
				}
			}
			if utxo.Value == 0 {
				continue
			}
			result = append(result, UTXOInfo{
				TxType:        byte(tx.TxType()),
				TxID:          common.ToReversedString(utxo.TxID),
				AssetID:       common.ToReversedString(core.ELAAssetID),
				VOut:          utxo.Index,
				Amount:        utxo.Value.String(),
				Address:       address,
				OutputLock:    tx.Outputs()[utxo.Index].OutputLock,
				Confirmations: bestHeight - height + 1,
			})
		}
	}
	return ResponsePack(Success, result)
}

func CreateRawTransaction(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	inputsParam, ok := param.String("inputs")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named inputs")
	}
	outputsParam, ok := param.String("outputs")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named outputs")
	}
	locktime, ok := param.Uint("locktime")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named locktime")
	}

	inputs := make([]string, 0)
	gjson.Parse(inputsParam).ForEach(func(key, value gjson.Result) bool {
		inputs = append(inputs, value.String())
		return true
	})

	outputs := make([]string, 0)
	gjson.Parse(outputsParam).ForEach(func(key, value gjson.Result) bool {
		outputs = append(outputs, value.String())
		return true
	})

	txInputs := make([]*common2.Input, 0)
	for _, v := range inputs {
		txIDStr := gjson.Get(v, "txid").String()
		txIDBytes, err := common.HexStringToBytes(txIDStr)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid txid when convert to bytes")
		}
		txID, err := common.Uint256FromBytes(common.BytesReverse(txIDBytes))
		if err != nil {
			return ResponsePack(InvalidParams, "invalid txid in inputs param")
		}
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *txID,
				Index: uint16(gjson.Get(v, "vout").Int()),
			},
		}
		txInputs = append(txInputs, input)
	}

	txOutputs := make([]*common2.Output, 0)
	for _, v := range outputs {
		amount := gjson.Get(v, "amount").String()
		value, err := common.StringToFixed64(amount)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid amount in inputs param")
		}
		address := gjson.Get(v, "address").String()
		programHash, err := common.Uint168FromAddress(address)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid address in outputs param")
		}
		output := &common2.Output{
			AssetID:     *account.SystemAssetID,
			Value:       *value,
			OutputLock:  0,
			ProgramHash: *programHash,
			Type:        common2.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		}
		txOutputs = append(txOutputs, output)
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		txInputs,
		txOutputs,
		locktime,
		[]*pg.Program{},
	)

	buf := new(bytes.Buffer)
	err := txn.Serialize(buf)
	if err != nil {
		return ResponsePack(InternalError, "txn serialize failed")
	}

	return ResponsePack(Success, common.BytesToHexString(buf.Bytes()))
}

func SignRawTransactionWithKey(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	dataParam, ok := param.String("data")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named data")
	}
	codesParam, ok := param.String("codes")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named codes")
	}
	privkeysParam, ok := param.String("privkeys")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named privkeys")
	}

	privkeys := make([]string, 0)
	gjson.Parse(privkeysParam).ForEach(func(key, value gjson.Result) bool {
		privkeys = append(privkeys, value.String())
		return true
	})

	accounts := make(map[common.Uint160]*account.Account, 0)
	for _, privkeyStr := range privkeys {
		privkey, err := common.HexStringToBytes(privkeyStr)
		if err != nil {
			return ResponsePack(InvalidParams, err.Error())
		}
		acc, err := account.NewAccountWithPrivateKey(privkey)
		if err != nil {
			return ResponsePack(InvalidTransaction, err.Error())
		}
		accounts[acc.ProgramHash.ToCodeHash()] = acc
	}

	txBytes, err := common.HexStringToBytes(dataParam)
	if err != nil {
		return ResponsePack(InvalidParams, "hex string to bytes error")
	}
	r := bytes.NewReader(txBytes)
	txn, err := functions.GetTransactionByBytes(r)
	if err != nil {
		return ResponsePack(InvalidTransaction, "invalid transaction")
	}
	if err := txn.Deserialize(r); err != nil {
		return ResponsePack(InvalidTransaction, err.Error())
	}

	codes := make([]string, 0)
	gjson.Parse(codesParam).ForEach(func(key, value gjson.Result) bool {
		codes = append(codes, value.String())
		return true
	})

	programs := make([]*pg.Program, 0)
	if len(txn.Programs()) > 0 {
		programs = txn.Programs()
	} else {
		for _, codeStr := range codes {
			code, err := common.HexStringToBytes(codeStr)
			if err != nil {
				return ResponsePack(InvalidParams, "invalid params codes")
			}
			program := &pg.Program{
				Code:      code,
				Parameter: nil,
			}
			programs = append(programs, program)
		}
	}

	signData := new(bytes.Buffer)
	if err := txn.SerializeUnsigned(signData); err != nil {
		return ResponsePack(InvalidTransaction, err.Error())
	}

	references, err := Chain.UTXOCache.GetTxReference(txn)
	if err != nil {
		return ResponsePack(InvalidTransaction, err.Error())
	}

	programHashes, err := blockchain.GetTxProgramHashes(txn, references)
	if err != nil {
		return ResponsePack(InternalError, err.Error())
	}

	if len(programs) != len(programHashes) {
		return ResponsePack(InternalError, "the number of program hashes is different with number of programs")
	}

	// sort the program hashes of owner and programs of the transaction
	common.SortProgramHashByCodeHash(programHashes)
	blockchain.SortPrograms(programs)

	for i, programHash := range programHashes {
		program := programs[i]
		codeHash := common.ToCodeHash(program.Code)
		ownerHash := programHash.ToCodeHash()
		if !codeHash.IsEqual(ownerHash) {
			return ResponsePack(InternalError, "the program hashes is different with corresponding program code")
		}

		prefixType := contract.GetPrefixType(programHash)
		if prefixType == contract.PrefixStandard {
			signedProgram, err := account.SignStandardTransaction(txn, program, accounts)
			if err != nil {
				return ResponsePack(InternalError, err.Error())
			}
			programs[i] = signedProgram
		} else if prefixType == contract.PrefixMultiSig {
			signedProgram, err := account.SignMultiSignTransaction(txn, program, accounts)
			if err != nil {
				return ResponsePack(InternalError, err.Error())
			}
			programs[i] = signedProgram
		} else {
			return ResponsePack(InternalError, "invalid program hash type")
		}
	}
	txn.SetPrograms(programs)

	result := new(bytes.Buffer)
	if err := txn.Serialize(result); err != nil {
		return ResponsePack(InternalError, err.Error())
	}

	return ResponsePack(Success, common.BytesToHexString(result.Bytes()))
}

func GetUnspends(param Params) map[string]interface{} {
	address, ok := param.String("addr")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	type UTXOUnspentInfo struct {
		TxID  string `json:"Txid"`
		Index uint16 `json:"Index"`
		Value string `json:"Value"`
	}
	type Result struct {
		AssetID   string            `json:"AssetId"`
		AssetName string            `json:"AssetName"`
		UTXO      []UTXOUnspentInfo `json:"UTXO"`
	}
	var results []Result

	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	for _, u := range utxos {
		var unspendsInfo []UTXOUnspentInfo
		unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{
			common.ToReversedString(u.TxID),
			u.Index,
			u.Value.String()})

		results = append(results, Result{
			common.ToReversedString(core.ELAAssetID),
			"ELA",
			unspendsInfo})
	}
	return ResponsePack(Success, results)
}

// Deprecated: May be removed in the next version
func GetUnspendOutput(param Params) map[string]interface{} {
	addr, ok := param.String("addr")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}
	programHash, err := common.Uint168FromAddress(addr)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	type UTXOUnspentInfo struct {
		TxID  string `json:"Txid"`
		Index uint16 `json:"Index"`
		Value string `json:"Value"`
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, utxo := range utxos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{
			TxID:  common.ToReversedString(utxo.TxID),
			Index: utxo.Index,
			Value: utxo.Value.String()})
	}
	return ResponsePack(Success, UTXOoutputs)
}

// BaseTransaction
func GetTransactionByHash(param Params) map[string]interface{} {
	str, ok := param.String("hash")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	bys, err := common.FromReversedString(str)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	var hash common.Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		return ResponsePack(InvalidTransaction, "")
	}
	txn, height, err := Store.GetTransaction(hash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	if false {
		w := new(bytes.Buffer)
		txn.Serialize(w)
		return ResponsePack(Success, common.BytesToHexString(w.Bytes()))
	}
	bHash, err := Chain.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	header, err := Chain.GetHeader(bHash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}

	return ResponsePack(Success, GetTransactionContextInfo(header, txn))
}

func GetExistWithdrawTransactions(param Params) map[string]interface{} {
	txList, ok := param.ArrayString("txs")
	if !ok {
		return ResponsePack(InvalidParams, "txs not found")
	}

	var resultTxHashes []string
	for _, txHash := range txList {
		txHashBytes, err := common.HexStringToBytes(txHash)
		if err != nil {
			return ResponsePack(InvalidParams, "")
		}
		hash, err := common.Uint256FromBytes(txHashBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "")
		}
		inStore := Store.IsSidechainTxHashDuplicate(*hash)
		inTxPool := TxMemPool.IsDuplicateSidechainTx(*hash)
		if inTxPool || inStore {
			resultTxHashes = append(resultTxHashes, txHash)
		}
	}

	return ResponsePack(Success, resultTxHashes)
}

func GetExistSideChainReturnDepositTransactions(param Params) map[string]interface{} {
	txList, ok := param.ArrayString("txs")
	if !ok {
		return ResponsePack(InvalidParams, "txs not found")
	}

	var resultTxHashes []string
	for _, txHash := range txList {
		txHashBytes, err := common.HexStringToBytes(txHash)
		if err != nil {
			return ResponsePack(InvalidParams, "")
		}
		hash, err := common.Uint256FromBytes(txHashBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "")
		}
		inStore := Store.IsSidechainReturnDepositTxHashDuplicate(*hash)
		inTxPool := TxMemPool.IsDuplicateSidechainReturnDepositTx(*hash)
		if inTxPool || inStore {
			resultTxHashes = append(resultTxHashes, txHash)
		}
	}

	return ResponsePack(Success, resultTxHashes)
}

// single producer info
type RPCProducerInfo struct {
	OwnerPublicKey string `json:"ownerpublickey"`
	NodePublicKey  string `json:"nodepublickey"`
	Nickname       string `json:"nickname"`
	Url            string `json:"url"`
	Location       uint64 `json:"location"`
	StakeUntil     uint32 `json:"stakeuntil"`
	Active         bool   `json:"active"`
	Votes          string `json:"votes"`
	DPoSV2Votes    string `json:"dposv2votes"`
	State          string `json:"state"`
	OnDuty         string `json:"onduty"`
	Identity       string `json:"identity"`
	RegisterHeight uint32 `json:"registerheight"`
	CancelHeight   uint32 `json:"cancelheight"`
	InactiveHeight uint32 `json:"inactiveheight"`
	IllegalHeight  uint32 `json:"illegalheight"`
	Index          uint64 `json:"index"`
}

// a group producer info include TotalDPoSV1Votes and producer count
type RPCProducersInfo struct {
	ProducerInfoSlice []RPCProducerInfo `json:"producers"`
	TotalVotes        string            `json:"totalvotes"`
	TotalDPoSV1Votes  string            `json:"totaldposv1votes"`
	TotalDPoSV2Votes  string            `json:"totaldposv2votes"`
	TotalCounts       uint64            `json:"totalcounts"`
}

// single cr candidate info
type RPCCRCandidateInfo struct {
	Code           string `json:"code"`
	CID            string `json:"cid"`
	DID            string `json:"did"`
	NickName       string `json:"nickname"`
	Url            string `json:"url"`
	Location       uint64 `json:"location"`
	State          string `json:"state"`
	Votes          string `json:"votes"`
	RegisterHeight uint32 `json:"registerheight"`
	CancelHeight   uint32 `json:"cancelheight"`

	Index uint64 `json:"index"`
}

// a group cr candidate info include TotalDPoSV1Votes and candidate count
type RPCCRCandidatesInfo struct {
	CRCandidateInfoSlice []RPCCRCandidateInfo `json:"crcandidatesinfo"`
	TotalVotes           string               `json:"totalvotes"`
	TotalCounts          uint64               `json:"totalcounts"`
}

type RPCSecretaryGeneralInfo struct {
	SecretaryGeneral string `json:"secretarygeneral"`
}

type RPCCommitteeCanUseAmount struct {
	CommitteeCanUseAmount string `json:"committeecanuseamount"`
}

type RPCCommitteeAssetInfo struct {
	CommitteeAssetBalance    string `json:"CommitteeAssetBalance"`
	CommitteeExpensesBalance string `json:"CommitteeExpensesBalance"`
	CommitteeCanUseAmount    string `json:"CommitteeCanUseAmount"`
	MaxProposalBudgetAmount  string `json:"MaxProposalBudgetAmount"`
}

type RPCCRRelatedStage struct {
	OnDuty              bool   `json:"onduty"`
	OnDutyStartHeight   uint32 `json:"ondutystartheight"`
	OnDutyEndHeight     uint32 `json:"ondutyendheight"`
	CurrentSession      uint32 `json:"currentsession"`
	InVoting            bool   `json:"invoting"`
	VotingStartHeight   uint32 `json:"votingstartheight"`
	VotingEndHeight     uint32 `json:"votingendheight"`
	InClaiming          bool   `json:"inClaiming"`
	ClaimingStartHeight uint32 `json:"claimingStartHeight"`
	ClaimingEndHeight   uint32 `json:"claimingEndHeight"`
}

// single cr member info
type RPCCRMemberInfo struct {
	Code             string `json:"code"`
	CID              string `json:"cid"`
	DID              string `json:"did"`
	DPOSPublicKey    string `json:"dpospublickey"`
	NickName         string `json:"nickname"`
	Url              string `json:"url"`
	Location         uint64 `json:"location"`
	ImpeachmentVotes string `json:"impeachmentvotes"`
	DepositAmount    string `json:"depositamout"`
	DepositAddress   string `json:"depositaddress"`
	Penalty          string `json:"penalty"`
	State            string `json:"state"`
	Index            uint64 `json:"index"`
}

// a group cr member info  include cr member count
type RPCCRMembersInfo struct {
	CRMemberInfoSlice []RPCCRMemberInfo `json:"crmembersinfo"`
	TotalCounts       uint64            `json:"totalcounts"`
}

type RPCCRCouncilMemberInfo struct {
	Code           string `json:"code"`
	CID            string `json:"cid"`
	DID            string `json:"did"`
	NickName       string `json:"nickname"`
	Url            string `json:"url"`
	Location       uint64 `json:"location"`
	DepositAddress string `json:"depositaddress"`
	Index          uint64 `json:"index"`
}

// a group cr member info  include cr member count
type RPCCRCouncilMembersInfo struct {
	CRMemberInfoSlice []RPCCRCouncilMemberInfo `json:"crmembersinfo"`
	TotalCounts       uint64                   `json:"totalcounts"`
}

type RPCCRMemberPerfornamce struct {
	Title           string `json:"title"`
	ProposalHash    string `json:"proposalHash"`
	ProposalState   string `json:"proposalState"`
	Opinion         string `json:"opinion"`
	OpinionHash     string `json:"opinionHash"`
	OpinionMessage  string `json:"opinionMessage"`
	ReviewHeight    uint32 `json:"reviewHeight"`
	ReviewTimestamp uint32 `json:"reviewTimestamp"`
}

// the CR Council Member's information including the performance
type RPCCRMemberInfoV2 struct {
	DID                     string                   `json:"did"`
	CID                     string                   `json:"cid"`
	Code                    string                   `json:"code"`
	NickName                string                   `json:"nickname"`
	Url                     string                   `json:"url"`
	Location                uint64                   `json:"location"`
	DepositAddress          string                   `json:"depositaddress"`
	DepositAmount           string                   `json:"depositamout"`
	DPOSPublicKey           string                   `json:"dpospublickey"`
	ImpeachmentVotes        string                   `json:"impeachmentvotes"`
	ImpeachmentThroughVotes string                   `json:"impeachmentThroughVotes"`
	Penalty                 string                   `json:"penalty"`
	Term                    []uint32                 `json:"term"`
	Performance             []RPCCRMemberPerfornamce `json:"performance"`
	State                   string                   `json:"state"`
}

// single CR Term Info
type RPCCRTermInfo struct {
	Index       uint64 `json:"index"`
	State       string `json:"state"`
	StartHeight uint32 `json:"startHeight"`
	EndHeight   uint32 `json:"endHeight"`
}

// a group CR Term Info  include CR term count
type RPCCRTermsInfo struct {
	CRTermInfoSlice []RPCCRTermInfo `json:"crtermsinfo"`
	TotalCounts     uint64          `json:"totalcounts"`
}

type RPCProposalBaseState struct {
	Status             string            `json:"status"`
	ProposalHash       string            `json:"proposalhash"`
	ProposalTitle      string            `json:"proposalTitle"`
	ProposalType       string            `json:"proposalType"`
	TxHash             string            `json:"txhash"`
	CRVotes            map[string]string `json:"crvotes"`
	VotersRejectAmount string            `json:"votersrejectamount"`
	RegisterHeight     uint32            `json:"registerHeight"`
	RegisterTimestamp  uint32            `json:"registerTimestamp"`
	TerminatedHeight   uint32            `json:"terminatedheight"`
	TrackingCount      uint8             `json:"trackingcount"`
	ProposalOwner      string            `json:"proposalowner"`
	ProposerDID        string            `json:"proposerDID"`
	Index              uint64            `json:"index"`
}

type RPCCRProposalBaseStateInfo struct {
	ProposalBaseStates []RPCProposalBaseState `json:"proposalbasestates"`
	TotalCounts        uint64                 `json:"totalcounts"`
}

type RPCCRCProposal struct {
	ProposalType       string                 `json:"proposaltype"`
	CategoryData       string                 `json:"categorydata"`
	OwnerPublicKey     string                 `json:"ownerpublickey"`
	CRCouncilMemberDID string                 `json:"crcouncilmemberdid"`
	DraftHash          string                 `json:"drafthash"`
	Recipient          string                 `json:"recipient"`
	Budgets            []BudgetInfo           `json:"budgets"`
	Milestone          []CRCProposalMilestone `json:"milestone"`
}

type RPCProposalState struct {
	Title                   string                          `json:"title"`
	Status                  string                          `json:"status"`
	Proposal                interface{}                     `json:"proposal"`
	ProposalHash            string                          `json:"proposalhash"`
	TxHash                  string                          `json:"txhash"`
	CRVotes                 map[string]string               `json:"crvotes"`
	CROpinions              []CRCProposalReviewOpinion      `json:"crOpinions"`
	VotersRejectAmount      string                          `json:"votersrejectamount"`
	RegisterHeight          uint32                          `json:"registerheight"`
	RegisterTimestamp       uint32                          `json:"registerTimestamp"`
	Abstract                string                          `json:"abstract"`
	Motivation              string                          `json:"motivation"`
	Goal                    string                          `json:"goal"`
	ImplementationTeamSlice []CRCProposalImplementationTeam `json:"implementationTeam"`
	PlanStatement           string                          `json:"planStatement"`
	BudgetStatement         string                          `json:"budgetStatement"`
	TerminatedHeight        uint32                          `json:"terminatedheight"`
	TrackingCount           uint8                           `json:"trackingcount"`
	ProposalOwner           string                          `json:"proposalowner"`
	ProposerDID             string                          `json:"proposerDID"`
	AvailableAmount         string                          `json:"availableamount"`
}

type RPCChangeProposalOwnerProposal struct {
	ProposalType       string `json:"proposaltype"`
	CategoryData       string `json:"categorydata"`
	OwnerPublicKey     string `json:"ownerpublickey"`
	DraftHash          string `json:"drafthash"`
	TargetProposalHash string `json:"targetproposalhash"`
	NewRecipient       string `json:"newrecipient"`
	NewOwnerPublicKey  string `json:"newownerpublickey"`
	CRCouncilMemberDID string `json:"crcouncilmemberdid"`
}

type RPCCloseProposal struct {
	ProposalType       string `json:"proposaltype"`
	CategoryData       string `json:"categorydata"`
	OwnerPublicKey     string `json:"ownerpublickey"`
	DraftHash          string `json:"drafthash"`
	TargetProposalHash string `json:"targetproposalhash"`
	CRCouncilMemberDID string `json:"crcouncilmemberdid"`
}

type RPCReservedCustomIDProposal struct {
	ProposalType         string   `json:"proposaltype"`
	CategoryData         string   `json:"categorydata"`
	OwnerPublicKey       string   `json:"ownerpublickey"`
	DraftHash            string   `json:"drafthash"`
	ReservedCustomIDList []string `json:"reservedcustomidlist"`
	CRCouncilMemberDID   string   `json:"crcouncilmemberdid"`
}

type RPCReceiveCustomIDProposal struct {
	ProposalType        string   `json:"proposaltype"`
	CategoryData        string   `json:"categorydata"`
	OwnerPublicKey      string   `json:"ownerpublickey"`
	DraftHash           string   `json:"drafthash"`
	ReceiveCustomIDList []string `json:"receivecustomidlist"`
	ReceiverDID         string   `json:"receiverdid"`
	CRCouncilMemberDID  string   `json:"crcouncilmemberdid"`
}

type RPCChangeCustomIDFeeProposal struct {
	ProposalType       string `json:"proposaltype"`
	CategoryData       string `json:"categorydata"`
	OwnerPublicKey     string `json:"ownerpublickey"`
	DraftHash          string `json:"drafthash"`
	Fee                int64  `json:"fee"`
	EIDEffectiveHeight uint32 `json:"eideffectiveheight"`
	CRCouncilMemberDID string `json:"crcouncilmemberdid"`
}

type RPCSecretaryGeneralProposal struct {
	ProposalType              string `json:"proposaltype"`
	CategoryData              string `json:"categorydata"`
	OwnerPublicKey            string `json:"ownerpublickey"`
	DraftHash                 string `json:"drafthash"`
	SecretaryGeneralPublicKey string `json:"secretarygeneralpublickey"`
	SecretaryGeneralDID       string `json:"secretarygeneraldid"`
	CRCouncilMemberDID        string `json:"crcouncilmemberdid"`
}

type RegisterSideChainInfo struct {
	SideChainName   string `json:"sidechainname"`
	MagicNumber     uint32 `json:"magic"`
	GenesisHash     string `json:"genesishash"`
	ExchangeRate    string `json:"exchangerate"`
	EffectiveHeight uint32 `json:"effectiveheight"`
	ResourcePath    string `json:"resourcepath"`
}

type RPCRegisterSideChainProposal struct {
	ProposalType       string                `json:"proposaltype"`
	CategoryData       string                `json:"categorydata"`
	OwnerPublicKey     string                `json:"ownerpublickey"`
	DraftHash          string                `json:"drafthash"`
	SideChainInfo      RegisterSideChainInfo `json:"sidechaininfo"`
	CRCouncilMemberDID string                `json:"crcouncilmemberdid"`
}

type RPCCRProposalStateInfo struct {
	ProposalState RPCProposalState `json:"proposalstate"`
}

type RPCDposV2RewardInfo struct {
	Address   string `json:"address"`
	Claimable string `json:"claimable"`
	Claiming  string `json:"claiming"`
	Claimed   string `json:"claimed"`
}

type RPCDPosV2Info struct {
	ConsensusAlgorithm       string `json:"consensusalgorithm"`
	Height                   uint32 `json:"height"`
	DPoSV2ActiveHeight       uint32 `json:"dposv2activeheight"`
	DPoSV2TransitStartHeight uint32 `json:"dposv2transitstartheight"`
}

func DposV2RewardInfo(param Params) map[string]interface{} {
	addr, ok := param.String("address")
	if ok {
		// need to get claimable reward from Standard or Multi-sign address,
		// also need to get claimable reward from Stake address.
		address, err := common.Uint168FromAddress(addr)
		if err != nil {
			return ResponsePack(InternalError, "invalid address")
		}
		// check prefix, if the prefix is not PrefixDPoSV2, we need to change it
		// to PrefixDPoSV2.
		stakeAddress := addr
		if address[0] != byte(contract.PrefixDPoSV2) {
			address[0] = byte(contract.PrefixDPoSV2)
			// create stake address from Standard or Multi-sign address.
			stakeAddress, err = address.ToAddress()
			if err != nil {
				return ResponsePack(InternalError, "invalid stake address")
			}
		}

		claimable := Chain.GetState().DPoSV2RewardInfo[stakeAddress]
		claiming := Chain.GetState().DposV2RewardClaimingInfo[stakeAddress]
		claimed := Chain.GetState().DposV2RewardClaimedInfo[stakeAddress]
		result := RPCDposV2RewardInfo{
			Address:   addr,
			Claimable: claimable.String(),
			Claiming:  claiming.String(),
			Claimed:   claimed.String(),
		}
		return ResponsePack(Success, result)
	} else {
		var result []RPCDposV2RewardInfo
		dposV2RewardInfo := Chain.GetState().DPoSV2RewardInfo
		for addr, value := range dposV2RewardInfo {
			result = append(result, RPCDposV2RewardInfo{
				Address:   addr,
				Claimable: value.String(),
				Claiming:  Chain.GetState().DposV2RewardClaimingInfo[addr].String(),
				Claimed:   Chain.GetState().DposV2RewardClaimedInfo[addr].String(),
			})
		}

		return ResponsePack(Success, result)
	}
}

func GetDPosV2Info(param Params) map[string]interface{} {
	consensusAlgorithm := Chain.GetState().GetConsensusAlgorithm().String()
	currentHeight := Store.GetHeight()
	dposV2ActiveHeight := Chain.GetState().DPoSV2ActiveHeight

	if currentHeight >= dposV2ActiveHeight {
		consensusAlgorithm = "DPoS 2.0"
	}
	dposV2TransitStartHeight := config.Parameters.DPoSV2StartHeight

	result := &RPCDPosV2Info{
		ConsensusAlgorithm:       consensusAlgorithm,
		Height:                   currentHeight,
		DPoSV2ActiveHeight:       dposV2ActiveHeight,
		DPoSV2TransitStartHeight: dposV2TransitStartHeight,
	}
	return ResponsePack(Success, result)
}

func ListProducers(param Params) map[string]interface{} {
	start, _ := param.Int("start")
	if start < 0 {
		start = 0
	}
	limit, ok := param.Int("limit")
	if !ok {
		limit = -1
	}
	s, ok := param.String("state")
	if ok {
		s = strings.ToLower(s)
	}
	identity, ok := param.String("identity")
	if ok {
		identity = strings.ToUpper(identity)
	} else {
		identity = "ALL"
	}
	var producers []*state.Producer
	switch s {
	case "all":
		ps := Chain.GetState().GetAllProducers()
		for i, _ := range ps {
			producers = append(producers, &ps[i])
		}
	case "pending":
		producers = Chain.GetState().GetPendingProducers()
	case "active":
		producers = Chain.GetState().GetActiveProducers()
	case "inactive":
		producers = Chain.GetState().GetInactiveProducers()
	case "canceled":
		producers = Chain.GetState().GetCanceledProducers()
	case "illegal":
		producers = Chain.GetState().GetIllegalProducers()
	case "returned":
		producers = Chain.GetState().GetReturnedDepositProducers()
	default:
		producers = Chain.GetState().GetProducers()
	}

	// Filter Producers by identity
	switch identity {
	case "V1":
		i := 0
		for _, p := range producers {
			if strings.Contains(p.Identity().String(), "V1") {
				producers[i] = p
				i++
			}
		}
		producers = producers[:i]
		sort.Slice(producers, func(i, j int) bool {
			if producers[i].Votes() == producers[j].Votes() {
				return bytes.Compare(producers[i].NodePublicKey(),
					producers[j].NodePublicKey()) < 0
			}
			return producers[i].Votes() > producers[j].Votes()
		})
	case "V2":
		i := 0
		for _, p := range producers {
			if strings.Contains(p.Identity().String(), "V2") {
				producers[i] = p
				i++
			}
		}
		producers = producers[:i]
		sort.Slice(producers, func(i, j int) bool {
			if producers[i].GetTotalDPoSV2VoteRights() == producers[j].GetTotalDPoSV2VoteRights() {
				return bytes.Compare(producers[i].NodePublicKey(),
					producers[j].NodePublicKey()) < 0
			}
			return producers[i].GetTotalDPoSV2VoteRights() > producers[j].GetTotalDPoSV2VoteRights()
		})
	case "ALL":
		sort.Slice(producers, func(i, j int) bool {
			if producers[i].GetTotalDPoSV2VoteRights() == producers[j].GetTotalDPoSV2VoteRights() {
				return bytes.Compare(producers[i].NodePublicKey(),
					producers[j].NodePublicKey()) < 0
			}
			return producers[i].GetTotalDPoSV2VoteRights() > producers[j].GetTotalDPoSV2VoteRights()
		})
	}

	var producerInfoSlice []RPCProducerInfo
	var totalVotes, totalDPoSV2Votes common.Fixed64
	for i, p := range producers {
		totalVotes += p.Votes()
		dposV2Votes := common.Fixed64(p.GetTotalDPoSV2VoteRights())

		totalDPoSV2Votes += dposV2Votes
		var onDutyState string
		switch p.State() {
		case state.Active:
			if dposV2Votes < ChainParams.DPoSV2EffectiveVotes {
				onDutyState = "Candidate"
			} else {
				onDutyState = "Valid"
			}
		default:
			onDutyState = "Invalid"

		}

		producerInfo := RPCProducerInfo{
			OwnerPublicKey: hex.EncodeToString(p.Info().OwnerKey),
			NodePublicKey:  hex.EncodeToString(p.Info().NodePublicKey),
			Nickname:       p.Info().NickName,
			Url:            p.Info().Url,
			Location:       p.Info().Location,
			StakeUntil:     p.Info().StakeUntil,
			Active:         p.State() == state.Active,
			Votes:          p.Votes().String(),
			DPoSV2Votes:    dposV2Votes.String(),
			State:          p.State().String(),
			OnDuty:         onDutyState,
			Identity:       p.Identity().String(),
			RegisterHeight: p.RegisterHeight(),
			CancelHeight:   p.CancelHeight(),
			InactiveHeight: p.InactiveSince(),
			IllegalHeight:  p.IllegalHeight(),
			Index:          uint64(i),
		}
		producerInfoSlice = append(producerInfoSlice, producerInfo)
	}

	count := int64(len(producers))
	if limit < 0 {
		limit = count
	}
	var rsProducerInfoSlice []RPCProducerInfo
	if start < count {
		end := start
		if start+limit <= count {
			end = start + limit
		} else {
			end = count
		}
		rsProducerInfoSlice = append(rsProducerInfoSlice, producerInfoSlice[start:end]...)
	}

	result := &RPCProducersInfo{
		ProducerInfoSlice: rsProducerInfoSlice,
		TotalVotes:        totalVotes.String(),
		TotalDPoSV1Votes:  totalVotes.String(),
		TotalDPoSV2Votes:  totalDPoSV2Votes.String(),
		TotalCounts:       uint64(count),
	}

	return ResponsePack(Success, result)
}

func GetSecretaryGeneral(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()

	result := &RPCSecretaryGeneralInfo{
		SecretaryGeneral: crCommittee.GetProposalManager().SecretaryGeneralPublicKey,
	}
	return ResponsePack(Success, result)
}

func GetCommitteeCanUseAmount(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()

	result := &RPCCommitteeCanUseAmount{
		CommitteeCanUseAmount: crCommittee.GetCommitteeCanUseAmount().String(),
	}
	return ResponsePack(Success, result)
}

func GetCommitteeAssetInfo(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()
	maxProposalBudgetAmount := (crCommittee.CRCCurrentStageAmount -
		crCommittee.CommitteeUsedAmount) * blockchain.CRCProposalBudgetsPercentage / 100

	result := &RPCCommitteeAssetInfo{
		CommitteeAssetBalance:    crCommittee.CRCFoundationBalance.String(),
		CommitteeExpensesBalance: crCommittee.CRCCommitteeBalance.String(),
		CommitteeCanUseAmount:    crCommittee.GetCommitteeCanUseAmount().String(),
		MaxProposalBudgetAmount:  maxProposalBudgetAmount.String(),
	}
	return ResponsePack(Success, result)
}

func GetCRRelatedStage(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	isOnDuty := cm.IsInElectionPeriod()
	currentHeight := Chain.GetHeight()
	isInVoting := cm.IsInVotingPeriod(currentHeight)

	var ondutyStartHeight, ondutyEndHeight uint32
	var currentSession uint64
	if isOnDuty {
		ondutyStartHeight = cm.GetCROnDutyStartHeight()
		ondutyEndHeight = ondutyStartHeight + cm.GetCROnDutyPeriod()
	}

	var votingStartHeight, votingEndHeight uint32
	votingStartHeight = cm.GetCRVotingStartHeight()
	votingEndHeight = votingStartHeight + cm.GetCRVotingPeriod()

	claimingStartHeight := votingEndHeight
	claimingEndHeight := claimingStartHeight + ChainParams.CRConfiguration.CRClaimPeriod

	isInClaiming := false
	if claimingStartHeight <= currentHeight && currentHeight <= claimingEndHeight {
		isInClaiming = true
	}

	if !isInVoting && !isInClaiming {
		votingStartHeight = cm.GetNextVotingStartHeight(currentHeight)
		votingEndHeight = votingStartHeight + cm.GetCRVotingPeriod()
		claimingStartHeight = votingEndHeight
		claimingEndHeight = claimingStartHeight + ChainParams.CRConfiguration.CRClaimPeriod
	}

	currentSession = cm.GetCurrentSession()

	result := &RPCCRRelatedStage{
		OnDuty:              isOnDuty,
		OnDutyStartHeight:   ondutyStartHeight,
		OnDutyEndHeight:     ondutyEndHeight,
		CurrentSession:      uint32(currentSession),
		InVoting:            isInVoting,
		VotingStartHeight:   votingStartHeight,
		VotingEndHeight:     votingEndHeight,
		InClaiming:          isInClaiming,
		ClaimingStartHeight: claimingStartHeight,
		ClaimingEndHeight:   claimingEndHeight,
	}
	return ResponsePack(Success, result)
}

// list cr candidates according to ( state , start and limit)
func ListCRCandidates(param Params) map[string]interface{} {
	start, _ := param.Int("start")
	if start < 0 {
		start = 0
	}
	limit, ok := param.Int("limit")
	if !ok {
		limit = -1
	}
	s, ok := param.String("state")
	if ok {
		s = strings.ToLower(s)
	}
	var candidates []*crstate.Candidate
	crCommittee := Chain.GetCRCommittee()
	switch s {
	case "all":
		candidates = crCommittee.GetAllCandidates()
	case "pending":
		candidates = crCommittee.GetCandidates(crstate.Pending)
	case "active":
		candidates = crCommittee.GetCandidates(crstate.Active)
	case "canceled":
		candidates = crCommittee.GetCandidates(crstate.Canceled)
	case "returned":
		candidates = crCommittee.GetCandidates(crstate.Returned)
	default:
		candidates = crCommittee.GetCandidates(crstate.Pending)
		candidates = append(candidates, crCommittee.GetCandidates(crstate.Active)...)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Votes == candidates[j].Votes {
			iCRInfo := candidates[i].Info
			jCRInfo := candidates[j].Info
			return iCRInfo.GetCodeHash().Compare(jCRInfo.GetCodeHash()) < 0
		}
		return candidates[i].Votes > candidates[j].Votes
	})

	var candidateInfoSlice []RPCCRCandidateInfo
	var totalVotes common.Fixed64
	for i, c := range candidates {
		totalVotes += c.Votes
		cidAddress, _ := c.Info.CID.ToAddress()
		var didAddress string
		if !c.Info.DID.IsEqual(emptyHash) {
			didAddress, _ = c.Info.DID.ToAddress()
		}
		candidateInfo := RPCCRCandidateInfo{
			Code:           hex.EncodeToString(c.Info.Code),
			CID:            cidAddress,
			DID:            didAddress,
			NickName:       c.Info.NickName,
			Url:            c.Info.Url,
			Location:       c.Info.Location,
			State:          c.State.String(),
			Votes:          c.Votes.String(),
			RegisterHeight: c.RegisterHeight,
			CancelHeight:   c.CancelHeight,
			Index:          uint64(i),
		}
		candidateInfoSlice = append(candidateInfoSlice, candidateInfo)
	}

	count := int64(len(candidates))
	if limit < 0 {
		limit = count
	}
	var rSCandidateInfoSlice []RPCCRCandidateInfo
	if start < count {
		end := start
		if start+limit <= count {
			end = start + limit
		} else {
			end = count
		}
		rSCandidateInfoSlice = append(rSCandidateInfoSlice, candidateInfoSlice[start:end]...)
	}

	result := &RPCCRCandidatesInfo{
		CRCandidateInfoSlice: rSCandidateInfoSlice,
		TotalVotes:           totalVotes.String(),
		TotalCounts:          uint64(count),
	}

	return ResponsePack(Success, result)
}

// list current crs according to (state)
func ListCurrentCRs(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	var crMembers []*crstate.CRMember
	if cm.IsInElectionPeriod() {
		crMembers = cm.GetCurrentMembers()
		sort.Slice(crMembers, func(i, j int) bool {
			return crMembers[i].Info.NickName < crMembers[j].Info.NickName
		})
	}

	var rsCRMemberInfoSlice []RPCCRMemberInfo
	for i, cr := range crMembers {
		cidAddress, _ := cr.Info.CID.ToAddress()
		var didAddress string
		if !cr.Info.DID.IsEqual(emptyHash) {
			didAddress, _ = cr.Info.DID.ToAddress()
		}
		depositAddr, _ := cr.DepositHash.ToAddress()
		memberInfo := RPCCRMemberInfo{
			Code:             hex.EncodeToString(cr.Info.Code),
			CID:              cidAddress,
			DID:              didAddress,
			DPOSPublicKey:    hex.EncodeToString(cr.DPOSPublicKey),
			NickName:         cr.Info.NickName,
			Url:              cr.Info.Url,
			Location:         cr.Info.Location,
			ImpeachmentVotes: cr.ImpeachmentVotes.String(),
			DepositAmount:    cm.GetAvailableDepositAmount(cr.Info.CID).String(),
			DepositAddress:   depositAddr,
			Penalty:          cm.GetPenalty(cr.Info.CID).String(),
			Index:            uint64(i),
			State:            cr.MemberState.String(),
		}
		rsCRMemberInfoSlice = append(rsCRMemberInfoSlice, memberInfo)
	}

	count := int64(len(crMembers))

	result := &RPCCRMembersInfo{
		CRMemberInfoSlice: rsCRMemberInfoSlice,
		TotalCounts:       uint64(count),
	}

	return ResponsePack(Success, result)
}

// list next crs according to (state)
func ListNextCRs(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	var crMembers []*crstate.CRMember
	crMembers = cm.GetNextMembers()
	sort.Slice(crMembers, func(i, j int) bool {
		return crMembers[i].Info.GetCodeHash().Compare(
			crMembers[j].Info.GetCodeHash()) < 0
	})

	var rsCRMemberInfoSlice []RPCCRMemberInfo
	for i, cr := range crMembers {
		cidAddress, _ := cr.Info.CID.ToAddress()
		var didAddress string
		if !cr.Info.DID.IsEqual(emptyHash) {
			didAddress, _ = cr.Info.DID.ToAddress()
		}
		depositAddr, _ := cr.DepositHash.ToAddress()
		memberInfo := RPCCRMemberInfo{
			Code:             hex.EncodeToString(cr.Info.Code),
			CID:              cidAddress,
			DID:              didAddress,
			DPOSPublicKey:    hex.EncodeToString(cr.DPOSPublicKey),
			NickName:         cr.Info.NickName,
			Url:              cr.Info.Url,
			Location:         cr.Info.Location,
			ImpeachmentVotes: cr.ImpeachmentVotes.String(),
			DepositAmount:    cm.GetAvailableDepositAmount(cr.Info.CID).String(),
			DepositAddress:   depositAddr,
			Penalty:          cm.GetPenalty(cr.Info.CID).String(),
			Index:            uint64(i),
			State:            cr.MemberState.String(),
		}
		rsCRMemberInfoSlice = append(rsCRMemberInfoSlice, memberInfo)
	}

	count := int64(len(crMembers))

	result := &RPCCRMembersInfo{
		CRMemberInfoSlice: rsCRMemberInfoSlice,
		TotalCounts:       uint64(count),
	}

	return ResponsePack(Success, result)
}

// list CR Terms
func ListCRTerms(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	crTerms := cm.GetCRTerms()
	var rsCRTermInfoSlice []RPCCRTermInfo

	currentTerm := cm.GetCurrentSession()

	for i, t := range crTerms {
		s := "history"
		if uint64(i) == currentTerm {
			s = "current"
		}

		termInfo := RPCCRTermInfo{
			Index:       uint64(i),
			State:       s,
			StartHeight: t.StartHeight,
			EndHeight:   t.EndHeight,
		}
		rsCRTermInfoSlice = append(rsCRTermInfoSlice, termInfo)
	}
	sort.Slice(rsCRTermInfoSlice, func(i, j int) bool {
		return rsCRTermInfoSlice[i].Index < rsCRTermInfoSlice[j].Index
	})

	count := int64(len(crTerms))

	result := &RPCCRTermsInfo{
		CRTermInfoSlice: rsCRTermInfoSlice,
		TotalCounts:     uint64(count),
	}

	return ResponsePack(Success, result)
}

// Get CR Members by Term
func ListCRMembers(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	crTerm, ok := param.Uint("term")
	if !ok {
		crTerm = uint32(cm.GetCurrentSession())
	}

	var crMembers []payload.CRMemberInfo
	crMembers = cm.GetCRCouncils()[crTerm]

	var rsCRMemberInfoSlice []RPCCRCouncilMemberInfo
	for _, cr := range crMembers {
		cidAddress, _ := cr.CID.ToAddress()
		var didAddress string
		if !cr.DID.IsEqual(emptyHash) {
			didAddress, _ = cr.DID.ToAddress()
		}
		depositAddr, _ := cr.DepositHash.ToAddress()
		memberInfo := RPCCRCouncilMemberInfo{
			Code:           hex.EncodeToString(cr.Code),
			CID:            cidAddress,
			DID:            didAddress,
			NickName:       cr.NickName,
			Url:            cr.Url,
			Location:       cr.Location,
			DepositAddress: depositAddr,
			Index:          0,
		}
		rsCRMemberInfoSlice = append(rsCRMemberInfoSlice, memberInfo)
	}
	sort.Slice(rsCRMemberInfoSlice, func(i, j int) bool {
		return rsCRMemberInfoSlice[i].NickName < rsCRMemberInfoSlice[j].NickName
	})
	for i, _ := range rsCRMemberInfoSlice {
		rsCRMemberInfoSlice[i].Index = uint64(i)
	}

	count := int64(len(crMembers))

	result := &RPCCRCouncilMembersInfo{
		CRMemberInfoSlice: rsCRMemberInfoSlice,
		TotalCounts:       uint64(count),
	}

	return ResponsePack(Success, result)
}

func GetCRMember(param Params) map[string]interface{} {
	cm := Chain.GetCRCommittee()
	var did *common.Uint168
	id, hasID := param.String("id")
	if hasID {
		programHash, err := common.Uint168FromAddress(id)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid id to programHash")
		}
		_did, exist := cm.GetDIDByID(*programHash)
		if !exist {
			return ResponsePack(InvalidParams, "invalid id")
		}
		did = _did
	} else {
		_did, hasDID := param.String("did")
		if !hasDID {
			return ResponsePack(InvalidParams, "")
		}
		programHash, err := common.Uint168FromAddress(_did)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid did to programHash")
		}
		did = programHash
	}

	members := cm.GetCRMembersInfo()
	cr, ok := members[*did]
	if !ok {
		return ResponsePack(InvalidParams, "invalid did")
	}

	cid := cr.Info.CID
	didAddress, _ := cr.Info.DID.ToAddress()
	cidAddress, _ := cid.ToAddress()
	depositAddr, _ := cr.Info.DepositHash.ToAddress()

	rsCRMemberPerfornamce := pasarCRMemberPerformance(cr.ProposalReviews)

	rpcMemberInfo := RPCCRMemberInfoV2{
		DID:                     didAddress,
		CID:                     cidAddress,
		Code:                    hex.EncodeToString(cr.Info.Code),
		NickName:                cr.Info.NickName,
		Url:                     cr.Info.Url,
		Location:                cr.Info.Location,
		DepositAddress:          depositAddr,
		DepositAmount:           cm.GetAvailableDepositAmount(cr.Info.CID).String(),
		DPOSPublicKey:           hex.EncodeToString(cm.GetDPOSPublicKeyByCID(cid)),
		ImpeachmentVotes:        cm.GetImpeachmentVotesByCID(cid).String(),
		ImpeachmentThroughVotes: cm.GetImpeachmentThroughVotes().String(),
		Penalty:                 cm.GetPenalty(cid).String(),
		Term:                    cr.Terms,
		Performance:             rsCRMemberPerfornamce,
		State:                   cr.MemberState.String(),
	}

	result := &rpcMemberInfo
	return ResponsePack(Success, result)
}

func pasarCRMemberPerformance(record map[common.Uint256]crstate.ProposalReviewRecord) []RPCCRMemberPerfornamce {
	crCommittee := Chain.GetCRCommittee()
	var proposalState *crstate.ProposalState
	var rsCRMemberPerfornamce []RPCCRMemberPerfornamce
	for k, v := range record {
		opinionHash := v.OpinionHash
		_messageData, _ := parseProposalDraftData(&opinionHash)
		opinionData, _ := _messageData.(CRCProposalMessageData)
		reviewHeight := v.ReviewHeight
		reviewTimestamp := getTimestampByHeight(reviewHeight)
		proposalState = crCommittee.GetProposal(k)
		p := RPCCRMemberPerfornamce{
			Title:           getProposalTitleByProposalHash(k),
			ProposalHash:    common.ToReversedString(k),
			ProposalState:   proposalState.Status.String(),
			Opinion:         v.Result.Name(),
			OpinionHash:     opinionHash.String(),
			OpinionMessage:  opinionData.Content,
			ReviewHeight:    reviewHeight,
			ReviewTimestamp: reviewTimestamp,
		}
		rsCRMemberPerfornamce = append(rsCRMemberPerfornamce, p)
		sort.Slice(rsCRMemberPerfornamce, func(i, j int) bool {
			return rsCRMemberPerfornamce[i].ReviewHeight >
				rsCRMemberPerfornamce[j].ReviewHeight
		})
	}
	return rsCRMemberPerfornamce
}

func ListCRProposalBaseState(param Params) map[string]interface{} {
	start, _ := param.Int("start")
	if start < 0 {
		start = 0
	}
	limit, ok := param.Int("limit")
	if !ok {
		limit = -1
	}
	s, ok := param.String("state")
	if ok {
		s = strings.ToLower(s)
	}
	order, ok := param.String("order")
	if ok {
		if order != "asc" && order != "desc" {
			return ResponsePack(InvalidParams, "invalid order")
		}
	} else {
		order = "desc"
	}
	var proposalMap crstate.ProposalsMap
	crCommittee := Chain.GetCRCommittee()
	switch s {
	case "all":
		proposalMap = crCommittee.GetAllProposals()
	case "registered":
		proposalMap = crCommittee.GetProposals(crstate.Registered)
	case "cragreed":
		proposalMap = crCommittee.GetProposals(crstate.CRAgreed)
	case "voteragreed":
		proposalMap = crCommittee.GetProposals(crstate.VoterAgreed)
	case "finished":
		proposalMap = crCommittee.GetProposals(crstate.Finished)
	case "crcanceled":
		proposalMap = crCommittee.GetProposals(crstate.CRCanceled)
	case "votercanceled":
		proposalMap = crCommittee.GetProposals(crstate.VoterCanceled)
	case "aborted":
		proposalMap = crCommittee.GetProposals(crstate.Aborted)
	case "terminated":
		proposalMap = crCommittee.GetProposals(crstate.Terminated)
	default:
		return ResponsePack(InvalidParams, "invalidate state")
	}

	var crVotes map[string]string
	var rpcProposalBaseStates []RPCProposalBaseState

	var index uint64
	for _, proposal := range proposalMap {
		crVotes = make(map[string]string)
		for k, v := range proposal.CRVotes {
			did, _ := k.ToAddress()
			crVotes[did] = v.Name()
		}
		proposalOwnerPubKey := proposal.ProposalOwner
		did, _ := blockchain.GetDiDFromPublicKey(proposalOwnerPubKey)
		proposerDID, _ := did.ToAddress()

		registerHeight := proposal.RegisterHeight
		_block, _ := Chain.GetBlockByHeight(registerHeight)
		registerTimestamp := _block.Timestamp

		draftHash := proposal.Proposal.DraftHash
		draftData, errorStr := parseProposalDraftData(&draftHash)
		var proposalData CRCProposalDraftData
		if errorStr == "" {
			proposalData, _ = draftData.(CRCProposalDraftData)
		}

		proposalTitle := ""
		if errorStr == "" {
			proposalTitle = proposalData.Title
		}

		rpcProposalBaseState := RPCProposalBaseState{
			Status:             proposal.Status.String(),
			ProposalHash:       common.ToReversedString(proposal.Proposal.Hash),
			ProposalTitle:      proposalTitle,
			ProposalType:       proposal.Proposal.ProposalType.Name(),
			TxHash:             common.ToReversedString(proposal.TxHash),
			CRVotes:            crVotes,
			VotersRejectAmount: proposal.VotersRejectAmount.String(),
			RegisterHeight:     registerHeight,
			RegisterTimestamp:  registerTimestamp,
			TrackingCount:      proposal.TrackingCount,
			TerminatedHeight:   proposal.TerminatedHeight,
			ProposalOwner:      hex.EncodeToString(proposalOwnerPubKey),
			ProposerDID:        proposerDID,
			Index:              index,
		}

		rpcProposalBaseStates = append(rpcProposalBaseStates, rpcProposalBaseState)
		index++
	}

	count := int64(len(rpcProposalBaseStates))
	if order == "desc" {
		sort.Slice(rpcProposalBaseStates, func(i, j int) bool {
			return rpcProposalBaseStates[i].
				RegisterHeight > rpcProposalBaseStates[j].RegisterHeight
		})
		for k := range rpcProposalBaseStates {
			rpcProposalBaseStates[k].Index = uint64(count) - 1 - uint64(k)
		}
	} else {
		sort.Slice(rpcProposalBaseStates, func(i, j int) bool {
			return rpcProposalBaseStates[i].
				RegisterHeight < rpcProposalBaseStates[j].RegisterHeight
		})
		for k := range rpcProposalBaseStates {
			rpcProposalBaseStates[k].Index = uint64(k)
		}
	}

	if limit < 0 {
		limit = count
	}
	var rRPCProposalBaseStates []RPCProposalBaseState
	if start < count {
		end := start
		if order == "desc" {
			end = count - start
			if start+limit > count {
				start = 0
			} else {
				start = count - start - limit
			}
		} else {
			if start+limit <= count {
				end = start + limit
			} else {
				end = count
			}
		}
		rRPCProposalBaseStates = append(rRPCProposalBaseStates, rpcProposalBaseStates[start:end]...)
	}

	result := &RPCCRProposalBaseStateInfo{
		ProposalBaseStates: rRPCProposalBaseStates,
		TotalCounts:        uint64(count),
	}

	return ResponsePack(Success, result)
}

func getProposalDraftData(draftHash *common.Uint256) ([]*zip.File, string) {
	draftData, _ := Chain.GetDB().GetProposalDraftDataByDraftHash(draftHash)
	if len(draftData) == 0 {
		return nil, "invalidate draft hash"
	}

	// Read ZipStream
	zipFiles, err := utils.ReadZipStream(draftData)
	if err != nil {
		return nil, "invalidate draftData"
	} else {
		return zipFiles, ""
	}
}

func parseProposalDraftData(draftHash *common.Uint256) (CRCDraftData, string) {
	zipFiles, err := getProposalDraftData(draftHash)
	if err != "" {
		return nil, "invalidate draftData"
	}
	// Read all the files from zip archive
	for _, zipFile := range zipFiles {
		fileName := zipFile.Name
		switch fileName {
		// Get ProposalData from proposal.json
		case "proposal.json":
			unzippedFileBytes, err := utils.ReadZipFile(zipFile)
			var proposalDraftData CRCProposalDraftData
			if err != nil {
				log.Error("Read Zip File Error:", err)
				return proposalDraftData, "invalidate draftData"
			}
			err = json.Unmarshal(unzippedFileBytes, &proposalDraftData)
			if err != nil {
				log.Error("Unmarshal Json File Error:", err)
				return proposalDraftData, "invalidate draftData"
			}
			return proposalDraftData, ""
		// Get Opinion or Message data from opinion.json or message.json
		case "opinion.json", "message.json":
			unzippedFileBytes, err := utils.ReadZipFile(zipFile)

			var messageData CRCProposalMessageData
			if err != nil {
				log.Error("Read Zip File Error:", err)
				return messageData, "invalidate opinionData"
			}
			err = json.Unmarshal(unzippedFileBytes, &messageData)
			if err != nil {
				log.Error("Unmarshal Json File Error:", err)
				return messageData, "invalidate opinionData"
			}
			return messageData, ""
		}
	}
	return nil, "invalidate draftHash"
}

func GetCRProposalState(param Params) map[string]interface{} {
	var proposalState *crstate.ProposalState
	crCommittee := Chain.GetCRCommittee()
	ProposalHashHexStr, ok := param.String("proposalhash")
	if ok {
		proposalHashBytes, err := common.FromReversedString(ProposalHashHexStr)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate proposalhash")
		}
		ProposalHash, err := common.Uint256FromBytes(proposalHashBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate proposalhash")
		}
		proposalState = crCommittee.GetProposal(*ProposalHash)
		if proposalState == nil {
			return ResponsePack(InvalidParams, "proposalhash not exist")
		}

	} else {
		DraftHashStr, ok := param.String("drafthash")

		if !ok {
			return ResponsePack(InvalidParams, "params at least one of proposalhash and DraftHash")
		}
		DraftHashStrBytes, err := common.FromReversedString(DraftHashStr)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate drafthash")
		}
		DraftHash, err := common.Uint256FromBytes(DraftHashStrBytes)

		if err != nil {
			return ResponsePack(InvalidParams, "invalidate drafthash")
		}
		proposalState = crCommittee.GetProposalByDraftHash(*DraftHash)
		if proposalState == nil {
			return ResponsePack(InvalidParams, "DraftHash not exist")
		}
	}

	proposalHash := proposalState.Proposal.Hash
	proposalDraftHash := proposalState.Proposal.DraftHash
	draftData, errorStr := parseProposalDraftData(&proposalDraftHash)
	var proposalData CRCProposalDraftData
	if errorStr == "" {
		proposalData, _ = draftData.(CRCProposalDraftData)
	}

	proposalTitle := ""
	if errorStr == "" {
		proposalTitle = proposalData.Title
	}

	budgetInDraftData := make(map[uint8]string)
	for _, budget := range proposalData.Budgets {
		budgetInDraftData[budget.Stage] = budget.PaymentCriteria
	}
	did, _ := blockchain.GetDiDFromPublicKey(proposalState.ProposalOwner)
	proposerDID, _ := did.ToAddress()

	implementationTeamSlice := proposalData.ImplementationTeam
	crVotes := make(map[string]string)
	for k, v := range proposalState.CRVotes {
		did, _ := k.ToAddress()
		crVotes[did] = v.Name()
	}
	crOpinions := make([]CRCProposalReviewOpinion, 0)
	for k, v := range proposalState.CROpinions {
		did, _ := k.ToAddress()
		_hash := common.ToReversedString(v)
		_messageData, errorStr := parseProposalDraftData(&v)
		var opinionData CRCProposalMessageData
		if errorStr == "" {
			opinionData, _ = _messageData.(CRCProposalMessageData)
		}
		crOpinions = append(crOpinions, CRCProposalReviewOpinion{
			DID:            did,
			OpinionHash:    _hash,
			OpinionMessage: opinionData.Content,
		})
	}

	registerHeight := proposalState.RegisterHeight
	_block, _ := Chain.GetBlockByHeight(registerHeight)
	registerTimestamp := _block.Timestamp
	rpcProposalState := RPCProposalState{
		Title:                   proposalTitle,
		Status:                  proposalState.Status.String(),
		ProposalHash:            common.ToReversedString(proposalHash),
		TxHash:                  common.ToReversedString(proposalState.TxHash),
		CRVotes:                 crVotes,
		CROpinions:              crOpinions,
		VotersRejectAmount:      proposalState.VotersRejectAmount.String(),
		RegisterHeight:          registerHeight,
		RegisterTimestamp:       registerTimestamp,
		Abstract:                proposalData.Abstract,
		Motivation:              proposalData.Motivation,
		Goal:                    proposalData.Goal,
		ImplementationTeamSlice: implementationTeamSlice,
		PlanStatement:           proposalData.PlanStatement,
		BudgetStatement:         proposalData.BudgetStatement,
		TrackingCount:           proposalState.TrackingCount,
		TerminatedHeight:        proposalState.TerminatedHeight,
		ProposalOwner:           hex.EncodeToString(proposalState.ProposalOwner),
		ProposerDID:             proposerDID,
		AvailableAmount:         crCommittee.AvailableWithdrawalAmount(proposalHash).String(),
	}

	switch proposalState.Proposal.ProposalType {
	case payload.Normal, payload.ELIP:
		var rpcProposal RPCCRCProposal
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.Budgets = make([]BudgetInfo, 0)
		for _, b := range proposalState.Proposal.Budgets {
			budgetStatus := proposalState.BudgetsStatus[b.Stage]
			paymentCriteria := budgetInDraftData[b.Stage]
			rpcProposal.Budgets = append(rpcProposal.Budgets, BudgetInfo{
				Type:            b.Type.Name(),
				Stage:           b.Stage,
				Amount:          b.Amount.String(),
				Status:          budgetStatus.Name(),
				PaymentCriteria: paymentCriteria,
			})
		}
		rpcProposal.Milestone = proposalData.Milestone

		var err error
		rpcProposal.Recipient, err = proposalState.Recipient.ToAddress()
		if err != nil {
			return ResponsePack(InternalError, "invalidate Recipient")
		}
		rpcProposalState.Proposal = rpcProposal

	case payload.SecretaryGeneral:
		var rpcProposal RPCSecretaryGeneralProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.SecretaryGeneralPublicKey =
			common.BytesToHexString(proposalState.Proposal.SecretaryGeneralPublicKey)
		sgDID, _ := proposalState.Proposal.SecretaryGeneralDID.ToAddress()
		rpcProposal.SecretaryGeneralDID = sgDID
		cmDID, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = cmDID
		rpcProposalState.Proposal = rpcProposal

	case payload.ChangeProposalOwner:
		var rpcProposal RPCChangeProposalOwnerProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.TargetProposalHash = common.ToReversedString(proposalState.Proposal.TargetProposalHash)
		var err error
		rpcProposal.NewRecipient, err = proposalState.Proposal.NewRecipient.ToAddress()
		if err != nil {
			return ResponsePack(InternalError, "invalidate NewRecipient")
		}
		rpcProposal.NewOwnerPublicKey = common.BytesToHexString(proposalState.Proposal.NewOwnerPublicKey)
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal

	case payload.CloseProposal:
		var rpcProposal RPCCloseProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.TargetProposalHash = common.ToReversedString(proposalState.Proposal.TargetProposalHash)
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal

	case payload.ReserveCustomID:
		var rpcProposal RPCReservedCustomIDProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.ReservedCustomIDList = proposalState.Proposal.ReservedCustomIDList
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal

	case payload.ReceiveCustomID:
		var rpcProposal RPCReceiveCustomIDProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.ReceiveCustomIDList = proposalState.Proposal.ReceivedCustomIDList
		rpcProposal.ReceiverDID, _ = proposalState.Proposal.ReceiverDID.ToAddress()
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal

	case payload.ChangeCustomIDFee:
		var rpcProposal RPCChangeCustomIDFeeProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)
		rpcProposal.Fee = int64(proposalState.Proposal.RateOfCustomIDFee)
		rpcProposal.EIDEffectiveHeight = proposalState.Proposal.EIDEffectiveHeight
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal

	case payload.RegisterSideChain:
		var rpcProposal RPCRegisterSideChainProposal
		rpcProposal.ProposalType = proposalState.Proposal.ProposalType.Name()
		rpcProposal.CategoryData = proposalState.Proposal.CategoryData
		rpcProposal.OwnerPublicKey = common.BytesToHexString(proposalState.Proposal.OwnerPublicKey)
		rpcProposal.DraftHash = common.ToReversedString(proposalState.Proposal.DraftHash)

		rpcProposal.SideChainInfo.SideChainName = proposalState.Proposal.SideChainName
		rpcProposal.SideChainInfo.MagicNumber = proposalState.Proposal.MagicNumber
		rpcProposal.SideChainInfo.GenesisHash = common.ToReversedString(proposalState.Proposal.GenesisHash)
		rpcProposal.SideChainInfo.ExchangeRate = proposalState.Proposal.ExchangeRate.String()
		rpcProposal.SideChainInfo.EffectiveHeight = proposalState.Proposal.EffectiveHeight
		rpcProposal.SideChainInfo.ResourcePath = proposalState.Proposal.ResourcePath
		did, _ := proposalState.Proposal.CRCouncilMemberDID.ToAddress()
		rpcProposal.CRCouncilMemberDID = did
		rpcProposalState.Proposal = rpcProposal
	}

	result := &RPCCRProposalStateInfo{ProposalState: rpcProposalState}
	return ResponsePack(Success, result)
}

func GetProposalDraftData(param Params) map[string]interface{} {
	hash, ok := param.String("drafthash")
	if !ok {
		return ResponsePack(InvalidParams, "not found hash")
	}
	draftHashStr, err := common.FromReversedString(hash)
	if err != nil {
		return ResponsePack(InvalidParams, "invalidate hash")
	}
	draftHash, err := common.Uint256FromBytes(draftHashStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalidate draft hash")
	}

	data, _ := Chain.GetDB().GetProposalDraftDataByDraftHash(draftHash)
	var result string
	if data != nil {
		result = common.BytesToHexString(data)
	} else {
		return ResponsePack(InvalidParams, "invalidate draft hash")
	}

	return ResponsePack(Success, result)
}

// Support query proposalTitle based on proposalHash or draftHash
func GetProposalTitle(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()

	// Get DraftHash
	var draftHash *common.Uint256
	ProposalHashHexStr, ok := param.String("proposalhash")
	if ok {
		proposalHashBytes, err := common.FromReversedString(ProposalHashHexStr)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate proposalhash")
		}
		ProposalHash, err := common.Uint256FromBytes(proposalHashBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate proposalhash")
		}
		proposalState := crCommittee.GetProposal(*ProposalHash)
		if proposalState == nil {
			return ResponsePack(InvalidParams, "invalidate proposalhash")
		}
		_draftHash := proposalState.Proposal.DraftHash
		draftHash = &_draftHash
	} else {
		DraftHashStr, ok := param.String("drafthash")
		if !ok {
			return ResponsePack(InvalidParams, "params at least one of proposalhash and DraftHash")
		}
		DraftHashStrBytes, err := common.FromReversedString(DraftHashStr)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate drafthash")
		}
		draftHash, err = common.Uint256FromBytes(DraftHashStrBytes)
		if err != nil {
			return ResponsePack(InvalidParams, "invalidate drafthash")
		}
	}

	// Parse draftData and get proposalTitle
	draftData, errorStr := parseProposalDraftData(draftHash)
	if errorStr != "" {
		return ResponsePack(InvalidParams, errorStr)
	}
	proposalDraftData, err := draftData.(CRCProposalDraftData)
	if err {
		return ResponsePack(Success, proposalDraftData.Title)
	} else {
		return ResponsePack(InvalidParams, "invalidate proposalhash")
	}

}

func getProposalTitleByProposalHash(hash common.Uint256) string {
	crCommittee := Chain.GetCRCommittee()
	proposalState := crCommittee.GetProposal(hash)
	_draftHash := proposalState.Proposal.DraftHash
	draftHash := &_draftHash
	draftData, errorStr := parseProposalDraftData(draftHash)
	if errorStr != "" {
		log.Error(InvalidParams, errorStr)
		return ""
	}
	proposalDraftData, err := draftData.(CRCProposalDraftData)
	if err {
		return proposalDraftData.Title
	} else {
		log.Error(InvalidParams, "invalidate proposalhash")
		return ""
	}

}

func getTimestampByHeight(height uint32) uint32 {
	hash, err := Chain.GetBlockHash(height)
	if err != nil {
		log.Error(UnknownTransaction, "")
		return 0
	}
	header, err := Chain.GetHeader(hash)
	if err != nil {
		log.Error(UnknownTransaction, "")
		return 0
	}
	return header.Timestamp
}

func ProducerStatus(param Params) map[string]interface{} {
	publicKey, ok := param.String("publickey")
	if !ok {
		return ResponsePack(InvalidParams, "public key not found")
	}
	publicKeyBytes, err := common.HexStringToBytes(publicKey)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid public key")
	}
	producer := Chain.GetState().GetProducer(publicKeyBytes)
	if producer == nil {
		return ResponsePack(InvalidParams, "unknown producer public key")
	}
	return ResponsePack(Success, producer.State().String())
}

func VoteStatus(param Params) map[string]interface{} {
	address, ok := param.String("address")
	if !ok {
		return ResponsePack(InvalidParams, "address not found")
	}

	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "Invalid address: "+address)
	}
	utxos, err := Store.GetFFLDB().GetUTXO(programHash)
	if err != nil {
		return ResponsePack(InvalidParams, "list unspent failed, "+err.Error())
	}
	var total common.Fixed64
	var voting common.Fixed64
	for _, utxo := range utxos {
		tx, _, err := Store.GetTransaction(utxo.TxID)
		if err != nil {
			return ResponsePack(InternalError, "unknown transaction "+utxo.TxID.String()+" from persisted utxo")
		}
		if tx.Outputs()[utxo.Index].Type == common2.OTVote {
			voting += utxo.Value
		}
		total += utxo.Value
	}

	pending := false
	for _, t := range TxMemPool.GetTxsInPool() {
		for _, i := range t.Inputs() {
			tx, _, err := Store.GetTransaction(i.Previous.TxID)
			if err != nil {
				return ResponsePack(InternalError, "unknown transaction "+i.Previous.TxID.String()+" from persisted utxo")
			}
			if tx.Outputs()[i.Previous.Index].ProgramHash.IsEqual(*programHash) {
				pending = true
			}
		}
		for _, o := range t.Outputs() {
			if o.Type == common2.OTVote && o.ProgramHash.IsEqual(*programHash) {
				pending = true
			}
		}
		if pending {
			break
		}
	}

	type voteInfo struct {
		Total   string `json:"total"`
		Voting  string `json:"voting"`
		Pending bool   `json:"pending"`
	}
	return ResponsePack(Success, &voteInfo{
		Total:   total.String(),
		Voting:  voting.String(),
		Pending: pending,
	})
}

func GetPoll(param Params) map[string]interface{} {
	type Poll struct {
		IDs []string `json:"ids"`
	}
	votings := Chain.GetState().InitateVotings
	ids := make([]string, 0)
	for _, v := range votings {
		ids = append(ids, v.ID.String())
	}
	return ResponsePack(Success, &Poll{IDs: ids})
}

func GetVotingInfo(param Params) map[string]interface{} {
	ids, exist := param.ArrayString("ids")
	if !exist {
		return ResponsePack(InvalidParams, "need a param called ids")
	}

	type VotingInfo struct {
		ID          string   `json:"id"`
		Status      string   `json:"status"`
		Description string   `json:"description"`
		StartTime   uint64   `json:"startTime"`
		EndTime     uint64   `json:"endTime"`
		Options     []string `json:"options"`
	}
	votings := Chain.GetState().InitateVotings
	idsMap := make(map[string]struct{})
	for _, id := range ids {
		idsMap[id] = struct{}{}
	}

	result := make([]VotingInfo, 0)
	for _, v := range votings {
		if _, ok := idsMap[v.ID.String()]; ok {
			var status string
			if time.Now().Unix() < int64(v.EndTime) {
				status = "voting"
			} else {
				status = "finished"
			}
			result = append(result, VotingInfo{
				ID:          v.ID.String(),
				Status:      status,
				Description: v.Description,
				StartTime:   v.StartTime,
				EndTime:     v.EndTime,
				Options:     v.Options,
			})
		}
	}

	return ResponsePack(Success, result)
}

func GetVotingDetails(param Params) map[string]interface{} {
	idStr, exist := param.String("id")
	if !exist {
		return ResponsePack(InvalidParams, "need a param called id")
	}
	id, err := common.Uint256FromHexString(idStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid id")
	}
	voting, ok := Chain.GetState().InitateVotings[*id]
	if !ok {
		return ResponsePack(InvalidParams, "id is not exist")
	}
	type VoteInfo struct {
		Voter  string `json:"voter"`
		Amount string `json:"amount"`
		Option uint32 `json:"option"`
	}
	type VotingDetails struct {
		ID          string     `json:"id"`
		Status      string     `json:"status"`
		Description string     `json:"description"`
		StartTime   uint64     `json:"startTime"`
		EndTime     uint64     `json:"endTime"`
		Options     []string   `json:"options"`
		Votes       []VoteInfo `json:"votes"`
	}

	var status string
	if time.Now().Unix() < int64(voting.EndTime) {
		status = "voting"
	} else {
		status = "finished"
	}

	votes := make([]VoteInfo, 0)
	userVotes := Chain.GetState().UserVotings
	for k, v := range userVotes[*id] {
		votes = append(votes, VoteInfo{
			Voter:  k,
			Amount: v.Amount,
			Option: v.OptionIndex,
		})
	}
	return ResponsePack(Success, &VotingDetails{
		ID:          idStr,
		Status:      status,
		Description: voting.Description,
		StartTime:   voting.StartTime,
		EndTime:     voting.EndTime,
		Options:     voting.Options,
		Votes:       votes,
	})
}

func GetDepositCoin(param Params) map[string]interface{} {
	pk, ok := param.String("ownerpublickey")
	if !ok {
		return ResponsePack(InvalidParams, "need a param called ownerpublickey")
	}
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid public key")
	}
	producer := Chain.GetState().GetProducer(pkBytes)
	if producer == nil {
		return ResponsePack(InvalidParams, "invalid publickey")
	}
	type depositCoin struct {
		Available string `json:"available"`
		Deducted  string `json:"deducted"`
		Deposit   string `json:"deposit"`
		Assets    string `json:"assets"`
	}

	depositAmount := common.Fixed64(0)
	availableAmount := common.Fixed64(0)

	depositAmount = producer.DepositAmount()
	availableAmount = producer.AvailableAmount()
	return ResponsePack(Success, &depositCoin{
		Available: availableAmount.String(),
		Deducted:  producer.Penalty().String(),
		Deposit:   depositAmount.String(),
		Assets:    producer.TotalAmount().String(),
	})
}

func GetCRDepositCoin(param Params) map[string]interface{} {
	crCommittee := Chain.GetCRCommittee()
	var availableDepositAmount, penaltyAmount, depositAmount, totalAmount common.Fixed64
	pubkey, hasPubkey := param.String("publickey")
	if hasPubkey {
		available, penalty, deposit, total, err := crCommittee.GetDepositAmountByPublicKey(pubkey)
		if err != nil {
			return ResponsePack(InvalidParams, err.Error())
		}
		availableDepositAmount = available
		penaltyAmount = penalty
		depositAmount = deposit
		totalAmount = total
	}
	id, hasID := param.String("id")
	if hasID {
		programHash, err := common.Uint168FromAddress(id)
		if err != nil {
			return ResponsePack(InvalidParams, "invalid id to programHash")
		}
		available, penalty, deposit, total, err := crCommittee.GetDepositAmountByID(*programHash)
		if err != nil {
			return ResponsePack(InvalidParams, err.Error())
		}
		availableDepositAmount = available
		penaltyAmount = penalty
		depositAmount = deposit
		totalAmount = total
	}

	if !hasPubkey && !hasID {
		return ResponsePack(InvalidParams, "need a param called "+
			"publickey or id")
	}

	type depositCoin struct {
		Available string `json:"available"`
		Deducted  string `json:"deducted"`
		Deposit   string `json:"deposit"`
		Assets    string `json:"assets"`
	}
	return ResponsePack(Success, &depositCoin{
		Available: availableDepositAmount.String(),
		Deducted:  penaltyAmount.String(),
		Deposit:   depositAmount.String(),
		Assets:    totalAmount.String(),
	})
}

func EstimateSmartFee(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.TransactionPermitted); rtn != nil {
		return rtn
	}

	confirm, ok := param.Int("confirmations")
	if !ok {
		return ResponsePack(InvalidParams, "need a param called confirmations")
	}
	if confirm > 25 {
		return ResponsePack(InvalidParams, "support only 25 confirmations at most")
	}
	var FeeRate = 10000 //basic fee rate 10000 sela per KB
	var count = 0

	// TODO just return fixed transaction fee for now, we didn't have that much
	// transactions in a block yet.

	return ResponsePack(Success, GetFeeRate(count, int(confirm))*FeeRate)
}

func GetFeeRate(count int, confirm int) int {
	gap := count - confirm
	if gap < 0 {
		gap = -1
	}
	return gap + 2
}

func DecodeRawTransaction(param Params) map[string]interface{} {
	if rtn := checkRPCServiceLevel(config.WalletPermitted); rtn != nil {
		return rtn
	}

	dataParam, ok := param.String("data")
	if !ok {
		return ResponsePack(InvalidParams, "need a parameter named data")
	}
	txBytes, err := common.HexStringToBytes(dataParam)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid raw tx data, "+err.Error())
	}
	r := bytes.NewReader(txBytes)
	txn, err := functions.GetTransactionByBytes(r)
	if err != nil {
		return ResponsePack(InvalidTransaction, "invalid transaction")
	}
	if err := txn.Deserialize(r); err != nil {
		return ResponsePack(InvalidParams, "invalid raw tx data, "+err.Error())
	}

	return ResponsePack(Success, GetTransactionInfo(txn))
}

func getPayloadInfo(tx interfaces.Transaction, payloadVersion byte) PayloadInfo {
	p := tx.Payload()
	switch object := p.(type) {
	case *payload.CoinBase:
		obj := new(CoinbaseInfo)
		obj.CoinbaseData = string(object.Content)
		return obj
	case *payload.RegisterAsset:
		obj := new(RegisterAssetInfo)
		obj.Asset = object.Asset
		obj.Amount = object.Amount.String()
		obj.Controller = common.BytesToHexString(common.BytesReverse(object.Controller.Bytes()))
		return obj
	case *payload.SideChainPow:
		obj := new(SideChainPowInfo)
		obj.BlockHeight = object.BlockHeight
		obj.SideBlockHash = object.SideBlockHash.String()
		obj.SideGenesisHash = object.SideGenesisHash.String()
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj
	case *payload.WithdrawFromSideChain:
		switch payloadVersion {
		case payload.WithdrawFromSideChainVersion:
			obj := new(WithdrawFromSideChainInfo)
			obj.BlockHeight = object.BlockHeight
			obj.GenesisBlockAddress = object.GenesisBlockAddress
			for _, hash := range object.SideChainTransactionHashes {
				obj.SideChainTransactionHashes = append(obj.SideChainTransactionHashes, hash.String())
			}
			return obj
		case payload.WithdrawFromSideChainVersionV1:
			return nil
		case payload.WithdrawFromSideChainVersionV2:
			obj := new(SchnorrWithdrawFromSideChainInfo)
			obj.Signers = make([]uint32, 0)
			for _, s := range object.Signers {
				obj.Signers = append(obj.Signers, uint32(s))
			}
			return obj
		}
		return nil

	case *payload.TransferCrossChainAsset:
		if payloadVersion == payload.TransferCrossChainVersionV1 {
			return nil
		}
		obj := new(TransferCrossChainAssetInfo)
		obj.CrossChainAddresses = object.CrossChainAddresses
		obj.OutputIndexes = object.OutputIndexes
		obj.CrossChainAmounts = object.CrossChainAmounts
		return obj
	case *payload.TransferAsset:
	case *payload.Record:
	case *payload.ProducerInfo:
		obj := new(ProducerInfo)
		obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
		obj.NodePublicKey = common.BytesToHexString(object.NodePublicKey)
		obj.NickName = object.NickName
		obj.Url = object.Url
		obj.Location = object.Location
		obj.NetAddress = object.NetAddress
		obj.StakeUntil = object.StakeUntil
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj
	case *payload.ProcessProducer:
		obj := new(CancelProducerInfo)
		obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj
	case *payload.InactiveArbitrators:
		var arbitrators []string
		for _, a := range object.Arbitrators {
			arbitrators = append(arbitrators, common.BytesToHexString(a))
		}
		obj := new(InactiveArbitratorsInfo)
		obj.Sponsor = common.BytesToHexString(object.Sponsor)
		obj.Arbitrators = arbitrators
		return obj
	case *payload.RevertToDPOS:
		obj := new(RevertToDPOSInfo)
		obj.WorkHeightInterval = object.WorkHeightInterval
		obj.RevertToPOWBlockHeight = object.RevertToPOWBlockHeight
		return obj
	case *payload.RevertToPOW:
		obj := new(RevertToPOWInfo)
		obj.Type = object.Type.String()
		obj.WorkingHeight = object.WorkingHeight
		return obj
	case *payload.ActivateProducer:
		obj := new(ActivateProducerInfo)
		obj.NodePublicKey = common.BytesToHexString(object.NodePublicKey)
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj
	case *payload.UpdateVersion:
		obj := new(UpdateVersionInfo)
		obj.StartHeight = object.StartHeight
		obj.EndHeight = object.EndHeight
		return obj
	case *payload.CRInfo:
		switch payloadVersion {
		case payload.CRInfoSchnorrVersion, payload.CRInfoMultiSignVersion:
			obj := new(MultiCRInfo)
			cid, _ := object.CID.ToAddress()
			obj.CID = cid
			did, _ := object.DID.ToAddress()
			if object.DID.IsEqual(emptyHash) {
				obj.DID = ""
			} else {
				obj.DID = did
			}
			obj.NickName = object.NickName
			obj.Url = object.Url
			obj.Location = object.Location
			return obj

		default:
			obj := new(CRInfo)
			obj.Code = common.BytesToHexString(object.Code)
			cid, _ := object.CID.ToAddress()
			obj.CID = cid
			did, _ := object.DID.ToAddress()
			if object.DID.IsEqual(emptyHash) {
				obj.DID = ""
			} else {
				obj.DID = did
			}
			obj.NickName = object.NickName
			obj.Url = object.Url
			obj.Location = object.Location
			obj.Signature = common.BytesToHexString(object.Signature)
			return obj
		}

	case *payload.UnregisterCR:
		obj := new(UnregisterCRInfo)
		cid, _ := object.CID.ToAddress()
		obj.CID = cid
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj
	case *payload.CRCProposal:

		switch object.ProposalType {
		case payload.Normal, payload.ELIP:
			var budgets []BudgetBaseInfo
			for _, b := range object.Budgets {
				budgets = append(budgets, BudgetBaseInfo{
					Type:   b.Type.Name(),
					Stage:  b.Stage,
					Amount: b.Amount.String(),
				})
			}
			obj := new(CRCProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.Budgets = budgets
			addr, _ := object.Recipient.ToAddress()
			obj.Recipient = addr
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.ChangeProposalOwner:
			obj := new(CRCChangeProposalOwnerInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.TargetProposalHash = common.ToReversedString(object.TargetProposalHash)
			addr, _ := object.NewRecipient.ToAddress()
			obj.NewRecipient = addr
			obj.NewOwnerPublicKey = common.BytesToHexString(object.NewOwnerKey)
			obj.Signature = common.BytesToHexString(object.Signature)
			obj.NewOwnerSignature = common.BytesToHexString(object.NewOwnerSignature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.CloseProposal:
			obj := new(CRCCloseProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.TargetProposalHash = common.ToReversedString(object.TargetProposalHash)
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.ReserveCustomID:
			obj := new(CRCReservedCustomIDProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.ReservedCustomIDList = object.ReservedCustomIDList
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.ReceiveCustomID:
			obj := new(CRCReceivedCustomIDProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.ReceiveCustomIDList = object.ReceivedCustomIDList
			obj.ReceiverDID, _ = object.ReceiverDID.ToAddress()
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.ChangeCustomIDFee:
			obj := new(CRCChangeCustomIDFeeInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.FeeRate = int64(object.RateOfCustomIDFee)
			obj.EIDEffectiveHeight = object.EIDEffectiveHeight
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.SecretaryGeneral:
			obj := new(CRCSecretaryGeneralProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.SecretaryGeneralPublicKey = common.BytesToHexString(object.SecretaryGeneralPublicKey)
			sgDID, _ := object.SecretaryGeneralDID.ToAddress()
			obj.SecretaryGeneralDID = sgDID
			obj.Signature = common.BytesToHexString(object.Signature)
			obj.SecretaryGeneraSignature = common.BytesToHexString(object.SecretaryGeneraSignature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj

		case payload.RegisterSideChain:
			obj := new(CRCRegisterSideChainProposalInfo)
			obj.ProposalType = object.ProposalType.Name()
			obj.CategoryData = object.CategoryData
			obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
			obj.DraftData = common.BytesToHexString(object.DraftData)
			obj.DraftHash = common.ToReversedString(object.DraftHash)
			obj.SideChainName = object.SideChainName
			obj.MagicNumber = object.MagicNumber
			obj.GenesisHash = common.ToReversedString(object.GenesisHash)
			obj.ExchangeRate = object.ExchangeRate
			obj.EffectiveHeight = object.EffectiveHeight
			obj.ResourcePath = object.ResourcePath
			obj.Signature = common.BytesToHexString(object.Signature)
			crmdid, _ := object.CRCouncilMemberDID.ToAddress()
			obj.CRCouncilMemberDID = crmdid
			obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilMemberSignature)
			obj.Hash = common.ToReversedString(object.Hash(payloadVersion))
			return obj
		}

	case *payload.RecordProposalResult:
		obj := new(CRCCustomIDProposalResultInfo)
		for _, r := range object.ProposalResults {
			result := ProposalResultInfo{
				ProposalHash: common.ToReversedString(r.ProposalHash),
				ProposalType: r.ProposalType.Name(),
				Result:       r.Result,
			}
			obj.ProposalResults = append(obj.ProposalResults, result)
		}

		return obj

	case *payload.CRCProposalReview:
		obj := new(CRCProposalReviewInfo)
		obj.ProposalHash = common.ToReversedString(object.ProposalHash)
		obj.VoteResult = object.VoteResult.Name()
		obj.OpinionData = common.BytesToHexString(object.OpinionData)
		obj.OpinionHash = common.ToReversedString(object.OpinionHash)
		did, _ := object.DID.ToAddress()
		obj.DID = did
		obj.Sign = common.BytesToHexString(object.Signature)
		return obj

	case *payload.CRCProposalTracking:
		obj := new(CRCProposalTrackingInfo)
		obj.ProposalTrackingType = object.ProposalTrackingType.Name()
		obj.ProposalHash = common.ToReversedString(object.ProposalHash)
		obj.MessageData = common.BytesToHexString(object.MessageData)
		obj.MessageHash = common.ToReversedString(object.MessageHash)
		obj.Stage = object.Stage
		obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
		obj.NewOwnerPublicKey = common.BytesToHexString(object.NewOwnerKey)
		obj.OwnerSignature = common.BytesToHexString(object.OwnerSignature)
		obj.NewOwnerPublicKey = common.BytesToHexString(object.NewOwnerKey)
		obj.SecretaryGeneralOpinionData = common.BytesToHexString(object.SecretaryGeneralOpinionData)
		obj.SecretaryGeneralOpinionHash = common.ToReversedString(object.SecretaryGeneralOpinionHash)
		obj.SecretaryGeneralSignature = common.BytesToHexString(object.SecretaryGeneralSignature)
		obj.NewOwnerSignature = common.BytesToHexString(object.NewOwnerSignature)
		return obj

	case *payload.CRCProposalWithdraw:
		obj := new(CRCProposalWithdrawInfo)
		obj.ProposalHash = common.ToReversedString(object.ProposalHash)
		obj.OwnerPublicKey = common.BytesToHexString(object.OwnerKey)
		if payloadVersion == payload.CRCProposalWithdrawVersion01 {
			recipient, err := object.Recipient.ToAddress()
			if err == nil {
				obj.Recipient = recipient
			}
			obj.Amount = object.Amount.String()
		}
		obj.Signature = common.BytesToHexString(object.Signature)
		return obj

	case *payload.CRCouncilMemberClaimNode:
		obj := new(CRCouncilMemberClaimNodeInfo)
		obj.NodePublicKey = common.BytesToHexString(object.NodePublicKey)
		obj.CRCouncilMemberDID, _ = object.CRCouncilCommitteeDID.ToAddress()
		obj.CRCouncilMemberSignature = common.BytesToHexString(object.CRCouncilCommitteeSignature)
		return obj

	case *payload.NextTurnDPOSInfo:
		if payloadVersion == payload.NextTurnDPOSInfoVersion {
			obj := new(NextTurnDPOSPayloadInfo)
			crPublicKeysString := make([]string, 0)
			dposPublicKeysString := make([]string, 0)
			for _, v := range object.CRPublicKeys {
				crPublicKeysString = append(crPublicKeysString, common.BytesToHexString(v))
			}
			for _, v := range object.DPOSPublicKeys {
				dposPublicKeysString = append(dposPublicKeysString, common.BytesToHexString(v))
			}
			obj.WorkingHeight = object.WorkingHeight
			obj.CRPublickeys = crPublicKeysString
			obj.DPOSPublicKeys = dposPublicKeysString
			return obj
		}

		obj := new(NextTurnDPOSPayloadInfoV2)
		crPublicKeysString := make([]string, 0)
		dposPublicKeysString := make([]string, 0)
		completeCRPublicKeysString := make([]string, 0)
		for _, v := range object.CRPublicKeys {
			crPublicKeysString = append(crPublicKeysString, common.BytesToHexString(v))
		}
		for _, v := range object.DPOSPublicKeys {
			dposPublicKeysString = append(dposPublicKeysString, common.BytesToHexString(v))
		}
		for _, v := range object.CompleteCRPublicKeys {
			completeCRPublicKeysString = append(completeCRPublicKeysString, common.BytesToHexString(v))
		}
		obj.WorkingHeight = object.WorkingHeight
		obj.CRPublicKeys = crPublicKeysString
		obj.DPOSPublicKeys = dposPublicKeysString
		obj.CompleteCRPublicKeys = completeCRPublicKeysString
		return obj

	case *payload.CRCProposalRealWithdraw:
		obj := new(CRCProposalRealWithdrawInfo)
		obj.WithdrawTransactionHashes = make([]string, 0)
		for _, hash := range object.WithdrawTransactionHashes {
			obj.WithdrawTransactionHashes =
				append(obj.WithdrawTransactionHashes, common.ToReversedString(hash))
		}
		return obj

	case *payload.DPOSIllegalProposals:
		obj := new(DPOSIllegalProposalsInfo)
		obj.Hash = common.ToReversedString(object.Hash())
		obj.Evidence = ProposalEvidenceInfo{
			Proposal: DPOSProposalInfo{
				Sponsor:    common.BytesToHexString(object.Evidence.Proposal.Sponsor),
				BlockHash:  common.ToReversedString(object.Evidence.Proposal.BlockHash),
				ViewOffset: object.Evidence.Proposal.ViewOffset,
				Sign:       common.BytesToHexString(object.Evidence.Proposal.Sign),
				Hash:       common.ToReversedString(object.Evidence.Proposal.Hash()),
			},
			BlockHeight: object.Evidence.BlockHeight,
		}
		obj.CompareEvidence = ProposalEvidenceInfo{
			Proposal: DPOSProposalInfo{
				Sponsor:    common.BytesToHexString(object.CompareEvidence.Proposal.Sponsor),
				BlockHash:  common.ToReversedString(object.CompareEvidence.Proposal.BlockHash),
				ViewOffset: object.CompareEvidence.Proposal.ViewOffset,
				Sign:       common.BytesToHexString(object.CompareEvidence.Proposal.Sign),
				Hash:       common.ToReversedString(object.CompareEvidence.Proposal.Hash()),
			},
			BlockHeight: object.CompareEvidence.BlockHeight,
		}
		return obj

	case *payload.DPOSIllegalVotes:
		obj := new(DPOSIllegalVotesInfo)
		obj.Hash = common.ToReversedString(object.Hash())
		obj.Evidence = VoteEvidenceInfo{
			ProposalEvidenceInfo: ProposalEvidenceInfo{
				Proposal: DPOSProposalInfo{
					Sponsor:    common.BytesToHexString(object.Evidence.Proposal.Sponsor),
					BlockHash:  common.ToReversedString(object.Evidence.Proposal.BlockHash),
					ViewOffset: object.Evidence.Proposal.ViewOffset,
					Sign:       common.BytesToHexString(object.Evidence.Proposal.Sign),
					Hash:       common.ToReversedString(object.Evidence.Proposal.Hash()),
				},
				BlockHeight: object.Evidence.BlockHeight,
			},
			Vote: DPOSProposalVoteInfo{
				ProposalHash: common.ToReversedString(object.Evidence.Vote.ProposalHash),
				Signer:       common.BytesToHexString(object.Evidence.Vote.Signer),
				Accept:       object.Evidence.Vote.Accept,
				Sign:         common.BytesToHexString(object.Evidence.Vote.Sign),
				Hash:         common.ToReversedString(object.Evidence.Vote.Hash()),
			},
		}
		obj.CompareEvidence = VoteEvidenceInfo{
			ProposalEvidenceInfo: ProposalEvidenceInfo{
				Proposal: DPOSProposalInfo{
					Sponsor:    common.BytesToHexString(object.CompareEvidence.Proposal.Sponsor),
					BlockHash:  common.ToReversedString(object.CompareEvidence.Proposal.BlockHash),
					ViewOffset: object.CompareEvidence.Proposal.ViewOffset,
					Sign:       common.BytesToHexString(object.CompareEvidence.Proposal.Sign),
					Hash:       common.ToReversedString(object.CompareEvidence.Proposal.Hash()),
				},
				BlockHeight: object.CompareEvidence.BlockHeight,
			},
			Vote: DPOSProposalVoteInfo{
				ProposalHash: common.ToReversedString(object.CompareEvidence.Vote.ProposalHash),
				Signer:       common.BytesToHexString(object.CompareEvidence.Vote.Signer),
				Accept:       object.CompareEvidence.Vote.Accept,
				Sign:         common.BytesToHexString(object.CompareEvidence.Vote.Sign),
				Hash:         common.ToReversedString(object.CompareEvidence.Vote.Hash()),
			},
		}
		return obj

	case *payload.DPOSIllegalBlocks:
		obj := new(DPOSIllegalBlocksInfo)
		obj.Hash = common.ToReversedString(object.Hash())
		obj.CoinType = uint32(object.CoinType)
		obj.BlockHeight = object.BlockHeight
		eviSigners := make([]string, 0)
		for _, s := range object.Evidence.Signers {
			eviSigners = append(eviSigners, common.BytesToHexString(s))
		}
		obj.Evidence = BlockEvidenceInfo{
			Header:       common.BytesToHexString(object.Evidence.Header),
			BlockConfirm: common.BytesToHexString(object.Evidence.BlockConfirm),
			Signers:      eviSigners,
			Hash:         common.ToReversedString(object.Evidence.BlockHash()),
		}
		compEviSigners := make([]string, 0)
		for _, s := range object.CompareEvidence.Signers {
			compEviSigners = append(compEviSigners, common.BytesToHexString(s))
		}
		obj.CompareEvidence = BlockEvidenceInfo{
			Header:       common.BytesToHexString(object.CompareEvidence.Header),
			BlockConfirm: common.BytesToHexString(object.CompareEvidence.BlockConfirm),
			Signers:      compEviSigners,
			Hash:         common.ToReversedString(object.CompareEvidence.BlockHash()),
		}
		return obj

	case *payload.Voting:
		obj := new(VotingInfo)
		for _, rc := range object.RenewalContents {
			obj.RenewalContents = append(obj.RenewalContents, RenewalVotesContentInfo{
				ReferKey: common.ToReversedString(rc.ReferKey),
				VotesInfo: VotesWithLockTimeInfo{
					Candidate: common.BytesToHexString(rc.VotesInfo.Candidate),
					Votes:     rc.VotesInfo.Votes.String(),
					LockTime:  rc.VotesInfo.LockTime,
				},
			})
		}
		for _, rc := range object.Contents {
			votesinfo := make([]VotesWithLockTimeInfo, 0)
			for _, detail := range rc.VotesInfo {
				var candidate string
				switch rc.VoteType {
				case outputpayload.CRC, outputpayload.CRCImpeachment:
					c, _ := common.Uint168FromBytes(detail.Candidate)
					candidate, _ = c.ToAddress()
				case outputpayload.CRCProposal:
					proposalHash, _ := common.Uint256FromBytes(detail.Candidate)
					candidate = common.ToReversedString(*proposalHash)
				default:
					candidate = common.BytesToHexString(detail.Candidate)
				}

				votesinfo = append(votesinfo, VotesWithLockTimeInfo{
					Candidate: candidate,
					Votes:     detail.Votes.String(),
					LockTime:  detail.LockTime,
				})
			}
			obj.Contents = append(obj.Contents, VotesContentInfo{
				VoteType:  byte(rc.VoteType),
				VotesInfo: votesinfo,
			})
		}
		return obj

	case *payload.ExchangeVotes:
		obj := new(ExchangeVotesInfo)
		return obj
	case *payload.ReturnVotes:
		address, _ := object.ToAddr.ToAddress()
		if payloadVersion == payload.ReturnVotesSchnorrVersion {
			obj := &ReturnVotesInfo{
				ToAddr: address,
				Value:  object.Value.String(),
			}
			return obj
		}
		obj := &ReturnVotesInfo{
			ToAddr:    address,
			Code:      common.BytesToHexString(object.Code),
			Value:     object.Value.String(),
			Signature: common.BytesToHexString(object.Signature),
		}
		return obj
	case *payload.VotesRealWithdrawPayload:
		obj := &RealVotesWithdrawInfo{
			RealReturnVotes: make([]RealReturnVotesInfo, 0),
		}
		for _, withdraw := range object.VotesRealWithdraw {
			address, _ := withdraw.StakeAddress.ToAddress()
			realReturnVotesInfo := RealReturnVotesInfo{
				ReturnVotesTXHash: common.ToReversedString(withdraw.ReturnVotesTXHash),
				StakeAddress:      address,
				Value:             withdraw.Value.String(),
			}
			obj.RealReturnVotes = append(obj.RealReturnVotes, realReturnVotesInfo)
		}
		return obj
	case *payload.DPoSV2ClaimReward:
		address, _ := object.ToAddr.ToAddress()
		if payloadVersion == payload.DposV2ClaimRewardVersionV1 {
			obj := &DposV2ClaimRewardInfo{
				ToAddr: address,
				Value:  object.Value.String(),
			}
			return obj
		}
		obj := &DposV2ClaimRewardInfo{
			ToAddr:    address,
			Code:      common.BytesToHexString(object.Code),
			Value:     object.Value.String(),
			Signature: common.BytesToHexString(object.Signature),
		}
		return obj
	case *payload.DposV2ClaimRewardRealWithdraw:
		obj := &DposV2ClaimRewardRealWithdrawInfo{
			WithdrawTransactionHashes: make([]string, 0),
		}
		for _, txHash := range object.WithdrawTransactionHashes {
			obj.WithdrawTransactionHashes = append(obj.WithdrawTransactionHashes, common.ToReversedString(txHash))
		}
		return obj

	case *payload.CreateNFT:
		if payloadVersion == payload.CreateNFTVersion {
			obj := &CreateNFTInfo{
				ID:               common.GetNFTID(object.ReferKey, tx.Hash()).ReversedString(),
				ReferKey:         object.ReferKey.ReversedString(),
				StakeAddress:     object.StakeAddress,
				GenesisBlockHash: common.ToReversedString(object.GenesisBlockHash),
			}
			return obj
		}

		obj := &CreateNFTInfoV2{
			ID:               common.GetNFTID(object.ReferKey, tx.Hash()).ReversedString(),
			ReferKey:         object.ReferKey.ReversedString(),
			StakeAddress:     object.StakeAddress,
			GenesisBlockHash: common.ToReversedString(object.GenesisBlockHash),
			StartHeight:      object.StartHeight,
			EndHeight:        object.EndHeight,
			Votes:            object.Votes.String(),
			VoteRights:       object.VoteRights.String(),
			TargetOwnerKey:   common.BytesToHexString(object.TargetOwnerKey),
		}
		return obj

	case *payload.NFTDestroyFromSideChain:
		nftIDs := make([]string, 0)
		nftStatkeAddresses := make([]string, 0)
		for _, id := range object.IDs {
			nftIDs = append(nftIDs, id.ReversedString())
		}
		for _, sa := range object.OwnerStakeAddresses {
			addr, _ := sa.ToAddress()
			nftStatkeAddresses = append(nftStatkeAddresses, addr)
		}
		obj := DestroyNFTInfo{
			IDs:                 nftIDs,
			OwnerStakeAddresses: nftStatkeAddresses,
			GenesisBlockHash:    common.ToReversedString(object.GenesisBlockHash),
		}
		return obj

	}
	return nil
}

func getOutputPayloadInfo(op common2.OutputPayload) OutputPayloadInfo {
	switch object := op.(type) {
	case *outputpayload.CrossChainOutput:
		obj := new(CrossChainOutputInfo)
		obj.Version = object.Version
		obj.TargetAddress = object.TargetAddress
		obj.TargetAmount = object.TargetAmount.String()
		obj.TargetData = common.BytesToHexString(object.TargetData)
		return obj
	case *outputpayload.Withdraw:
		obj := new(WithdrawInfo)
		obj.Version = object.Version
		obj.GenesisBlockAddress = object.GenesisBlockAddress
		obj.SideChainTransactionHash = object.SideChainTransactionHash.String()
		obj.TargetData = common.BytesToHexString(object.TargetData)
		return obj
	case *outputpayload.ReturnSideChainDeposit:
		obj := new(ReturnSideChainDepositInfo)
		obj.Version = object.Version
		obj.GenesisBlockAddress = object.GenesisBlockAddress
		obj.DepositTransactionHash = common.ToReversedString(object.DepositTransactionHash)
		return obj
	case *outputpayload.DefaultOutput:
		obj := new(DefaultOutputInfo)
		return obj
	case *outputpayload.VoteOutput:
		obj := new(VoteOutputInfo)
		obj.Version = object.Version
		for _, content := range object.Contents {
			var contentInfo VoteContentInfo
			contentInfo.VoteType = content.VoteType
			switch contentInfo.VoteType {
			case outputpayload.Delegate:
				for _, cv := range content.CandidateVotes {
					contentInfo.CandidatesInfo = append(contentInfo.CandidatesInfo,
						CandidateVotes{
							Candidate: common.BytesToHexString(cv.Candidate),
							Votes:     cv.Votes.String(),
						})
				}
			case outputpayload.DposV2:
				for _, cv := range content.CandidateVotes {
					contentInfo.CandidatesInfo = append(contentInfo.CandidatesInfo,
						CandidateVotes{
							Candidate: common.BytesToHexString(cv.Candidate),
							Votes:     cv.Votes.String(),
						})
				}
			case outputpayload.CRC:
				for _, cv := range content.CandidateVotes {
					c, _ := common.Uint168FromBytes(cv.Candidate)
					addr, _ := c.ToAddress()
					contentInfo.CandidatesInfo = append(contentInfo.CandidatesInfo,
						CandidateVotes{
							Candidate: addr,
							Votes:     cv.Votes.String(),
						})
				}
			case outputpayload.CRCProposal:
				for _, cv := range content.CandidateVotes {
					c, _ := common.Uint256FromBytes(cv.Candidate)
					contentInfo.CandidatesInfo = append(contentInfo.CandidatesInfo,
						CandidateVotes{
							Candidate: common.ToReversedString(*c),
							Votes:     cv.Votes.String(),
						})
				}
			case outputpayload.CRCImpeachment:
				for _, cv := range content.CandidateVotes {
					c, _ := common.Uint168FromBytes(cv.Candidate)
					addr, _ := c.ToAddress()
					contentInfo.CandidatesInfo = append(contentInfo.CandidatesInfo,
						CandidateVotes{
							Candidate: addr,
							Votes:     cv.Votes.String(),
						})
				}
			}
			obj.Contents = append(obj.Contents, contentInfo)
		}
		return obj
	case *outputpayload.ExchangeVotesOutput:
		addr, _ := object.StakeAddress.ToAddress()
		obj := new(ExchangeVotesOutputInfo)
		obj.Version = object.Version
		obj.StakeAddress = addr
		return obj
	}

	return nil
}

func VerifyAndSendTx(tx interfaces.Transaction) error {
	// if transaction is verified unsuccessfully then will not put it into transaction pool
	if err := TxMemPool.AppendToTxPool(tx); err != nil {
		log.Warn("[httpjsonrpc] VerifyTransaction failed when AppendToTxnPool. Errcode:", err.Code())
		return err
	}

	// Relay tx inventory to other peers.
	txHash := tx.Hash()
	iv := msg.NewInvVect(msg.InvTypeTx, &txHash)
	Server.RelayInventory(iv, tx)

	return nil
}

type RPCTransaction struct {
	Address common.Uint168 `json:"address"`
	Txid    common.Uint256 `json:"txid"`
	Action  string         `json:"action"`
	Type    string         `json:"type"`
	Amount  uint64         `json:"amount"`
	Time    uint64         `json:"time"`
	Fee     uint64         `json:"fee"`
	Height  uint64         `json:"height"`
	Memo    string         `json:"memo"`
	Inputs  []string       `json:"inputs"`
	Outputs []string       `json:"outputs"`
}

type RPCTransactionHistoryInfo struct {
	TxHistory  interface{} `json:"txhistory"`
	TotalCount uint64      `json:"totalcount"`
}

func GetHistory(param Params) map[string]interface{} {
	address, ok := param.String("address")
	if !ok {
		return ResponsePack(InvalidParams, "")
	}

	_, err := common.Uint168FromAddress(address)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid address, "+err.Error())
	}

	order, ok := param.String("order")
	if ok {
		if order != "asc" && order != "desc" {
			return ResponsePack(InvalidParams, "")
		}
	} else {
		order = "desc"
	}
	skip, ok := param.Uint("skip")
	if !ok {
		skip = 0
	}
	limit, ok := param.Uint("limit")
	if !ok {
		limit = 10
	} else if limit > 50 {
		return ResponsePack(InvalidParams, "invalid limit")
	}
	timestamp, ok := param.Uint("timestamp")
	if !ok {
		timestamp = 0
	}
	txHistory, txCount := blockchain.StoreEx.GetTxHistoryByLimit(address, order, skip, limit, timestamp)

	result := RPCTransactionHistoryInfo{
		TxHistory:  txHistory,
		TotalCount: uint64(txCount),
	}

	return ResponsePack(Success, result)
}

func ResponsePack(errCode ServerErrCode, result interface{}) map[string]interface{} {
	if errCode != 0 && (result == "" || result == nil) {
		result = ErrMap[errCode]
	}
	return map[string]interface{}{"Result": result, "Error": errCode}
}

func checkRPCServiceLevel(level config.RPCServiceLevel) map[string]interface{} {
	if level < config.RPCServiceLevelFromString(ChainParams.RPCServiceLevel) {
		return ResponsePack(InvalidMethod,
			"requesting method if out of service level")
	}
	return nil
}
