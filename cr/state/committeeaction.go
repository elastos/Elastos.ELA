// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"math"
	"sort"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/utils"
)

// processTransactions takes the transactions and the Height when they have been
// packed into a block.  Then loop through the transactions to update CR
// State and Votes according to transactions content.
func (c *Committee) processTransactions(txs []interfaces.Transaction, height uint32) {
	sortedTxs := make([]interfaces.Transaction, 0)
	if len(txs) < 1 {
		return
	}
	for _, tx := range txs {
		sortedTxs = append(sortedTxs, tx)
	}
	SortTransactions(sortedTxs[1:])
	for _, tx := range sortedTxs {
		c.processTransaction(tx, height)
	}

	// Check if any pending inactive CR member has got 6 confirms, then set them
	// to elected.
	activateCRMemberFromInactive := func(cr *CRMember) {
		oriState := cr.MemberState
		oriActivateRequestHeight := cr.ActivateRequestHeight
		c.state.History.Append(height, func() {
			cr.MemberState = MemberElected
			cr.ActivateRequestHeight = math.MaxUint32
		}, func() {
			cr.MemberState = oriState
			cr.ActivateRequestHeight = oriActivateRequestHeight
		})
	}

	if c.InElectionPeriod {
		for _, v := range c.Members {
			m := v
			if (m.MemberState == MemberInactive || m.MemberState == MemberIllegal) &&
				height > m.ActivateRequestHeight &&
				height-m.ActivateRequestHeight+1 >= ActivateDuration {
				activateCRMemberFromInactive(m)
			}
		}
	}
}

// SortTransactions purpose is to process some transaction first.
func SortTransactions(txs []interfaces.Transaction) {
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].IsCRCProposalWithdrawTx() {
			return true
		}
		return !txs[j].IsCRCProposalWithdrawTx()
	})
}

// processTransaction take a transaction and the Height it has been packed into
// a block, then update producers State and Votes according to the transaction
// content.
func (c *Committee) processTransaction(tx interfaces.Transaction, height uint32) {

	// prioritize cancel Votes
	c.processCancelVotes(tx, height)

	switch tx.TxType() {
	case common2.RegisterCR:
		c.state.registerCR(tx, height)

	case common2.UpdateCR:
		c.state.updateCR(tx.Payload().(*payload.CRInfo), height)

	case common2.UnregisterCR:
		c.state.unregisterCR(tx.Payload().(*payload.UnregisterCR), height)

	case common2.TransferAsset:
		c.processVotes(tx, height)

	case common2.Voting:
		c.processVoting(tx, height)

	case common2.ReturnCRDepositCoin:
		c.state.returnDeposit(tx, height)

	case common2.CRCProposal:
		c.manager.registerProposal(tx, height, c.state.CurrentSession, c.state.History)

	case common2.CRCProposalReview:
		c.manager.proposalReview(tx, height, c.state.History)

	case common2.CRCProposalTracking:
		c.proposalTracking(tx, height)

	case common2.CRCProposalWithdraw:
		c.manager.proposalWithdraw(tx, height, c.state.History)

	case common2.CRCAppropriation:
		c.processCRCAppropriation(height, c.state.History)

	case common2.CRCProposalRealWithdraw:
		c.processCRCRealWithdraw(tx, height, c.state.History)

	case common2.CRCouncilMemberClaimNode:
		c.processCRCouncilMemberClaimNode(tx, height, c.state.History)

	case common2.ActivateProducer:
		c.activateProducer(tx, height, c.state.History)
	}

	if tx.TxType() != common2.RegisterCR {
		c.state.processDeposit(tx, height)
	}
	c.processCRCAddressRelatedTx(tx, height)
}

// proposalTracking deal with CRC proposal transaction.
func (c *Committee) proposalTracking(tx interfaces.Transaction, height uint32) {
	unusedBudget := c.manager.proposalTracking(tx, height, c.state.History)
	c.state.History.Append(height, func() {
		c.CRCCommitteeUsedAmount -= unusedBudget
	}, func() {
		c.CRCCommitteeUsedAmount += unusedBudget
	})
}

