package dpos

import (
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/account"
	"github.com/elastos/Elastos.ELA/dpos/dtime"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/manager"
	"github.com/elastos/Elastos.ELA/dpos/p2p"
	"github.com/elastos/Elastos.ELA/dpos/p2p/msg"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/dpos/store"
	ep2p "github.com/elastos/Elastos.ELA/p2p"
	emsg "github.com/elastos/Elastos.ELA/p2p/msg"
)

const dataPathDPoS = "elastos/data/dpos"

type blockItem struct {
	Block     *types.Block
	Confirmed bool
}

type messageItem struct {
	ID      peer.PID
	Message ep2p.Message
}

type network struct {
	listener           manager.NetworkEventListener
	proposalDispatcher *manager.ProposalDispatcher
	peersLock          sync.Mutex
	store              store.IDposStore
	publicKey          []byte

	p2pServer    p2p.Server
	messageQueue chan *messageItem
	quit         chan bool

	badNetworkChan           chan bool
	changeViewChan           chan bool
	recoverChan              chan bool
	recoverTimeoutChan       chan bool
	blockReceivedChan        chan blockItem
	confirmReceivedChan      chan *payload.Confirm
	illegalBlocksEvidence    chan *payload.DPOSIllegalBlocks
	sidechainIllegalEvidence chan *payload.SidechainIllegalData
	inactiveArbiters         chan *payload.InactiveArbitrators
}

func (n *network) Initialize(dnConfig manager.DPOSNetworkConfig) {
	n.proposalDispatcher = dnConfig.ProposalDispatcher
	n.store = dnConfig.Store
	n.publicKey = dnConfig.PublicKey
}

func (n *network) Start() {
	n.p2pServer.Start()

	go func() {
	out:
		for {
			select {
			case msgItem := <-n.messageQueue:
				n.processMessage(msgItem)
			case <-n.changeViewChan:
				n.changeView()
			case <-n.badNetworkChan:
				n.badNetwork()
			case <-n.recoverChan:
				n.recover()
			case <-n.recoverTimeoutChan:
				n.recoverTimeout()
			case blockItem := <-n.blockReceivedChan:
				n.blockReceived(blockItem.Block, blockItem.Confirmed)
			case confirm := <-n.confirmReceivedChan:
				n.confirmReceived(confirm)
			case evidence := <-n.illegalBlocksEvidence:
				n.illegalBlocksReceived(evidence)
			case evidence := <-n.inactiveArbiters:
				n.inactiveArbitersAccepeted(evidence)
			case sidechainEvidence := <-n.sidechainIllegalEvidence:
				n.sidechainIllegalEvidenceReceived(sidechainEvidence)
			case <-n.quit:
				break out
			}
		}
	}()
}

func (n *network) Stop() error {
	n.quit <- true
	return n.p2pServer.Stop()
}

func (n *network) UpdatePeers(peers []peer.PID) {
	log.Info("[UpdatePeers] peers:", len(peers), " height: ",
		blockchain.DefaultLedger.Blockchain.GetHeight())

	n.p2pServer.ConnectPeers(peers)
}

func (n *network) SendMessageToPeer(id peer.PID, msg ep2p.Message) error {
	return n.p2pServer.SendMessageToPeer(id, msg)
}

func (n *network) BroadcastMessage(msg ep2p.Message) {
	log.Info("[BroadcastMessage] msg:", msg.CMD())
	n.p2pServer.BroadcastMessage(msg)
}

func (n *network) GetActivePeers() []p2p.Peer {
	return n.p2pServer.ConnectedPeers()
}

func (n *network) PostChangeViewTask() {
	n.changeViewChan <- true
}

func (n *network) RecoverTimeout() {
	n.recoverTimeoutChan <- true
}

func (n *network) PostBlockReceivedTask(b *types.Block, confirmed bool) {
	n.blockReceivedChan <- blockItem{b, confirmed}
}

func (n *network) PostIllegalBlocksTask(p *payload.DPOSIllegalBlocks) {
	n.illegalBlocksEvidence <- p
}

