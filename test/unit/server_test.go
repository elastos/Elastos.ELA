// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common/config"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/elanet"
	"github.com/elastos/Elastos.ELA/elanet/netsync"
	"github.com/elastos/Elastos.ELA/elanet/routes"
	"github.com/elastos/Elastos.ELA/p2p/peer"
	svr "github.com/elastos/Elastos.ELA/p2p/server"
	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction2.GetTransaction
	functions.GetTransactionByBytes = transaction2.GetTransactionByBytes
	functions.CreateTransaction = transaction2.CreateTransaction
	functions.GetTransactionParameters = transaction2.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

// iPeer fakes a server.IPeer for test.
type iPeer struct {
	*peer.Peer
}

func (p *iPeer) ToPeer() *peer.Peer {
	return p.Peer
}

func (p *iPeer) AddBanScore(persistent, transient uint32, reason string) {}

func (p *iPeer) BanScore() uint32 { return 0 }

// mockPeer creates a fake server.IPeer instance.
func mockPeer() svr.IPeer {
	return &iPeer{Peer: &peer.Peer{}}
}

// TestHandlePeerMsg tests the adding/removing peer messages.
func TestHandlePeerMsg(t *testing.T) {
	peers := make(map[svr.IPeer]*elanet.ServerPeer)

	s := &elanet.NetServer{
		SyncManager: netsync.New(&netsync.Config{MaxPeers: 2}),
		Routes:      routes.New(&routes.Config{}),
		ChainParams: &config.DefaultParams,
	}

	// New peers should be added.
	reply := make(chan bool, 1)
	p1, p2, p3 := mockPeer(), mockPeer(), mockPeer()
	s.HandlePeerMsg(peers, elanet.NewPeerMsg{p1, reply})
	<-reply
	s.HandlePeerMsg(peers, elanet.NewPeerMsg{p2, reply})
	<-reply

	assert.Equal(t, 2, len(peers))

	replyDonePeer := make(chan struct{}, 1)
	// Unknown done peer should not change peers.
	s.HandlePeerMsg(peers, elanet.DonePeerMsg{p3, replyDonePeer})
	<-replyDonePeer
	assert.Equal(t, 2, len(peers))

	// p1 should be removed.
	s.HandlePeerMsg(peers, elanet.DonePeerMsg{p1, replyDonePeer})
	<-replyDonePeer
	assert.Equal(t, 1, len(peers))

	// Same peer can not be removed twice.
	s.HandlePeerMsg(peers, elanet.DonePeerMsg{p1, replyDonePeer})
	<-replyDonePeer
	assert.Equal(t, 1, len(peers))

	// New peer can be added.
	s.HandlePeerMsg(peers, elanet.NewPeerMsg{p3, reply})
	<-reply
	assert.Equal(t, 2, len(peers))
}
