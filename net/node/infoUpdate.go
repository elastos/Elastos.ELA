package node

import (
	"DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/ledger"
	"DNA_POW/events"
	. "DNA_POW/net/message"
	. "DNA_POW/net/protocol"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func keepAlive(from *Noder, dst *Noder) {
	// Need move to node function or keep here?
}

func (node *node) hasSyncPeer() (bool, Noder) {
	node.local.nbrNodes.RLock()
	defer node.local.nbrNodes.RUnlock()
	noders := node.local.GetNeighborNoder()
	for _, n := range noders {
		if n.IsSyncHeaders() == true {
			return true, n
		}
	}
	return false, nil
}

func (node *node) SyncBlkInNonCheckpointMode() {
	needSync := node.needSync()
	if needSync == false {
		node.local.SetSyncHeaders(false)
	} else {
		var syncNode Noder
		hasSyncPeer, syncNode := node.local.hasSyncPeer()
		if hasSyncPeer == false {
			syncNode = node.GetBestHeightNoder()
		} else {
			rb := syncNode.GetRequestBlockList()
			for k := range rb {
				if rb[k].Before(time.Now().Add(-3 * time.Second)) {
					ReqBlkData(syncNode, k)
				}
			}
		}
		hash := ledger.DefaultLedger.Store.GetCurrentBlockHash()
		blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)
		var emptyHash common.Uint256
		SendMsgSyncBlockHeaders(syncNode, blocator, emptyHash)
	}
}

func (node *node) SyncBlks() {
	if !node.GetStartSync() {
		return
	}
	needSync := node.needSync()
	log.Debug("needSync: ", needSync)
	if needSync == false {
		node.local.SetSyncHeaders(false)
		syncNode, err := node.FindSyncNode()
		if err == nil {
			syncNode.SetSyncHeaders(false)
		}
	} else {
		var syncNode Noder
		hasSyncPeer, syncNode := node.local.hasSyncPeer()
		if hasSyncPeer == false {
			syncNode = node.GetBestHeightNoder()
			hash := ledger.DefaultLedger.Store.GetCurrentBlockHash()
			if node.LocalNode().GetHeaderFisrtModeStatus() {
				SendMsgSyncHeaders(syncNode, hash)
			} else {
				blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)
				var emptyHash common.Uint256
				SendMsgSyncBlockHeaders(syncNode, blocator, emptyHash)
			}
		} else {
			rb := syncNode.GetRequestBlockList()
			if len(rb) == 0 {
				newSyncNode := node.GetBestHeightNoder()
				hash := ledger.DefaultLedger.Store.GetCurrentBlockHash()
				if node.LocalNode().GetHeaderFisrtModeStatus() {
					SendMsgSyncHeaders(newSyncNode, hash)
				} else {
					blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)
					var emptyHash common.Uint256
					SendMsgSyncBlockHeaders(newSyncNode, blocator, emptyHash)
				}
			} else {
				for k := range rb {
					if rb[k].Before(time.Now().Add(-3 * time.Second)) {
						log.Info("request block hash ", k)
						ReqBlkData(syncNode, k)
					}
				}
			}
		}
	}
}

func (node *node) SendPingToNbr() {
	noders := node.local.GetNeighborNoder()
	for _, n := range noders {
		if n.GetState() == ESTABLISH {
			buf, err := NewPingMsg()
			if err != nil {
				log.Error("failed build a new ping message")
			} else {
				go n.Tx(buf)
			}
		}
	}
}

func (node *node) HeartBeatMonitor() {
	noders := node.local.GetNeighborNoder()
	var periodUpdateTime uint
	if config.Parameters.GenBlockTime > config.MINGENBLOCKTIME {
		periodUpdateTime = config.Parameters.GenBlockTime / TIMESOFUPDATETIME
	} else {
		periodUpdateTime = config.DEFAULTGENBLOCKTIME / TIMESOFUPDATETIME
	}
	for _, n := range noders {
		if n.GetState() == ESTABLISH {
			t := n.GetLastRXTime()
			if t.Before(time.Now().Add(-1 * time.Second * time.Duration(periodUpdateTime) * KEEPALIVETIMEOUT)) {
				log.Warn("keepalive timeout!!!")
				n.SetState(INACTIVITY)
				n.CloseConn()
			}
		}
	}
}

func (node *node) ReqNeighborList() {
	buf, _ := NewMsg("getaddr", node.local)
	go node.Tx(buf)
}