// processVotes takes a transaction, if the transaction including any vote
// outputs, validate and update CR Votes.
func (c *Committee) processVoting(tx interfaces.Transaction, height uint32) {

	pld := tx.Payload().(*payload.Voting)
	for _, content := range pld.Contents {
		switch content.VoteType {
		case outputpayload.CRC:
			for _, v := range content.VotesInfo {
				votes := v
				// record CRC votes information
				detailVoteInfo := payload.DetailVoteInfo{
					BlockHeight:    height,
					PayloadVersion: tx.PayloadVersion(),
					VoteType:       content.VoteType,
					Info:           votes,
				}

				referKey := detailVoteInfo.ReferKey()
				c.state.History.Append(height, func() {
					c.DetailCRVotes[referKey] = detailVoteInfo
				}, func() {
					delete(c.DetailCRVotes, referKey)
				})

				c.state.processVoteCRC(height, v.Candidate, v.Votes)
			}

		case outputpayload.CRCProposal:
			for _, v := range content.VotesInfo {
				c.state.processVoteCRCProposal(height, v.Candidate, v.Votes)
			}

		case outputpayload.CRCImpeachment:
			for _, v := range content.VotesInfo {
				c.processImpeachment(height, v.Candidate, v.Votes, c.state.History)
			}
		}
	}
}

// processVotes takes a transaction, if the transaction including any vote
// outputs, validate and update CR Votes.
func (c *Committee) processVotes(tx interfaces.Transaction, height uint32) {
	if tx.Version() >= common2.TxVersion09 {
		for i, output := range tx.Outputs() {
			if output.Type != common2.OTVote {
				continue
			}
			p, _ := output.Payload.(*outputpayload.VoteOutput)
			if p.Version < outputpayload.VoteProducerAndCRVersion {
				continue
			}
			// process CRC content
			var exist bool
			for _, content := range p.Contents {
				if content.VoteType == outputpayload.CRC ||
					content.VoteType == outputpayload.CRCProposal ||
					content.VoteType == outputpayload.CRCImpeachment {
					exist = true
					break
				}
			}
			if exist {
				op := common2.NewOutPoint(tx.Hash(), uint16(i))
				c.state.History.Append(height, func() {
					c.state.Votes[op.ReferKey()] = struct{}{}
				}, func() {
					delete(c.state.Votes, op.ReferKey())
				})
				c.processVoteOutput(output, height)
			}
		}
	}
}

// processVoteOutput takes a transaction output with vote payload.
func (c *Committee) processVoteOutput(output *common2.Output, height uint32) {
	p := output.Payload.(*outputpayload.VoteOutput)
	for _, vote := range p.Contents {
		for _, cv := range vote.CandidateVotes {
			switch vote.VoteType {
			case outputpayload.CRC:
				c.state.processVoteCRC(height, cv.Candidate, cv.Votes)

			case outputpayload.CRCProposal:
				c.state.processVoteCRCProposal(height, cv.Candidate, cv.Votes)

			case outputpayload.CRCImpeachment:
				c.processImpeachment(height, cv.Candidate, cv.Votes, c.state.History)
			}
		}
	}
}

// processCancelVotes takes a transaction, if the transaction takes a previous
// vote output then try to subtract the vote.
func (c *Committee) processCancelVotes(tx interfaces.Transaction, height uint32) {
	var exist bool
	for _, input := range tx.Inputs() {
		referKey := input.ReferKey()
		if _, ok := c.state.Votes[referKey]; ok {
			exist = true
		}
	}
	if !exist {
		return
	}

	references, err := c.state.GetTxReference(tx)
	if err != nil {
		log.Errorf("get tx reference failed, tx hash:%s", common.ToReversedString(tx.Hash()))
		return
	}
	for _, input := range tx.Inputs() {
		referKey := input.ReferKey()
		_, ok := c.state.Votes[referKey]
		if ok {
			out := references[input]
			c.processVoteCancel(&out, height)
		}
	}
}

