package manager

import (
	"bytes"
	"errors"
	"time"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/dpos/account"
	"github.com/elastos/Elastos.ELA/dpos/log"
	msg2 "github.com/elastos/Elastos.ELA/dpos/p2p/msg"
)

type ProposalDispatcher interface {
	AbnormalRecovering

	//status
	GetProcessingBlock() *types.Block
	GetProcessingProposal() *types.DPosProposal
	IsProcessingBlockEmpty() bool
	CurrentHeight() uint32

	//proposal
	StartProposal(b *types.Block)
	CleanProposals(changeView bool)
	FinishProposal()
	TryStartSpeculatingProposal(b *types.Block)
	ProcessProposal(d types.DPosProposal)

	FinishConsensus()

	ProcessVote(v types.DPosProposalVote, accept bool)
	AddPendingVote(v types.DPosProposalVote)

	OnAbnormalStateDetected()
	RequestAbnormalRecovering()
	TryAppendAndBroadcastConfirmBlockMsg() bool
}

type proposalDispatcher struct {
	processingBlock    *types.Block
	processingProposal *types.DPosProposal
	acceptVotes        map[common.Uint256]types.DPosProposalVote
	rejectedVotes      map[common.Uint256]types.DPosProposalVote
	pendingProposals   map[common.Uint256]types.DPosProposal
	pendingVotes       map[common.Uint256]types.DPosProposalVote

	illegalMonitor IllegalBehaviorMonitor
	eventMonitor   *log.EventMonitor
	consensus      Consensus
	network        DposNetwork
	manager        DposManager
	account        account.DposAccount
}

func (p *proposalDispatcher) OnAbnormalStateDetected() {
	p.RequestAbnormalRecovering()
}

func (p *proposalDispatcher) RequestAbnormalRecovering() {
	height := p.CurrentHeight()
	msgItem := &msg2.RequestConsensus{Height: height}
	peerID := p.network.GetActivePeer()
	if peerID == nil {
		log.Error("[RequestAbnormalRecovering] can not find active peer")
		return
	}
	p.network.SendMessageToPeer(*peerID, msgItem)
}

func (p *proposalDispatcher) GetProcessingBlock() *types.Block {
	return p.processingBlock
}

func (p *proposalDispatcher) GetProcessingProposal() *types.DPosProposal {
	return p.processingProposal
}

func (p *proposalDispatcher) ProcessVote(v types.DPosProposalVote, accept bool) {
	log.Info("[ProcessVote] start")
	defer log.Info("[ProcessVote] end")

	if !blockchain.IsVoteValid(&v) {
		log.Info("Invalid vote")
		return
	}

	if p.alreadyExistVote(v) {
		log.Info("Already has vote")
		return
	}

	if anotherVote, legal := p.illegalMonitor.IsLegalVote(&v); !legal {
		p.illegalMonitor.ProcessIllegalVote(&v, anotherVote)
		return
	}

	if accept {
		p.countAcceptedVote(v)
	} else {
		p.countRejectedVote(v)
	}
}

func (p *proposalDispatcher) AddPendingVote(v types.DPosProposalVote) {
	p.pendingVotes[v.Hash()] = v
}

func (p *proposalDispatcher) IsProcessingBlockEmpty() bool {
	return p.processingBlock == nil
}

func (p *proposalDispatcher) StartProposal(b *types.Block) {
	log.Info("[StartProposal] start")
	defer log.Info("[StartProposal] end")

	if p.processingBlock != nil {
		log.Info("[StartProposal] start proposal failed")
		return
	}
	p.processingBlock = b

	p.network.BroadcastMessage(msg2.NewInventory(b.Hash()))
	proposal := types.DPosProposal{Sponsor: p.manager.GetPublicKey(), BlockHash: b.Hash(), ViewOffset: p.consensus.GetViewOffset()}
	var err error
	proposal.Sign, err = p.account.SignProposal(&proposal)
	if err != nil {
		log.Error("[StartProposal] start proposal failed:", err.Error())
		return
	}

	log.Info("[StartProposal] sponsor:", p.manager.GetPublicKey())

	m := &msg2.Proposal{
		Proposal: proposal,
	}

	log.Info("[StartProposal] send proposal message finished, Proposal Hash: ", msg2.GetMessageHash(m))
	p.network.BroadcastMessage(m)

	rawData := new(bytes.Buffer)
	proposal.Serialize(rawData)
	proposalEvent := log.ProposalEvent{
		Proposal:     proposal.Sponsor,
		BlockHash:    proposal.BlockHash,
		ReceivedTime: time.Now(),
		ProposalHash: proposal.Hash(),
		RawData:      rawData.Bytes(),
		Result:       false,
	}
	p.eventMonitor.OnProposalArrived(&proposalEvent)

	p.acceptProposal(proposal)
}

