package node

import (
	"time"

	"github.com/elastos/Elastos.ELA/log"

	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
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

func printSyncState() {
	isCurrent := IsCurrent()
	log.Info("needSync: ", !isCurrent)
	chain.DumpState()
	log.Info("[", len(chain.Index), len(chain.BlockCache), len(chain.Orphans), "]")
	if isCurrent && syncNode != nil {
		syncNode.stallTimer.stop()
		syncNode = nil
	}
}

func stopSyncing() {
	if syncNode != nil {
		syncNode.Disconnect()
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
			node.SendMessage(msg.NewPing(uint64(store.GetHeight())))

		case <-node.quit:
			break out
		}
	}
}