func (n *network) PostSidechainIllegalDataTask(p *payload.SidechainIllegalData) {
	n.sidechainIllegalEvidence <- p
}

func (n *network) PostInactiveArbitersTask(p *payload.InactiveArbitrators) {
	n.inactiveArbiters <- p
}

func (n *network) PostConfirmReceivedTask(p *payload.Confirm) {
	n.confirmReceivedChan <- p
}

func (n *network) notifyFlag(flag p2p.NotifyFlag) {
	if flag == p2p.NFBadNetwork {
		n.badNetworkChan <- true
	}
}

func (n *network) handleMessage(pid peer.PID, msg ep2p.Message) {
	n.messageQueue <- &messageItem{pid, msg}
}

func (n *network) processMessage(msgItem *messageItem) {
	switch m := msgItem.Message.(type) {
	case *msg.Proposal:
		n.listener.OnProposalReceived(msgItem.ID, &m.Proposal)

	case *msg.VoteAccept:
		n.listener.OnVoteAccepted(msgItem.ID, &m.Payload)

	case *msg.VoteReject:
		n.listener.OnVoteRejected(msgItem.ID, &m.Payload)

	case *msg.Ping:
		n.listener.OnPing(msgItem.ID, uint32(m.Nonce))

	case *msg.Pong:
		n.listener.OnPong(msgItem.ID, uint32(m.Nonce))

	case *emsg.Block:
		n.listener.OnBlock(msgItem.ID, m.Serializable.(*types.Block))

	case *msg.Inventory:
		n.listener.OnInv(msgItem.ID, m.BlockHash)

	case *msg.GetBlock:
		n.listener.OnGetBlock(msgItem.ID, m.BlockHash)

	case *msg.GetBlocks:
		n.listener.OnGetBlocks(msgItem.ID, m.StartBlockHeight, m.EndBlockHeight)

	case *msg.ResponseBlocks:
		n.listener.OnResponseBlocks(msgItem.ID, m.BlockConfirms)

	case *msg.RequestConsensus:
		n.listener.OnRequestConsensus(msgItem.ID, m.Height)

	case *msg.ResponseConsensus:
		n.listener.OnResponseConsensus(msgItem.ID, &m.Consensus)

	case *msg.RequestProposal:
		n.listener.OnRequestProposal(msgItem.ID, m.ProposalHash)

	case *msg.IllegalProposals:
		n.listener.OnIllegalProposalReceived(msgItem.ID, &m.Proposals)

	case *msg.IllegalVotes:
		n.listener.OnIllegalVotesReceived(msgItem.ID, &m.Votes)

	case *msg.SidechainIllegalData:
		n.listener.OnSidechainIllegalEvidenceReceived(&m.Data)

	case *emsg.Tx:
		tx := m.Serializable.(*types.Transaction)
		if tx.IsInactiveArbitrators() {
			n.listener.OnInactiveArbitratorsReceived(msgItem.ID, tx)
		}

	case *msg.ResponseInactiveArbitrators:
		n.listener.OnResponseInactiveArbitratorsReceived(
			&m.TxHash, m.Signer, m.Sign)

	}
}

func (n *network) badNetwork() {
	n.listener.OnBadNetwork()
}

func (n *network) recover() {
	n.listener.OnRecover()
}

func (n *network) recoverTimeout() {
	n.listener.OnRecoverTimeout()
}

func (n *network) changeView() {
	n.listener.OnChangeView()
}

func (n *network) blockReceived(b *types.Block, confirmed bool) {
	n.listener.OnBlockReceived(b, confirmed)
}

func (n *network) confirmReceived(p *payload.Confirm) {
	n.listener.OnConfirmReceived(p)
}

func (n *network) illegalBlocksReceived(i *payload.DPOSIllegalBlocks) {
	n.listener.OnIllegalBlocksTxReceived(i)
}

func (n *network) inactiveArbitersAccepeted(p *payload.InactiveArbitrators) {
	n.listener.OnInactiveArbitratorsAccepted(p)
}