func (p *proposalDispatcher) TryStartSpeculatingProposal(b *types.Block) {
	log.Info("[TryStartSpeculatingProposal] start")
	defer log.Info("[TryStartSpeculatingProposal] end")

	if p.processingBlock != nil {
		log.Warn("[TryStartSpeculatingProposal] processingBlock is not nil")
		return
	}
	p.processingBlock = b
}

func (p *proposalDispatcher) FinishProposal() {
	log.Info("[FinishProposal] start")
	defer log.Info("[FinishProposal] end")

	if p.processingBlock == nil {
		log.Warn("[FinishProposal] nil processing block")
		return
	}

	proposal, blockHash := p.processingProposal.Sponsor, p.processingBlock.Hash()

	if !p.TryAppendAndBroadcastConfirmBlockMsg() {
		log.Warn("Add block failed, no need to broadcast confirm message")
		return
	}

	p.FinishConsensus()

	proposalEvent := log.ProposalEvent{
		Proposal:  proposal,
		BlockHash: blockHash,
		EndTime:   time.Now(),
		Result:    true,
	}
	p.eventMonitor.OnProposalFinished(&proposalEvent)
}

func (p *proposalDispatcher) CleanProposals(changeView bool) {
	log.Info("Clean proposals")

	//todo clear pending proposals that are lower than current consensus height
	p.illegalMonitor.Reset(changeView)

	p.processingBlock = nil
	p.processingProposal = nil
	p.acceptVotes = make(map[common.Uint256]types.DPosProposalVote)
	p.rejectedVotes = make(map[common.Uint256]types.DPosProposalVote)
	p.pendingVotes = make(map[common.Uint256]types.DPosProposalVote)
}

func (p *proposalDispatcher) ProcessProposal(d types.DPosProposal) {
	log.Info("[ProcessProposal] start")
	defer log.Info("[ProcessProposal] end")

	if p.processingProposal != nil && d.BlockHash.IsEqual(p.processingProposal.Hash()) {
		log.Info("Already processing processing")
		return
	}

	if _, ok := p.pendingProposals[d.Hash()]; ok {
		log.Info("Already have proposal, wait for processing")
		return
	}

	if !blockchain.IsProposalValid(&d) {
		log.Warn("Invalid proposal.")
		return
	}

	p.illegalMonitor.AddProposal(d)
	if anotherProposal, ok := p.illegalMonitor.IsLegalProposal(&d); !ok {
		p.illegalMonitor.ProcessIllegalProposal(&d, anotherProposal)
		return
	}

	if !p.consensus.IsArbitratorOnDuty(d.Sponsor) {
		currentArbiter := p.manager.GetArbitrators().GetNextOnDutyArbitrator(p.consensus.GetViewOffset())
		log.Info("viewOffset:", p.consensus.GetViewOffset(), "current arbiter:",
			common.BytesToHexString(currentArbiter), "sponsor:", d.Sponsor)
		p.rejectProposal(d)
		log.Warn("reject: current arbiter is not sponsor")
		return
	}

	currentBlock, ok := p.manager.GetBlockCache().TryGetValue(d.BlockHash)
	if !ok || !p.consensus.IsRunning() {
		p.pendingProposals[d.Hash()] = d
		log.Info("Received pending proposal.")
		return
	} else {
		p.TryStartSpeculatingProposal(currentBlock)
	}

	if currentBlock.Height != p.processingBlock.Height {
		log.Warn("[ProcessProposal] Invalid block height")
		return
	}

	if !d.BlockHash.IsEqual(p.processingBlock.Hash()) {
		log.Warn("[ProcessProposal] Invalid block hash")
		return
	}

	p.acceptProposal(d)
}

