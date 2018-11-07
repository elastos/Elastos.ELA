package node

import (
	"net"
	"sort"

	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	"github.com/elastos/Elastos.ELA.Utility/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/connmgr"
)

var (
	newNodes  = make(chan *node, protocol.DefaultMaxPeers)
	doneNodes = make(chan *node, protocol.DefaultMaxPeers)
	query     = make(chan interface{})
	quit      = make(chan struct{})
)

// nodeState maintains state of inbound, persistent, outbound peers as well
// as banned peers and outbound groups.
type nodeState struct {
	inboundNodes    map[uint64]*node
	outboundNodes   map[uint64]*node
	persistentNodes map[uint64]*node
	outboundGroups  map[string]int
}

// Count returns the count of all known peers.
func (ps *nodeState) Count() int {
	return len(ps.inboundNodes) + len(ps.outboundNodes) +
		len(ps.persistentNodes)
}

// forAllOutboundNodes is a helper function that runs closure on all outbound
// peers known to nodeState.
func (ps *nodeState) forAllOutboundNodes(closure func(n *node)) {
	for _, e := range ps.outboundNodes {
		closure(e)
	}
	for _, e := range ps.persistentNodes {
		closure(e)
	}
}

// forAllNodes is a helper function that runs closure on all peers known to
// nodeState.
func (ps *nodeState) forAllNodes(closure func(n *node)) {
	for _, e := range ps.inboundNodes {
		closure(e)
	}
	ps.forAllOutboundNodes(closure)
}

func inboundNodeConnected(conn net.Conn) {
	n := NewInboundNode(conn)
	go nodeDoneHandler(n)
}

func outboundNodeConnected(c *connmgr.ConnReq, conn net.Conn) {
	n, err := NewOutboundPeer(conn, c.Addr.String(), c.Permanent)
	if err != nil {
		log.Debugf("Cannot create outbound peer %s: %v", c.Addr, err)
		connManager.Disconnect(c.ID())
	}
	n.connReq = c
	n.SetState(protocol.HAND)
	n.SendMessage(NewVersion(LocalNode))
	go nodeDoneHandler(n)
	addrManager.Attempt(n.NA())
}

func nodeDoneHandler(n *node) {
	select {
	case <-n.quit:
		DoneNode(n)
	}
}

// handleAddNodeMsg deals with adding new peers.  It is invoked from the
// peerHandler goroutine.
func handleAddNodeMsg(state *nodeState, n *node) bool {
	if n == nil {
		return false
	}

	// Limit max number of total peers.
	if !n.persistent && state.Count() >= protocol.DefaultMaxPeers {
		log.Infof("Max peers reached [%d] - disconnecting peer %s",
			protocol.DefaultMaxPeers, n)
		n.Disconnect()
		return false
	}

	// Add the new peer and start it.
	log.Debugf("New peer %s", n)
	if n.inbound {
		state.inboundNodes[n.ID()] = n
	} else {
		state.outboundGroups[addrmgr.GroupKey(n.NA())]++
		if n.persistent {
			state.persistentNodes[n.ID()] = n
		} else {
			state.outboundNodes[n.ID()] = n
		}
	}

	return true
}

// handleDoneNodeMsg deals with peers that have signalled they are done.  It is
// invoked from the peerHandler goroutine.
func handleDoneNodeMsg(state *nodeState, n *node) {
	var list map[uint64]*node
	if n.persistent {
		list = state.persistentNodes
	} else if n.Inbound() {
		list = state.inboundNodes
	} else {
		list = state.outboundNodes
	}
	if _, ok := list[n.ID()]; ok {
		if !n.Inbound() && n.VersionKnown() {
			state.outboundGroups[addrmgr.GroupKey(n.NA())]--
		}
		if !n.Inbound() && n.connReq != nil {
			connManager.Disconnect(n.connReq.ID())
		}
		delete(list, n.ID())
		log.Debugf("Removed peer %s", n)
		return
	}

	if n.connReq != nil {
		connManager.Disconnect(n.connReq.ID())
	}

	// Update the address' last seen time if the peer has acknowledged
	// our version and has sent us its version as well.
	if n.VerAckReceived() && n.VersionKnown() && n.NA() != nil {
		addrManager.Connected(n.NA())
	}

	// If we get here it means that either we didn't know about the peer
	// or we purposefully deleted it.
}

type getNodeMsg struct {
	id    uint64
	reply chan struct {
		node protocol.Noder
		ok   bool
	}
}

type getNodesMsg struct {
	reply chan []protocol.Noder
}

type getNodeCountMsg struct {
	reply chan int
}

type getNodeHeightsMsg struct {
	reply chan []uint64
}

type getOutboundGroup struct {
	key   string
	reply chan int
}

type getSyncNodeMsg struct {
	reply chan protocol.Noder
}

type getBestNodeMsg struct {
	reply chan protocol.Noder
}

func handleGetNodeMsg(state *nodeState, msg getNodeMsg) {
	nd, ok := (*node)(nil), false
	state.forAllNodes(func(n *node) {
		if n.id == msg.id {
			nd, ok = n, true
		}
	})

	msg.reply <- struct {
		node protocol.Noder
		ok   bool
	}{node: nd, ok: ok}
}

