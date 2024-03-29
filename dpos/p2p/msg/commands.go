// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package msg

import (
	"bytes"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p"
)

const (
	CmdVersion  = "version"
	CmdVerAck   = "verack"
	CmdAddr     = "addr"
	CmdPing     = "ping"
	CmdPong     = "pong"
	CmdInv      = "inv"
	CmdGetBlock = "getblock"

	CmdReceivedProposal            = "proposal"
	CmdAcceptVote                  = "acc_vote"
	CmdRejectVote                  = "rej_vote"
	CmdGetBlocks                   = "get_blc"
	CmdResponseBlocks              = "res_blc"
	CmdRequestConsensus            = "req_con"
	CmdResponseConsensus           = "res_con"
	CmdRequestProposal             = "req_pro"
	CmdIllegalProposals            = "ill_pro"
	CmdIllegalVotes                = "ill_vote"
	CmdSidechainIllegalData        = "side_ill"
	CmdResponseInactiveArbitrators = "ina_ars"
	CmdResponseRevertToDPOS        = "rev_to_dpos"
	CmdResetConsensusView          = "reset_view"
)

func GetMessageHash(msg p2p.Message) common.Uint256 {
	buf := new(bytes.Buffer)
	msg.Serialize(buf)
	msgHash := common.Sha256D(buf.Bytes())
	return msgHash
}