func (p *proposalDispatcher) TryAppendAndBroadcastConfirmBlockMsg() bool {
	currentVoteSlot := &types.DPosProposalVoteSlot{
		Hash:     p.processingBlock.Hash(),
		Proposal: *p.processingProposal,
		Votes:    make([]types.DPosProposalVote, 0),
	}
	for _, v := range p.acceptVotes {
		currentVoteSlot.Votes = append(currentVoteSlot.Votes, v)
	}

	log.Info("[TryAppendAndBroadcastConfirmBlockMsg] append confirm.")
	p.manager.Relay(nil, &types.BlockConfirm{
		ConfirmFlag: true,
		Confirm:     currentVoteSlot,
	})
	if err := p.manager.AppendConfirm(currentVoteSlot); err != nil {
		log.Error("[AppendConfirm] err:", err.Error())
		return false
	}

	return true
}

func (p *proposalDispatcher) OnBlockAdded(b *types.Block) {

	if p.consensus.IsRunning() {
		for k, v := range p.pendingProposals {
			if v.BlockHash.IsEqual(b.Hash()) {
				p.ProcessProposal(v)
				delete(p.pendingProposals, k)
				break
			}
		}
	}
}

func (p *proposalDispatcher) FinishConsensus() {
	if p.consensus.IsRunning() {
		log.Info("[FinishConsensus] start")
		defer log.Info("[FinishConsensus] end")

		c := log.ConsensusEvent{EndTime: time.Now(), Height: p.CurrentHeight()}
		p.eventMonitor.OnConsensusFinished(&c)
		p.consensus.SetReady()
		p.CleanProposals(false)
	}
}

func (p *proposalDispatcher) CollectConsensusStatus(height uint32, status *msg2.ConsensusStatus) error {
	if height > p.CurrentHeight() {
		return errors.New("Requesting height greater than current processing height")
	}

	status.AcceptVotes = make([]types.DPosProposalVote, 0, len(p.acceptVotes))
	for _, v := range p.acceptVotes {
		status.AcceptVotes = append(status.AcceptVotes, v)
	}

	status.RejectedVotes = make([]types.DPosProposalVote, 0, len(p.rejectedVotes))
	for _, v := range p.rejectedVotes {
		status.RejectedVotes = append(status.RejectedVotes, v)
	}

	status.PendingProposals = make([]types.DPosProposal, 0, len(p.pendingProposals))
	for _, v := range p.pendingProposals {
		status.PendingProposals = append(status.PendingProposals, v)
	}

	status.PendingVotes = make([]types.DPosProposalVote, 0, len(p.pendingVotes))
	for _, v := range p.pendingVotes {
		status.PendingVotes = append(status.PendingVotes, v)
	}

	return nil
}

func (p *proposalDispatcher) RecoverFromConsensusStatus(status *msg2.ConsensusStatus) error {
	p.acceptVotes = make(map[common.Uint256]types.DPosProposalVote)
	for _, v := range status.AcceptVotes {
		p.acceptVotes[v.Hash()] = v
	}

	p.rejectedVotes = make(map[common.Uint256]types.DPosProposalVote)
	for _, v := range status.RejectedVotes {
		p.rejectedVotes[v.Hash()] = v
	}

	p.pendingProposals = make(map[common.Uint256]types.DPosProposal)
	for _, v := range status.PendingProposals {
		p.pendingProposals[v.Hash()] = v
	}

	p.pendingVotes = make(map[common.Uint256]types.DPosProposalVote)
	for _, v := range status.PendingVotes {
		p.pendingVotes[v.Hash()] = v
	}

	return nil
}

func (p *proposalDispatcher) CurrentHeight() uint32 {
	var height uint32
	currentBlock := p.GetProcessingBlock()
	if currentBlock != nil {
		height = currentBlock.Height
	} else {
		height = blockchain.DefaultLedger.Blockchain.BlockHeight
	}
	return height
}

func (p *proposalDispatcher) alreadyExistVote(v types.DPosProposalVote) bool {
	_, ok := p.acceptVotes[v.Hash()]
	if ok {
		log.Info("[alreadyExistVote]: ", v.Signer, "already in the AcceptVotes!")
		return true
	}

	_, ok = p.rejectedVotes[v.Hash()]
	if ok {
		log.Info("[alreadyExistVote]: ", v.Signer, "already in the RejectedVotes!")
		return true
	}

	return false
}

func (p *proposalDispatcher) countAcceptedVote(v types.DPosProposalVote) {
	log.Info("[countAcceptedVote] start")
	defer log.Info("[countAcceptedVote] end")

	if v.Accept {
		log.Info("[countAcceptedVote] Received needed sign, collect it into AcceptVotes!")
		p.acceptVotes[v.Hash()] = v

		if p.manager.GetArbitrators().HasArbitersMajorityCount(uint32(len(p.acceptVotes))) {
			log.Info("Collect majority signs, finish proposal.")
			p.FinishProposal()
		}
	}
}