func handleGetNodesMsg(state *nodeState, msg getNodesMsg) {
	nodes := make([]protocol.Noder, 0, state.Count())
	state.forAllNodes(func(n *node) {
		if n.Connected() {
			nodes = append(nodes, n)
		}
	})

	// Sort by node id before return
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID() < nodes[j].ID()
	})

	msg.reply <- nodes
}

func handleGetNodeCountMsg(state *nodeState, msg getNodeCountMsg) {
	connected := 0
	state.forAllNodes(func(n *node) {
		if n.Connected() {
			connected++
		}
	})
	msg.reply <- connected
}

func handleGetNodeHeightsMsg(state *nodeState, msg getNodeHeightsMsg) {
	nodes := make([]protocol.Noder, 0, state.Count())
	state.forAllNodes(func(n *node) {
		nodes = append(nodes, n)
	})

	// Sort by node id before return
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID() < nodes[j].ID()
	})

	heights := make([]uint64, 0, len(nodes))
	for _, n := range nodes {
		heights = append(heights, n.Height())
	}

	msg.reply <- heights
}

func handleGetOutboundGroup(state *nodeState, msg getOutboundGroup) {
	count, ok := state.outboundGroups[msg.key]
	if ok {
		msg.reply <- count
	} else {
		msg.reply <- 0
	}
}

func handleGetSyncNodeMsg(state *nodeState, msg getSyncNodeMsg) {
	sn := (protocol.Noder)(nil)
	state.forAllNodes(func(n *node) {
		if n.IsSyncHeaders() {
			sn = n
		}
	})

	msg.reply <- sn
}

func handleGetBestNodeMsg(state *nodeState, msg getBestNodeMsg) {
	var nodes = make([]*node, 0, state.Count())
	state.forAllNodes(func(n *node) {
		// Do not let external node become sync node
		if !n.IsExternal() && n.Connected() {
			nodes = append(nodes, n)
		}
	})

	// no node available
	if len(nodes) < 1 {
		msg.reply <- nil
		return
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].height > nodes[i].height
	})

	msg.reply <- nodes[0]
}

// nodeHandler is used to handle node operations such as adding and removing
// nodes to and from the server, banning nodes, and broadcasting messages to
// nodes.  It must be run in a goroutine.
func nodeHandler() {
	// Start the address manager, which are needed by nodes.  This is done
	// here since their lifecycle is closely tied  to this handler and rather
	// than adding more channels, it's easier and slightly faster to simply
	// start and stop them in this handler.
	addrManager.Start()

	state := &nodeState{
		inboundNodes:    make(map[uint64]*node),
		persistentNodes: make(map[uint64]*node),
		outboundNodes:   make(map[uint64]*node),
		outboundGroups:  make(map[string]int),
	}

	go connManager.Start()

out:
	for {
		select {
		// New nodes connected to the server.
		case p := <-newNodes:
			handleAddNodeMsg(state, p)

			// Disconnected nodes.
		case p := <-doneNodes:
			handleDoneNodeMsg(state, p)

		case qmsg := <-query:
			switch qmsg := qmsg.(type) {
			case getNodeMsg:
				handleGetNodeMsg(state, qmsg)

			case getNodesMsg:
				handleGetNodesMsg(state, qmsg)

			case getNodeCountMsg:
				handleGetNodeCountMsg(state, qmsg)

			case getNodeHeightsMsg:
				handleGetNodeHeightsMsg(state, qmsg)

			case getOutboundGroup:
				handleGetOutboundGroup(state, qmsg)

			case getSyncNodeMsg:
				handleGetSyncNodeMsg(state, qmsg)

			case getBestNodeMsg:
				handleGetBestNodeMsg(state, qmsg)

			}

		case <-quit:
			// Disconnect all nodes on server shutdown.
			state.forAllNodes(func(n *node) {
				n.Disconnect()
			})
			break out
		}
	}

	connManager.Stop()
	addrManager.Stop()

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-newNodes:
		case <-doneNodes:
		case <-query:
		default:
			break cleanup
		}
	}
}

func AddNode(node *node) {
	newNodes <- node
}

func DoneNode(node *node) {
	doneNodes <- node
}

func GetNeighborNode(id uint64) (protocol.Noder, bool) {
	reply := make(chan struct {
		node protocol.Noder
		ok   bool
	})
	query <- getNodeMsg{id: id, reply: reply}
	ret := <-reply
	return ret.node, ret.ok
}

func GetNeighborNodes() []protocol.Noder {
	reply := make(chan []protocol.Noder)
	query <- getNodesMsg{reply: reply}
	return <-reply
}

func GetNeighbourCount() int {
	reply := make(chan int)
	query <- getNodeCountMsg{reply: reply}
	return <-reply
}

func GetNeighborHeights() []uint64 {
	reply := make(chan []uint64)
	query <- getNodeHeightsMsg{reply: reply}
	return <-reply
}

// OutboundGroupCount returns the number of nodes connected to the given
// outbound group key.
func OutboundGroupCount(key string) int {
	reply := make(chan int)
	query <- getOutboundGroup{key: key, reply: reply}
	return <-reply
}

func IsNeighborNode(id uint64) bool {
	_, ok := GetNeighborNode(id)
	return ok
}

func GetSyncNode() protocol.Noder {
	reply := make(chan protocol.Noder)
	query <- getSyncNodeMsg{reply: reply}
	return <-reply
}

func GetBestNode() protocol.Noder {
	reply := make(chan protocol.Noder)
	query <- getBestNodeMsg{reply: reply}
	return <-reply
}
