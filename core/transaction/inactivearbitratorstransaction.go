// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type InactiveArbitratorsTransaction struct {
	BaseTransaction
}

func (t *InactiveArbitratorsTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *InactiveArbitratorsTransaction) SpecialCheck() (elaerr.ELAError, bool) {

	if t.contextParameters.BlockChain.GetState().SpecialTxExists(t) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("tx already exists")), true
	}

	return elaerr.Simple(elaerr.ErrTxPayload, CheckInactiveArbitrators(t)), true
}

func CheckInactiveArbitrators(txn interfaces.Transaction) error {
	p, ok := txn.Payload().(*payload.InactiveArbitrators)
	if !ok {
		return errors.New("invalid payload")
	}

	if !blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(p.Sponsor) {
		return errors.New("sponsor is not belong to arbitrators")
	}

	for _, v := range p.Arbitrators {
		if !blockchain.DefaultLedger.Arbitrators.IsActiveProducer(v) &&
			!blockchain.DefaultLedger.Arbitrators.IsDisabledProducer(v) {
			return errors.New("inactive arbitrator is not belong to " +
				"arbitrators")
		}
		if blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(v) {
			return errors.New("inactive arbiters should not include CRC")
		}
	}

	if err := checkCRCArbitratorsSignatures(txn.Programs()[0]); err != nil {
		return err
	}

	return nil
}

func checkCRCArbitratorsSignatures(program *program.Program) error {

	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1

	crcArbitratorsCount := blockchain.DefaultLedger.Arbitrators.GetCRCArbitersCount()
	minSignCount := int(float64(crcArbitratorsCount)*
		state.MajoritySignRatioNumerator/state.MajoritySignRatioDenominator) + 1
	if m < 1 || m > n || n != crcArbitratorsCount || m < minSignCount {
		fmt.Printf("m:%d n:%d minSignCount:%d crc:  %d", m, n, minSignCount, crcArbitratorsCount)
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	for _, pk := range publicKeys {
		if !blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(pk[1:]) {
			return errors.New("invalid multi sign public key")
		}
	}
	return nil
}