func (p *proposalDispatcher) countRejectedVote(v types.DPosProposalVote) {
	log.Info("[countRejectedVote] start")
	defer log.Info("[countRejectedVote] end")

	if !v.Accept {
		log.Info("[countRejectedVote] Received invalid sign, collect it into RejectedVotes!")
		p.rejectedVotes[v.Hash()] = v

		if p.manager.GetArbitrators().HasArbitersMinorityCount(uint32(len(p.rejectedVotes))) {
			p.CleanProposals(true)
			p.consensus.ChangeView()
		}
	}
}

func (p *proposalDispatcher) acceptProposal(d types.DPosProposal) {
	log.Info("[acceptProposal] start")
	defer log.Info("[acceptProposal] end")

	p.setProcessingProposal(d)
	vote := types.DPosProposalVote{ProposalHash: d.Hash(), Signer: p.manager.GetPublicKey(), Accept: true}
	var err error
	vote.Sign, err = p.account.SignVote(&vote)
	if err != nil {
		log.Error("[acceptProposal] sign failed")
		return
	}
	voteMsg := &msg2.Vote{Command: msg2.CmdAcceptVote, Vote: vote}
	p.ProcessVote(vote, true)

	p.network.BroadcastMessage(voteMsg)
	log.Info("[acceptProposal] send acc_vote msg:", msg2.GetMessageHash(voteMsg).String())

	rawData := new(bytes.Buffer)
	vote.Serialize(rawData)
	voteEvent := log.VoteEvent{Signer: vote.Signer, ReceivedTime: time.Now(), Result: true, RawData: rawData.Bytes()}
	p.eventMonitor.OnVoteArrived(&voteEvent)
}

func (p *proposalDispatcher) rejectProposal(d types.DPosProposal) {
	p.setProcessingProposal(d)

	vote := types.DPosProposalVote{ProposalHash: d.Hash(), Signer: p.manager.GetPublicKey(), Accept: false}
	var err error
	vote.Sign, err = p.account.SignVote(&vote)
	if err != nil {
		log.Error("[rejectProposal] sign failed")
		return
	}
	msg := &msg2.Vote{Command: msg2.CmdRejectVote, Vote: vote}
	log.Info("[rejectProposal] send rej_vote msg:", msg2.GetMessageHash(msg))

	_, ok := p.manager.GetBlockCache().TryGetValue(d.BlockHash)
	if !ok {
		log.Error("[rejectProposal] can't find block")
		return
	}
	p.ProcessVote(vote, false)
	p.network.BroadcastMessage(msg)

	rawData := new(bytes.Buffer)
	vote.Serialize(rawData)
	voteEvent := log.VoteEvent{Signer: vote.Signer, ReceivedTime: time.Now(), Result: false, RawData: rawData.Bytes()}
	p.eventMonitor.OnVoteArrived(&voteEvent)
}

func (p *proposalDispatcher) setProcessingProposal(d types.DPosProposal) {
	p.processingProposal = &d

	for _, v := range p.pendingVotes {
		if v.ProposalHash.IsEqual(d.Hash()) {
			p.ProcessVote(v, v.Accept)
		}
	}
	p.pendingVotes = make(map[common.Uint256]types.DPosProposalVote)
}

func NewDispatcherAndIllegalMonitor(consensus Consensus, eventMonitor *log.EventMonitor, network DposNetwork, manager DposManager, dposAccount account.DposAccount) (ProposalDispatcher, IllegalBehaviorMonitor) {
	p := &proposalDispatcher{
		processingBlock:    nil,
		processingProposal: nil,
		acceptVotes:        make(map[common.Uint256]types.DPosProposalVote),
		rejectedVotes:      make(map[common.Uint256]types.DPosProposalVote),
		pendingProposals:   make(map[common.Uint256]types.DPosProposal),
		pendingVotes:       make(map[common.Uint256]types.DPosProposalVote),
		eventMonitor:       eventMonitor,
		consensus:          consensus,
		network:            network,
		manager:            manager,
		account:            dposAccount,
	}
	i := &illegalBehaviorMonitor{
		dispatcher:      p,
		cachedProposals: make(map[common.Uint256]*types.DPosProposal),
		evidenceCache:   evidenceCache{make(map[common.Uint256]types.DposIllegalData)},
		manager:         manager,
	}
	p.illegalMonitor = i
	return p, i
}