func (node *node) ConnectSeeds() {
	if node.nbrNodes.GetConnectionCnt() < MINCONNCNT {
		seedNodes := config.Parameters.SeedList
		for _, nodeAddr := range seedNodes {
			found := false
			var n Noder
			var ip net.IP
			node.nbrNodes.Lock()
			for _, tn := range node.nbrNodes.List {
				addr := getNodeAddr(tn)
				ip = addr.IpAddr[:]
				addrstring := ip.To16().String() + ":" + strconv.Itoa(int(addr.Port))
				if nodeAddr == addrstring {
					n = tn
					found = true
					break
				}
			}
			node.nbrNodes.Unlock()
			if found {
				if n.GetState() == ESTABLISH {
					if node.LocalNode().NeedMoreAddresses() {
						n.ReqNeighborList()
					}
				}
			} else { //not found
				go node.Connect(nodeAddr)
			}
		}
	}
}

func (node *node) ConnectNode() {
	cntcount := node.nbrNodes.GetConnectionCnt()
	if cntcount < node.GetMaxOutboundCnt() {
		nbrAddr, _ := node.GetNeighborAddrs()
		addrs := node.RandGetAddresses(nbrAddr)
		for _, nodeAddr := range addrs {
			addr := nodeAddr.IpAddr
			port := nodeAddr.Port
			var ip net.IP
			ip = addr[:]
			na := ip.To16().String() + ":" + strconv.Itoa(int(port))
			go node.Connect(na)
		}
	}
}

func getNodeAddr(n *node) NodeAddr {
	var addr NodeAddr
	addr.IpAddr, _ = n.GetAddr16()
	addr.Time = n.GetTime()
	addr.Services = n.Services()
	addr.Port = n.GetPort()
	addr.ID = n.GetID()
	return addr
}

func (node *node) reconnect() {
	node.RetryConnAddrs.Lock()
	defer node.RetryConnAddrs.Unlock()
	lst := make(map[string]int)
	for addr := range node.RetryAddrs {
		node.RetryAddrs[addr] = node.RetryAddrs[addr] + 1
		rand.Seed(time.Now().UnixNano())
		log.Trace("Try to reconnect peer, peer addr is ", addr)
		<-time.After(time.Duration(rand.Intn(CONNMAXBACK)) * time.Millisecond)
		log.Trace("Back off time`s up, start connect node")
		node.Connect(addr)
		if node.RetryAddrs[addr] < MAXRETRYCOUNT {
			lst[addr] = node.RetryAddrs[addr]
		}
	}
	node.RetryAddrs = lst

}

func (n *node) TryConnect() {
	if n.fetchRetryNodeFromNeiborList() > 0 {
		n.reconnect()
	}
}

func (n *node) fetchRetryNodeFromNeiborList() int {
	n.nbrNodes.Lock()
	defer n.nbrNodes.Unlock()
	var ip net.IP
	neibornodes := make(map[uint64]*node)
	for _, tn := range n.nbrNodes.List {
		addr := getNodeAddr(tn)
		ip = addr.IpAddr[:]
		nodeAddr := ip.To16().String() + ":" + strconv.Itoa(int(addr.Port))
		if tn.GetState() == INACTIVITY {
			//add addr to retry list
			n.AddInRetryList(nodeAddr)
			//close legacy node
			if tn.conn != nil {
				tn.CloseConn()
			}
		} else {
			//add others to tmp node map
			n.RemoveFromRetryList(nodeAddr)
			neibornodes[tn.GetID()] = tn
		}
	}
	n.nbrNodes.List = neibornodes
	return len(n.RetryAddrs)
}

// FIXME part of node info update function could be a node method itself intead of
// a node map method
// Fixme the Nodes should be a parameter
func (node *node) updateNodeInfo() {
	var periodUpdateTime uint
	if config.Parameters.GenBlockTime > config.MINGENBLOCKTIME {
		periodUpdateTime = config.Parameters.GenBlockTime / TIMESOFUPDATETIME
	} else {
		periodUpdateTime = config.DEFAULTGENBLOCKTIME / TIMESOFUPDATETIME
	}
	ticker := time.NewTicker(time.Second * (time.Duration(periodUpdateTime)) * 2)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			node.SendPingToNbr()
			node.SyncBlks()
			//node.SyncBlkInNonCheckpointMode()
			node.HeartBeatMonitor()
		case <-quit:
			ticker.Stop()
			return
		}
	}
	// TODO when to close the timer
	//close(quit)
}

func (node *node) CheckConnCnt() {
	//compare if connect count is larger than DefaultMaxPeers, disconnect one of the connection
	if node.nbrNodes.GetConnectionCnt() > node.GetDefaultMaxPeers() {
		disconnNode := node.RandGetANbr()
		node.eventQueue.GetEvent("disconnect").Notify(events.EventNodeDisconnect, disconnNode)
	}
}

func (node *node) updateConnection() {
	t := time.NewTimer(time.Second * CONNMONITOR)
	for {
		select {
		case <-t.C:
			node.ConnectSeeds()
			//node.TryConnect()
			node.ConnectNode()
			node.CheckConnCnt()
			t.Stop()
			t.Reset(time.Second * CONNMONITOR)
		}
	}
}
