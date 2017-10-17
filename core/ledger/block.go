package ledger

import (
	. "DNA_POW/common"
	"DNA_POW/common/log"
	"DNA_POW/common/serialization"
	"DNA_POW/core/asset"
	"DNA_POW/core/contract/program"
	sig "DNA_POW/core/signature"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	"DNA_POW/crypto"
	. "DNA_POW/errors"
	"DNA_POW/vm"
	"io"
	"time"
)

const BlockVersion uint32 = 0
const GenesisNonce uint64 = 2083236893

type Block struct {
	Blockdata    *Blockdata
	Transactions []*tx.Transaction

	hash *Uint256
}

func (b *Block) Serialize(w io.Writer) error {
	b.Blockdata.Serialize(w)
	err := serialization.WriteUint32(w, uint32(len(b.Transactions)))
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Block item Transactions length serialization failed.")
	}

	for _, transaction := range b.Transactions {
		transaction.Serialize(w)
	}
	return nil
}

func (b *Block) Deserialize(r io.Reader) error {
	if b.Blockdata == nil {
		b.Blockdata = new(Blockdata)
	}
	b.Blockdata.Deserialize(r)

	//Transactions
	var i uint32
	Len, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	var txhash Uint256
	var tharray []Uint256
	for i = 0; i < Len; i++ {
		transaction := new(tx.Transaction)
		transaction.Deserialize(r)
		txhash = transaction.Hash()
		b.Transactions = append(b.Transactions, transaction)
		tharray = append(tharray, txhash)
	}

	b.Blockdata.TransactionsRoot, err = crypto.ComputeRoot(tharray)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Block Deserialize merkleTree compute failed")
	}

	return nil
}

func (b *Block) Trim(w io.Writer) error {
	b.Blockdata.Serialize(w)
	err := serialization.WriteUint32(w, uint32(len(b.Transactions)))
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Block item Transactions length serialization failed.")
	}
	for _, transaction := range b.Transactions {
		temp := *transaction
		hash := temp.Hash()
		hash.Serialize(w)
	}
	return nil
}

func (b *Block) FromTrimmedData(r io.Reader) error {
	if b.Blockdata == nil {
		b.Blockdata = new(Blockdata)
	}
	b.Blockdata.Deserialize(r)

	//Transactions
	var i uint32
	Len, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	var txhash Uint256
	var tharray []Uint256
	for i = 0; i < Len; i++ {
		txhash.Deserialize(r)
		transaction := new(tx.Transaction)
		transaction.SetHash(txhash)
		b.Transactions = append(b.Transactions, transaction)
		tharray = append(tharray, txhash)
	}

	b.Blockdata.TransactionsRoot, err = crypto.ComputeRoot(tharray)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Block Deserialize merkleTree compute failed")
	}

	return nil
}

func (b *Block) GetMessage() []byte {
	return sig.GetHashData(b)
}

func (b *Block) GetProgramHashes() ([]Uint160, error) {

	return b.Blockdata.GetProgramHashes()
}

func (b *Block) SetPrograms(prog []*program.Program) {
	b.Blockdata.SetPrograms(prog)
	return
}

func (b *Block) GetPrograms() []*program.Program {
	return b.Blockdata.GetPrograms()
}

func (b *Block) Hash() Uint256 {
	if b.hash == nil {
		b.hash = new(Uint256)
		*b.hash = b.Blockdata.Hash()
	}
	return *b.hash
}

func (b *Block) Verify() error {
	log.Info("This function is expired.please use Validation/blockValidator to Verify Block.")
	return nil
}

func (b *Block) Type() InventoryType {
	return BLOCK
}

func GenesisBlockInit(defaultBookKeeper []*crypto.PubKey) (*Block, error) {
	//getBookKeeper
	nextBookKeeper, err := GetBookKeeperAddress(defaultBookKeeper)
	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "[Block],GenesisBlockInit err with GetBookKeeperAddress")
	}
	//blockdata
	genesisBlockdata := &Blockdata{
		Version:          BlockVersion,
		PrevBlockHash:    Uint256{},
		TransactionsRoot: Uint256{},
		Timestamp:        uint32(uint32(time.Date(2017, time.February, 23, 0, 0, 0, 0, time.UTC).Unix())),
		Bits:             0x1d00ffff,
		Nonce:            uint32(0),
		Height:           uint32(0),
		ConsensusData:    GenesisNonce,
		NextBookKeeper:   nextBookKeeper,
		Program: &program.Program{
			Code:      []byte{},
			Parameter: []byte{byte(vm.PUSHT)},
		},
	}
	admin, err := ToCodeHash([]byte{byte(vm.PUSHT)})
	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "[GenesisBlock], share admin error")
	}

	//transaction
	trans := &tx.Transaction{
		TxType:         tx.BookKeeping,
		PayloadVersion: payload.BookKeepingPayloadVersion,
		Payload: &payload.BookKeeping{
			Nonce: GenesisNonce,
		},
		Attributes:    []*tx.TxAttribute{},
		UTXOInputs:    []*tx.UTXOTxInput{},
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       []*tx.TxOutput{},
		Programs:      []*program.Program{},
	}

	systemToken := &tx.Transaction{
		TxType:         tx.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.RegisterAsset{
			Asset: &asset.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x01, // share
			},
			Amount:     1 * 100000000,
			Issuer:     defaultBookKeeper[0],
			Controller: admin,
		},
		Attributes: []*tx.TxAttribute{},
		UTXOInputs: []*tx.UTXOTxInput{},
		Outputs:    []*tx.TxOutput{},
		Programs:   []*program.Program{},
	}

	systemTokenIssurance := &tx.Transaction{
		TxType:         tx.IssueAsset,
		PayloadVersion: 0,
		Payload:        &payload.IssueAsset{},
		Attributes:     []*tx.TxAttribute{},
		UTXOInputs:     []*tx.UTXOTxInput{},
		Outputs: []*tx.TxOutput{
			{
				AssetID:     systemToken.Hash(),
				Value:       1 * 100000000,
				ProgramHash: admin,
			},
		},
		Programs: []*program.Program{
			{
				Code:      []byte{},
				Parameter: []byte{byte(vm.PUSHT)},
			},
		},
	}
	//block
	genesisBlock := &Block{
		Blockdata:    genesisBlockdata,
		Transactions: []*tx.Transaction{trans, systemToken, systemTokenIssurance},
	}

	txHashes := []Uint256{}
	for _, tx := range genesisBlock.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	merkleRoot, err := crypto.ComputeRoot(txHashes)
	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "[GenesisBlock], merkle root error")
	}
	genesisBlock.Blockdata.TransactionsRoot = merkleRoot

	return genesisBlock, nil
}

func (b *Block) RebuildMerkleRoot() error {
	txs := b.Transactions
	transactionHashes := []Uint256{}
	for _, tx := range txs {
		transactionHashes = append(transactionHashes, tx.Hash())
	}
	hash, err := crypto.ComputeRoot(transactionHashes)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[Block] , RebuildMerkleRoot ComputeRoot failed.")
	}
	b.Blockdata.TransactionsRoot = hash
	return nil

}

func (bd *Block) SerializeUnsigned(w io.Writer) error {
	return bd.Blockdata.SerializeUnsigned(w)
}
