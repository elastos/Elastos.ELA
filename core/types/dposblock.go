package types

import (
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/elanet/pact"
)

// DPOSBlock defines a block in DPOS consensus format.
type DPOSBlock struct {
	DPOSHeader
	Transactions []*Transaction
}

func (b *DPOSBlock) Serialize(w io.Writer) error {
	if err := b.DPOSHeader.Serialize(w); err != nil {
		return fmt.Errorf("DPOS block serialize failed, %s", err)
	}

	count := len(b.Transactions)
	// Limit to max transactions per block.
	if count > pact.MaxTxPerBlock {
		str := fmt.Sprintf("too many transactions in block [%v]", count)
		return common.FuncError("DPOSBlock.Serialize", str)
	}

	if err := common.WriteVarUint(w, uint64(count)); err != nil {
		return fmt.Errorf("DPOS block serialize failed, %s", err)
	}

	for _, transaction := range b.Transactions {
		if err := transaction.Serialize(w); err != nil {
			return fmt.Errorf("DPOS block serialize failed, %s", err)
		}
	}

	return nil
}

func (b *DPOSBlock) Deserialize(r io.Reader) error {
	if err := b.DPOSHeader.Deserialize(r); err != nil {
		return fmt.Errorf("DPOS block deserialize failed, %s", err)
	}

	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return fmt.Errorf("DPOS block deserialize failed, %s", err)
	}

	// Limit to max transactions per block.
	if count > pact.MaxTxPerBlock {
		str := fmt.Sprintf("too many transactions in block [%v]", count)
		return common.FuncError("DPOSBlock.Deserialize", str)
	}

	for i := uint64(0); i < count; i++ {
		var tx Transaction
		if err := tx.Deserialize(r); err != nil {
			return fmt.Errorf("DPOS block deserialize failed, %s", err)
		}
		b.Transactions = append(b.Transactions, &tx)
	}

	return nil
}

// ToBlock returns the origin POW block of the DPOS block.
func (b *DPOSBlock) ToBlock() *Block {
	return &Block{Header: b.Header, Transactions: b.Transactions}
}

// NewDPOSBlock creates a DPOS format block with the origin POW block and DPOS
// confirm.
func NewDPOSBlock(block *Block, confirm *payload.Confirm) *DPOSBlock {
	b := DPOSBlock{
		DPOSHeader: DPOSHeader{
			Header:      block.Header,
			HaveConfirm: confirm != nil,
		},
		Transactions: block.Transactions,
	}
	if confirm != nil {
		b.Confirm = *confirm
	}
	return &b
}
