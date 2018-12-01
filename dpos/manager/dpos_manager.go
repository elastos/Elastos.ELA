package manager

import (
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/p2p/msg"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/node"

	"github.com/elastos/Elastos.ELA.Utility/common"
	utip2p "github.com/elastos/Elastos.ELA.Utility/p2p"
	utimsg "github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

type DposNetwork interface {
	Initialize(proposalDispatcher ProposalDispatcher)

	Start()
	Stop() error

	SendMessageToPeer(id peer.PID, msg utip2p.Message) error
	BroadcastMessage(msg utip2p.Message)

	UpdatePeers(arbitrators [][]byte) error
	ChangeHeight(height uint32) error

	GetActivePeer() *peer.PID
}

type StatusSyncEventListener interface {
	OnPing(id peer.PID, height uint32)
	OnPong(id peer.PID, height uint32)
	OnBlock(id peer.PID, block *core.Block)
	OnInv(id peer.PID, blockHash common.Uint256)
	OnGetBlock(id peer.PID, blockHash common.Uint256)
	OnGetBlocks(id peer.PID, startBlockHeight, endBlockHeight uint32)
	OnResponseBlocks(id peer.PID, blockConfirms []*core.BlockConfirm)
	OnRequestConsensus(id peer.PID, height uint32)
	OnResponseConsensus(id peer.PID, status *msg.ConsensusStatus)
	OnRequestProposal(id peer.PID, hash common.Uint256)
	OnIllegalProposalReceived(id peer.PID, proposals *core.DposIllegalProposals)
	OnIllegalVotesReceived(id peer.PID, votes *core.DposIllegalVotes)
}

type NetworkEventListener interface {
	StatusSyncEventListener

	OnProposalReceived(id peer.PID, p core.DPosProposal)
	OnVoteReceived(id peer.PID, p core.DPosProposalVote)
	OnVoteRejected(id peer.PID, p core.DPosProposalVote)

	OnChangeView()
	OnBadNetwork()

	OnBlockReceived(b *core.Block, confirmed bool)
	OnConfirmReceived(p *core.DPosProposalVoteSlot)
}

type AbnormalRecovering interface {
	CollectConsensusStatus(height uint32, status *msg.ConsensusStatus) error
	RecoverFromConsensusStatus(status *msg.ConsensusStatus) error
}

type DposManager interface {
	NetworkEventListener

	GetPublicKey() string
	GetBlockCache() *ConsensusBlockCache

	Initialize(handler DposHandlerSwitch, dispatcher ProposalDispatcher, consensus Consensus, network DposNetwork)

	Recover()

	ProcessHigherBlock(b *core.Block)
	ConfirmBlock()

	ChangeConsensus(onDuty bool)
}

type dposManager struct {
	publicKey  string
	blockCache *ConsensusBlockCache

	handler    DposHandlerSwitch
	network    DposNetwork
	dispatcher ProposalDispatcher
	consensus  Consensus
}

func NewManager(name string) DposManager {
	m := &dposManager{
		publicKey:  name,
		blockCache: &ConsensusBlockCache{},
	}
	m.blockCache.Reset()

	return m
}

func (d *dposManager) Initialize(handler DposHandlerSwitch, dispatcher ProposalDispatcher, consensus Consensus, network DposNetwork) {
	d.handler = handler
	d.dispatcher = dispatcher
	d.consensus = consensus
	d.network = network
	d.blockCache.Listener = d.dispatcher.(*proposalDispatcher)
}

func (d *dposManager) GetPublicKey() string {
	return d.publicKey
}

func (d *dposManager) GetBlockCache() *ConsensusBlockCache {
	return d.blockCache
}

func (d *dposManager) Recover() {
	d.changeHeight()
	for {
		if peerID := d.network.GetActivePeer(); peerID != nil {
			d.handler.RequestAbnormalRecovering()
			return
		}

		time.Sleep(time.Second)
	}
}

func (d *dposManager) ProcessHigherBlock(b *core.Block) {
	log.Info("[ProcessHigherBlock] broadcast inv and try start new consensus")
	d.network.BroadcastMessage(msg.NewInventory(b.Hash()))
	d.handler.TryStartNewConsensus(b)
}

func (d *dposManager) ConfirmBlock() {
	d.handler.FinishConsensus()
}

func (d *dposManager) ChangeConsensus(onDuty bool) {
	d.handler.SwitchTo(onDuty)
}

func (d *dposManager) OnProposalReceived(id peer.PID, p core.DPosProposal) {
	log.Info("[OnProposalReceived] start")
	defer log.Info("[OnProposalReceived] end")

	d.handler.StartNewProposal(p)
}

func (d *dposManager) OnVoteReceived(id peer.PID, p core.DPosProposalVote) {
	log.Info("[OnVoteReceived] start")
	defer log.Info("[OnVoteReceived] end")
	d.handler.ProcessAcceptVote(id, p)
}

func (d *dposManager) OnVoteRejected(id peer.PID, p core.DPosProposalVote) {
	log.Info("[OnVoteRejected] start")
	defer log.Info("[OnVoteRejected] end")
	d.handler.ProcessRejectVote(id, p)
}

func (d *dposManager) OnPing(id peer.PID, height uint32) {
	d.processHeartBeat(id, height)
}

func (d *dposManager) OnPong(id peer.PID, height uint32) {
	d.processHeartBeat(id, height)
}

func (d *dposManager) OnBlock(id peer.PID, block *core.Block) {
	if block.Header.Height == blockchain.DefaultLedger.Blockchain.GetBestHeight()+1 {
		if _, err := node.LocalNode.AppendBlock(&core.BlockConfirm{
			BlockFlag: true,
			Block:     block,
		}); err != nil {
			log.Error("[OnBlock] err: ", err.Error())
		}
	}
}

func (d *dposManager) OnInv(id peer.PID, blockHash common.Uint256) {
	if _, err := getBlock(blockHash); err != nil {
		log.Info("[ProcessInv] send getblock:", blockHash.String())
		d.network.SendMessageToPeer(id, msg.NewGetBlock(blockHash))
	}
}

func (d *dposManager) OnGetBlock(id peer.PID, blockHash common.Uint256) {
	if block, err := getBlock(blockHash); err == nil {
		d.network.SendMessageToPeer(id, utimsg.NewBlock(block))
	}
}

func (d *dposManager) OnGetBlocks(id peer.PID, startBlockHeight, endBlockHeight uint32) {
	d.handler.ResponseGetBlocks(id, startBlockHeight, endBlockHeight)
}

func (d *dposManager) OnResponseBlocks(id peer.PID, blockConfirms []*core.BlockConfirm) {
	log.Info("[OnResponseBlocks] start")
	defer log.Info("[OnResponseBlocks] end")

	if err := blockchain.DefaultLedger.AppendBlocksAndConfirms(blockConfirms); err != nil {
		log.Error("Response blocks error: ", err)
	}
}

func (d *dposManager) OnRequestConsensus(id peer.PID, height uint32) {
	d.handler.HelpToRecoverAbnormal(id, height)
}

func (d *dposManager) OnResponseConsensus(id peer.PID, status *msg.ConsensusStatus) {
	log.Info("[OnResponseConsensus], status:", *status)
	d.handler.RecoverAbnormal(status)
}

func (d *dposManager) OnBadNetwork() {
	d.dispatcher.OnAbnormalStateDetected()
}

func (d *dposManager) OnChangeView() {
	d.consensus.TryChangeView()
}

func (d *dposManager) OnBlockReceived(b *core.Block, confirmed bool) {
	log.Info("[OnBlockReceived] start")
	defer log.Info("[OnBlockReceived] end")

	if confirmed {
		d.ConfirmBlock()
		d.changeHeight()
		log.Info("[OnBlockReceived] received confirmed block")
		return
	}

	if blockchain.DefaultLedger.Blockchain.BlockHeight < b.Height { //new height block coming
		d.ProcessHigherBlock(b)
	} else {
		log.Warn("a.Leger.LastBlock.Height", blockchain.DefaultLedger.Blockchain.BlockHeight, "b.Height", b.Height)
	}
}

func (d *dposManager) OnConfirmReceived(p *core.DPosProposalVoteSlot) {
	log.Info("[OnConfirmReceived] started, hash:", p.Hash)
	defer log.Info("[OnConfirmReceived] end")

	d.ConfirmBlock()
	d.changeHeight()
}

func (d *dposManager) OnIllegalProposalReceived(id peer.PID, proposals *core.DposIllegalProposals) {
	//todo complete me
}

func (d *dposManager) OnIllegalVotesReceived(id peer.PID, votes *core.DposIllegalVotes) {
	//todo complete me
}

func (d *dposManager) OnRequestProposal(id peer.PID, hash common.Uint256) {
	currentProposal := d.dispatcher.GetProcessingProposal()
	if currentProposal != nil {
		responseProposal := &msg.Proposal{Proposal: *currentProposal}
		d.network.SendMessageToPeer(id, responseProposal)
	}
}

func (d *dposManager) changeHeight() {
	if err := d.network.ChangeHeight(d.dispatcher.CurrentHeight()); err != nil {
		log.Error("Error occurred with change height: ", err)
		return
	}

	currentArbiter := blockchain.DefaultLedger.Arbitrators.GetOnDutyArbitrator()
	onDuty := d.publicKey == common.BytesToHexString(currentArbiter)

	if onDuty {
		log.Info("[onDutyArbitratorChanged] not onduty -> onduty")
	} else {
		log.Info("[onDutyArbitratorChanged] onduty -> not onduty")
	}
	d.ChangeConsensus(onDuty)
}

func (d *dposManager) processHeartBeat(id peer.PID, height uint32) {
	if d.tryRequestBlocks(id, height) {
		log.Info("Found higher block, requesting it.")
	}
}

func (d *dposManager) tryRequestBlocks(id peer.PID, sourceHeight uint32) bool {
	height := d.dispatcher.CurrentHeight()
	if sourceHeight > height {
		m := &msg.GetBlocks{
			StartBlockHeight: height,
			EndBlockHeight:   sourceHeight}
		d.network.SendMessageToPeer(id, m)

		return true
	}
	return false
}

func getBlock(blockHash common.Uint256) (*core.Block, error) {
	block, have := node.LocalNode.GetBlock(blockHash)
	if have {
		return block, nil
	}

	return blockchain.DefaultLedger.GetBlockWithHash(blockHash)
}
