// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/crypto"
)

type Voting struct {
	Contents []VotesContent
}

func (p *Voting) Data(version byte) []byte {

	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *Voting) Serialize(w io.Writer, version byte) error {

	if err := common.WriteVarUint(w, uint64(len(p.Contents))); err != nil {
		return err
	}
	for _, content := range p.Contents {
		if err := content.Serialize(w, version); err != nil {
			return err
		}
	}

	return nil
}

func (p *Voting) Deserialize(r io.Reader, version byte) error {

	contentsCount, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	p.Contents = make([]VotesContent, 0)
	for i := uint64(0); i < contentsCount; i++ {
		var content VotesContent
		if err := content.Deserialize(r, version); err != nil {
			return err
		}
		p.Contents = append(p.Contents, content)
	}

	return nil
}

func (p *Voting) Validate() error {

	typeMap := make(map[outputpayload.VoteType]struct{})
	for _, content := range p.Contents {
		if _, exists := typeMap[content.VoteType]; exists {
			return errors.New("duplicate vote type")
		}
		typeMap[content.VoteType] = struct{}{}
		if len(content.VotesInfo) == 0 || (content.VoteType == outputpayload.Delegate &&
			len(content.VotesInfo) > outputpayload.MaxVoteProducersPerTransaction) {
			return errors.New("invalid public key count")
		}
		if content.VoteType != outputpayload.Delegate && content.VoteType != outputpayload.CRC &&
			content.VoteType != outputpayload.CRCProposal && content.VoteType != outputpayload.CRCImpeachment && content.VoteType != outputpayload.DposV2 {
			return errors.New("invalid vote type")
		}

		candidateMap := make(map[string]struct{})
		for _, cv := range content.VotesInfo {
			c := common.BytesToHexString(cv.Candidate)
			if _, exists := candidateMap[c]; exists {
				return errors.New("duplicate candidate")
			}
			candidateMap[c] = struct{}{}

			if cv.Votes <= 0 {
				return errors.New("invalid candidate votes")
			}
		}
	}

	return nil
}

func (p Voting) String() string {
	return fmt.Sprint("Vote: {\n\t\t\t",
		"Vote: ", p.Contents, "\n\t\t\t}")
}

// VotesWithLockTime defines the voting information for individual candidates.
type VotesWithLockTime struct {
	Candidate []byte
	Votes     common.Fixed64
	LockTime  uint32
}

func (vi *VotesWithLockTime) Serialize(w io.Writer, version byte) error {

	if err := common.WriteVarBytes(w, vi.Candidate); err != nil {
		return err
	}
	if err := vi.Votes.Serialize(w); err != nil {
		return err
	}
	if err := common.WriteUint32(w, vi.LockTime); err != nil {
		return err
	}

	return nil
}

func (vi *VotesWithLockTime) Deserialize(r io.Reader, version byte) error {

	candidate, err := common.ReadVarBytes(
		r, crypto.MaxMultiSignCodeLength, "candidate votes")
	if err != nil {
		return err
	}
	vi.Candidate = candidate

	if err := vi.Votes.Deserialize(r); err != nil {
		return err
	}
	if vi.LockTime, err = common.ReadUint32(r); err != nil {
		return err
	}

	return nil
}

func (vi *VotesWithLockTime) String() string {
	return fmt.Sprint("Content: {"+
		"\n\t\t\t\t", "Candidate: ", common.BytesToHexString(vi.Candidate),
		"\n\t\t\t\t", "Votes: ", vi.Votes,
		"\n\t\t\t\t", "LockTime: ", vi.LockTime,
		"}\n\t\t\t\t")
}

// VotesContent defines the vote type and vote information of candidates.
type VotesContent struct {
	VoteType  outputpayload.VoteType
	VotesInfo []VotesWithLockTime
}

func (vc *VotesContent) Serialize(w io.Writer, version byte) error {
	if _, err := w.Write([]byte{byte(vc.VoteType)}); err != nil {
		return err
	}
	if err := common.WriteVarUint(w, uint64(len(vc.VotesInfo))); err != nil {
		return err
	}
	for _, candidate := range vc.VotesInfo {
		if err := candidate.Serialize(w, version); err != nil {
			return err
		}
	}

	return nil
}

func (vc *VotesContent) Deserialize(r io.Reader, version byte) error {
	voteType, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	vc.VoteType = outputpayload.VoteType(voteType[0])

	candidatesCount, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	for i := uint64(0); i < candidatesCount; i++ {
		var cv VotesWithLockTime
		if cv.Deserialize(r, version); err != nil {
			return err
		}
		vc.VotesInfo = append(vc.VotesInfo, cv)
	}

	return nil
}

func (vc VotesContent) String() string {
	candidates := make([]string, 0)
	for _, c := range vc.VotesInfo {
		candidates = append(candidates, common.BytesToHexString(c.Candidate))
	}

	if len(vc.VotesInfo) != 0 && vc.VotesInfo[0].Votes == 0 {
		return fmt.Sprint("Content: {\n\t\t\t\t",
			"VoteType: ", vc.VoteType, "\n\t\t\t\t",
			"Candidates: ", candidates, "}\n\t\t\t\t")
	}

	return fmt.Sprint("Content: {\n\t\t\t\t",
		"VoteType: ", vc.VoteType, "\n\t\t\t\t",
		"CandidateVotes: ", vc.VotesInfo, "}\n\t\t\t\t")
}
