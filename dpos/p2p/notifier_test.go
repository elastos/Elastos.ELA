package p2p

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/stretchr/testify/assert"
)

func TestNotifier(t *testing.T) {
	test.SkipShort(t)
	// Start peer-to-peer server
	pid := peer.PID{}

	notifyChan := make(chan NotifyFlag)
	notifier := NewNotifier(NFNetStabled|NFBadNetwork, func(flag NotifyFlag) {
		notifyChan <- flag
	})

	peers := 90
	portBase := 40000
	peerList := make([]peer.PID, 0, peers)
	addrList := sync.Map{}
	for i := 0; i < peers; i++ {
		rand.Read(pid[:])
		port := portBase + i
		peerList = append(peerList, pid)
		addrList.Store(pid, fmt.Sprintf("localhost:%d", port))
	}

	priKey, pubKey, _ := crypto.GenerateKeyPair()
	ePubKey, _ := pubKey.EncodePoint(true)
	copy(pid[:], ePubKey)
	server, err := NewServer(&Config{
		PID:             pid,
		MagicNumber:     123123,
		ProtocolVersion: 0,
		Services:        0,
		DefaultPort:     20338,
		Sign: func(nonce []byte) []byte {
			sign, _ := crypto.Sign(priKey, nonce)
			return sign
		},
		MakeEmptyMessage: makeEmptyMessage,
		StateNotifier:    notifier,
		GetAddr: func(pid peer.PID) (addr string, ok bool) {
			o, ok := addrList.Load(pid)
			return o.(string), ok
		},
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	server.Start()
	defer server.Stop()

	server.ConnectPeers(peerList)

	// Mock peers not started, wait for connection timeout.
	select {
	case f := <-notifyChan:
		if !assert.Equal(t, NFBadNetwork, f) {
			t.FailNow()
		}
	case <-time.After(time.Second * 31):
		t.Fatalf("Connect peers timeout")
	}

	// Start mock peers and connect them.
	peerChan := make(chan *peer.Peer, peers)
	peerList = peerList[:0]
	for i := 0; i < peers; i++ {
		priKey, pubKey, _ := crypto.GenerateKeyPair()
		ePubKey, _ := pubKey.EncodePoint(true)
		copy(pid[:], ePubKey)
		port := portBase + i
		peerList = append(peerList, pid)
		addrList.Store(pid, fmt.Sprintf("localhost:%d", port))
		err := mockRemotePeer(pid, priKey, uint16(port), peerChan, nil)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}
	server.ConnectPeers(peerList)

	// Wait for network stable notify.
	select {
	case f := <-notifyChan:
		if !assert.Equal(t, NFNetStabled, f) {
			t.FailNow()
		}
	case <-time.After(time.Second * 10):
		t.Fatalf("Connect peers timeout")
	}

	// Disconnect 31 peers to trigger bad network notify
	for i := 0; i < 31; i++ {
		p := <-peerChan
		p.Disconnect()
	}
	select {
	case f := <-notifyChan:
		if !assert.Equal(t, NFBadNetwork, f) {
			t.FailNow()
		}
	case <-time.After(time.Second * 10):
		t.Fatalf("Connect peers timeout")
	}

	// Reconnection will trigger another network stable notify.
	select {
	case f := <-notifyChan:
		if !assert.Equal(t, NFNetStabled, f) {
			t.FailNow()
		}
	case <-time.After(time.Second * 10):
		t.Fatalf("Connect peers timeout")
	}
}
