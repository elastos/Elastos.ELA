package node

import (
	"time"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/log"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg/v0"
)

type stallTimer struct {
	timeout    time.Duration
	lastUpdate time.Time
	quit       chan struct{}
	onTimeout  func()
}

func newSyncTimer(onTimeout func()) *stallTimer {
	return &stallTimer{
		timeout:   syncBlockTimeout,
		onTimeout: onTimeout,
	}
}

func (t *stallTimer) start() {
	go func() {
		t.quit = make(chan struct{}, 1)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Now().After(t.lastUpdate.Add(t.timeout)) {
					t.onTimeout()
					goto QUIT
				}
			case <-t.quit:
				goto QUIT
			}
		}
	QUIT:
		t.quit = nil
	}()
}

func (t *stallTimer) update() {
	t.lastUpdate = time.Now()
}

func (t *stallTimer) stop() {
	if t.quit != nil {
		t.quit <- struct{}{}
	}
}

func (node *node) SyncBlocks() {
	needSync := !IsCurrent()
	log.Info("needSync: ", needSync)
	log.Info("BlockHeight = ", chain.DefaultLedger.Blockchain.BlockHeight)
	chain.DefaultLedger.Blockchain.DumpState()
	bc := chain.DefaultLedger.Blockchain
	log.Info("[", len(bc.Index), len(bc.BlockCache), len(bc.Orphans), "]")
	if needSync {
		syncNode := GetSyncNode()
		if syncNode == nil {
			LocalNode.ResetRequestedBlock()
			syncNode = GetBestNode()
			if syncNode == nil {
				return
			}
			hash := chain.DefaultLedger.Store.GetCurrentBlockHash()
			locator := chain.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)

			SendGetBlocks(syncNode, locator, EmptyHash)
			LocalNode.SetSyncHeaders(true)
			syncNode.SetSyncHeaders(true)
			// Start sync timer
			LocalNode.stallTimer.start()
		} else if syncNode.Version() < p2p.EIP001Version {
			list := LocalNode.GetRequestBlockList()
			var requests = make(map[Uint256]time.Time)
			node.requestedBlockLock.Lock()
			for i, v := range list {
				requests[i] = v
				if len(requests) >= p2p.MaxHeaderHashes {
					break
				}
			}
			node.requestedBlockLock.Unlock()
			if len(requests) == 0 {
				syncNode.SetSyncHeaders(false)
				LocalNode.SetStartHash(EmptyHash)
				LocalNode.SetStopHash(EmptyHash)
				syncNode := GetBestNode()
				if syncNode == nil {
					return
				}
				hash := chain.DefaultLedger.Store.GetCurrentBlockHash()
				locator := chain.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)

				SendGetBlocks(syncNode, locator, EmptyHash)
			} else {
				for hash, t := range requests {
					if time.Now().After(t.Add(syncBlockTimeout)) {
						log.Infof("request block hash %x ", hash.Bytes())
						LocalNode.AddRequestedBlock(hash)
						syncNode.SendMessage(v0.NewGetData(hash))
					}
				}
			}
		}
	} else {
		stopSyncing()
	}
}

func stopSyncing() {
	// Stop sync timer
	LocalNode.stallTimer.stop()
	LocalNode.SetSyncHeaders(false)
	LocalNode.SetStartHash(EmptyHash)
	LocalNode.SetStopHash(EmptyHash)
	syncNode := GetSyncNode()
	if syncNode != nil {
		syncNode.SetSyncHeaders(false)
	}
}

func (node *node) pingHandler() {
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

out:
	for {
		select {
		case <-pingTicker.C:

			// send ping message to node
			node.SendMessage(msg.NewPing(uint64(chain.DefaultLedger.Store.GetHeight())))

		case <-node.quit:
			break out
		}
	}
}