// processVoteCancel takes a previous vote output and decrease CR Votes.
func (c *Committee) processVoteCancel(output *common2.Output, height uint32) {
	p := output.Payload.(*outputpayload.VoteOutput)
	for _, vote := range p.Contents {
		for _, cv := range vote.CandidateVotes {
			switch vote.VoteType {
			case outputpayload.CRC:
				cid, err := common.Uint168FromBytes(cv.Candidate)
				if err != nil {
					continue
				}
				candidate := c.state.GetCandidate(*cid)
				if candidate == nil {
					continue
				}
				v := cv.Votes
				c.state.History.Append(height, func() {
					candidate.Votes -= v
				}, func() {
					candidate.Votes += v
				})

			case outputpayload.CRCProposal:
				proposalHash, err := common.Uint256FromBytes(cv.Candidate)
				if err != nil {
					continue
				}
				proposalState := c.manager.getProposal(*proposalHash)
				if proposalState == nil || proposalState.Status != CRAgreed {
					continue
				}
				v := cv.Votes
				c.state.History.Append(height, func() {
					proposalState.VotersRejectAmount -= v
				}, func() {
					proposalState.VotersRejectAmount += v
				})

			case outputpayload.CRCImpeachment:
				c.processCancelImpeachment(height, cv.Candidate, cv.Votes, c.state.History)
			}
		}
	}
}

func (c *Committee) processCancelImpeachment(height uint32, member []byte,
	votes common.Fixed64, history *utils.History) {
	var crMember *CRMember
	for _, v := range c.Members {
		if bytes.Equal(v.Info.CID.Bytes(), member) &&
			(v.MemberState == MemberElected ||
				v.MemberState == MemberInactive || v.MemberState == MemberIllegal) {
			crMember = v
			break
		}
	}
	if crMember == nil {
		return
	}
	history.Append(height, func() {
		crMember.ImpeachmentVotes -= votes
	}, func() {
		crMember.ImpeachmentVotes += votes
	})
	return
}

// processCRCRelatedAmount takes a transaction, if the transaction takes a previous
// output to CRC related address then try to subtract the vote.
func (c *Committee) processCRCAddressRelatedTx(tx interfaces.Transaction, height uint32) {
	if tx.IsCRCProposalTx() {
		proposal := tx.Payload().(*payload.CRCProposal)
		var budget common.Fixed64
		for _, b := range proposal.Budgets {
			budget += b.Amount
		}
		c.state.History.Append(height, func() {
			c.CRCCommitteeUsedAmount += budget
		}, func() {
			c.CRCCommitteeUsedAmount -= budget
		})
	}

	for _, input := range tx.Inputs() {
		if amount, ok := c.state.CRCFoundationOutputs[input.Previous.ReferKey()]; ok {
			c.state.History.Append(height, func() {
				c.CRAssetsAddressUTXOCount--
				c.CRCFoundationBalance -= amount
			}, func() {
				c.CRAssetsAddressUTXOCount++
				c.CRCFoundationBalance += amount
			})
		} else if amount, ok := c.state.CRCCommitteeOutputs[input.Previous.ReferKey()]; ok {
			c.state.History.Append(height, func() {
				c.CRCCommitteeBalance -= amount
			}, func() {
				c.CRCCommitteeBalance += amount
			})
		}
	}

	for _, output := range tx.Outputs() {
		amount := output.Value
		if output.ProgramHash.IsEqual(c.Params.CRAssetsAddress) {
			c.state.History.Append(height, func() {
				c.CRAssetsAddressUTXOCount++
				c.CRCFoundationBalance += amount
			}, func() {
				c.CRAssetsAddressUTXOCount--
				c.CRCFoundationBalance -= amount
			})
		} else if output.ProgramHash.IsEqual(c.Params.CRExpensesAddress) {
			c.state.History.Append(height, func() {
				c.CRCCommitteeBalance += amount
			}, func() {
				c.CRCCommitteeBalance -= amount
			})
		} else if output.ProgramHash.IsEqual(c.Params.DestroyELAAddress) {
			c.state.History.Append(height, func() {
				c.DestroyedAmount += amount
			}, func() {
				c.DestroyedAmount -= amount
			})
		}
	}
}