func (n *network) sidechainIllegalEvidenceReceived(
	s *payload.SidechainIllegalData) {
	n.BroadcastMessage(&msg.SidechainIllegalData{Data: *s})
	n.listener.OnSidechainIllegalEvidenceReceived(s)
}

func (n *network) getCurrentHeight(pid peer.PID) uint64 {
	return uint64(blockchain.DefaultLedger.Blockchain.GetHeight())
}

func NewDposNetwork(account account.Account, medianTime dtime.MedianTimeSource,
	localhost string, listener manager.NetworkEventListener) (*network, error) {
	network := &network{
		listener:                 listener,
		messageQueue:             make(chan *messageItem, 10000), //todo config handle capacity though config file
		quit:                     make(chan bool),
		badNetworkChan:           make(chan bool),
		changeViewChan:           make(chan bool),
		recoverChan:              make(chan bool),
		recoverTimeoutChan:       make(chan bool),
		blockReceivedChan:        make(chan blockItem, 10),        //todo config handle capacity though config file
		confirmReceivedChan:      make(chan *payload.Confirm, 10), //todo config handle capacity though config file
		illegalBlocksEvidence:    make(chan *payload.DPOSIllegalBlocks),
		sidechainIllegalEvidence: make(chan *payload.SidechainIllegalData),
		inactiveArbiters:         make(chan *payload.InactiveArbitrators),
	}

	notifier := p2p.NewNotifier(p2p.NFNetStabled|p2p.NFBadNetwork, network.notifyFlag)

	var pid peer.PID
	copy(pid[:], account.PublicKeyBytes())
	server, err := p2p.NewServer(&p2p.Config{
		DataDir:          dataPathDPoS,
		PID:              pid,
		EnableHub:        true,
		Localhost:        localhost,
		MagicNumber:      config.Parameters.DPoSConfiguration.Magic,
		DefaultPort:      config.Parameters.DPoSConfiguration.DPoSPort,
		TimeSource:       medianTime,
		MakeEmptyMessage: makeEmptyMessage,
		HandleMessage:    network.handleMessage,
		PingNonce:        network.getCurrentHeight,
		PongNonce:        network.getCurrentHeight,
		Sign:             account.Sign,
		StateNotifier:    notifier,
	})
	if err != nil {
		return nil, err
	}

	network.p2pServer = server
	return network, nil
}

func makeEmptyMessage(cmd string) (message ep2p.Message, err error) {
	switch cmd {
	case ep2p.CmdBlock:
		message = emsg.NewBlock(&types.Block{})
	case ep2p.CmdTx:
		message = emsg.NewTx(&types.Transaction{})
	case msg.CmdVoteAccept:
		message = &msg.VoteAccept{}
	case msg.CmdVoteReject:
		message = &msg.VoteReject{}
	case msg.CmdReceivedProposal:
		message = &msg.Proposal{}
	case msg.CmdInv:
		message = &msg.Inventory{}
	case msg.CmdGetBlock:
		message = &msg.GetBlock{}
	case msg.CmdGetBlocks:
		message = &msg.GetBlocks{}
	case msg.CmdResponseBlocks:
		message = &msg.ResponseBlocks{}
	case msg.CmdRequestConsensus:
		message = &msg.RequestConsensus{}
	case msg.CmdResponseConsensus:
		message = &msg.ResponseConsensus{}
	case msg.CmdRequestProposal:
		message = &msg.RequestProposal{}
	case msg.CmdIllegalProposals:
		message = &msg.IllegalProposals{}
	case msg.CmdIllegalVotes:
		message = &msg.IllegalVotes{}
	case msg.CmdSidechainIllegalData:
		message = &msg.SidechainIllegalData{}
	case msg.CmdResponseInactiveArbitrators:
		message = &msg.ResponseInactiveArbitrators{}
	default:
		return nil, errors.New("Received unsupported message, CMD " + cmd)
	}
	return message, nil
}
