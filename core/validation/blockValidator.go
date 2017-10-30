package validation

import (
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/auxpow"
	"DNA_POW/core/ledger"
	tx "DNA_POW/core/transaction"
	. "DNA_POW/errors"
	"errors"
	"fmt"
	"math/big"
)

func VerifyBlock(block *ledger.Block, ld *ledger.Ledger, completely bool) error {
	if block.Blockdata.Height == 0 {
		return nil
	}
	err := VerifyBlockData(block.Blockdata, ld)
	if err != nil {
		return err
	}

	flag, err := VerifySignableData(block)
	if flag == false || err != nil {
		return err
	}

	if block.Transactions == nil {
		return errors.New(fmt.Sprintf("No Transactions Exist in Block."))
	}
	if block.Transactions[0].TxType != tx.BookKeeping {
		return errors.New(fmt.Sprintf("Blockdata Verify failed first Transacion in block is not BookKeeping type."))
	}
	for index, v := range block.Transactions {
		if v.TxType == tx.BookKeeping && index != 0 {
			return errors.New(fmt.Sprintf("This Block Has BookKeeping transaction after first transaction in block."))
		}
	}

	//verfiy block's transactions
	if completely {
		/*
			//TODO: NextBookKeeper Check.
			bookKeeperaddress, err := ledger.GetBookKeeperAddress(ld.Blockchain.GetBookKeepersByTXs(block.Transactions))
			if err != nil {
				return errors.New(fmt.Sprintf("GetBookKeeperAddress Failed."))
			}
			if block.Blockdata.NextBookKeeper != bookKeeperaddress {
				return errors.New(fmt.Sprintf("BookKeeper is not validate."))
			}
		*/
		for _, txVerify := range block.Transactions {
			if errCode := VerifyTransaction(txVerify); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransaction failed when verifiy block"))
			}
			if errCode := VerifyTransactionWithLedger(txVerify, ledger.DefaultLedger); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransactionWithLedger failed when verifiy block"))
			}
		}
		if err := VerifyTransactionWithBlock(block.Transactions); err != nil {
			return errors.New(fmt.Sprintf("VerifyTransactionWithBlock failed when verifiy block"))
		}
	}

	return nil
}

func PowVerifyBlock(block *ledger.Block, ld *ledger.Ledger, completely bool) error {
	//TODO main chian
	//TODO orphan block

	blockHash := block.Hash()
	log.Tracef("Processing block %v", blockHash)

	if ledger.DefaultLedger.BlockInLedger(blockHash) {
		log.Debug("Receive ", " duplicated block.")
		return nil
	}

	if block.Blockdata.Height == 0 {
		return nil
	}

	err := PowVerifyBlockData(block.Blockdata, ld)
	if err != nil {
		return err
	}

	//	flag, err := VerifySignableData(block)
	//	if flag == false || err != nil {
	//		return err
	//	}

	if block.Transactions == nil {
		return errors.New(fmt.Sprintf("No Transactions Exist in Block."))
	}
	//	if block.Transactions[0].TxType != tx.BookKeeping {
	//		return errors.New(fmt.Sprintf("Blockdata Verify failed first Transacion in block is not BookKeeping type."))
	//	}
	//	for index, v := range block.Transactions {
	//		if v.TxType == tx.BookKeeping && index != 0 {
	//			return errors.New(fmt.Sprintf("This Block Has BookKeeping transaction after first transaction in block."))
	//		}
	//	}
	//
	//verfiy block's transactions
	if completely {
		/*
			//TODO: NextBookKeeper Check.
			bookKeeperaddress, err := ledger.GetBookKeeperAddress(ld.Blockchain.GetBookKeepersByTXs(block.Transactions))
			if err != nil {
				return errors.New(fmt.Sprintf("GetBookKeeperAddress Failed."))
			}
			if block.Blockdata.NextBookKeeper != bookKeeperaddress {
				return errors.New(fmt.Sprintf("BookKeeper is not validate."))
			}
		*/
		for _, txVerify := range block.Transactions {
			if errCode := VerifyTransaction(txVerify); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransaction failed when verifiy block"))
			}
			if errCode := VerifyTransactionWithLedger(txVerify, ledger.DefaultLedger); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransactionWithLedger failed when verifiy block"))
			}
		}
		if err := VerifyTransactionWithBlock(block.Transactions); err != nil {
			return errors.New(fmt.Sprintf("VerifyTransactionWithBlock failed when verifiy block"))
		}
	}

	return nil
}

func VerifyHeader(bd *ledger.Header, ledger *ledger.Ledger) error {
	return VerifyBlockData(bd.Blockdata, ledger)
}
func VerifyBlockData(bd *ledger.Blockdata, ledger *ledger.Ledger) error {
	if bd.Height == 0 {
		return nil
	}

	prevHeader, err := ledger.Blockchain.GetHeader(bd.PrevBlockHash)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[BlockValidator], Cannnot find prevHeader..")
	}
	if prevHeader == nil {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], Cannnot find previous block.")
	}

	if prevHeader.Blockdata.Height+1 != bd.Height {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block height is incorrect.")
	}

	if prevHeader.Blockdata.Timestamp >= bd.Timestamp {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block timestamp is incorrect.")
	}

	return nil
}

func PowVerifyBlockData(bd *ledger.Blockdata, ledger *ledger.Ledger) error {
	if bd.Height == 0 {
		return nil
	}

	prevHeader, err := ledger.Blockchain.GetHeader(bd.PrevBlockHash)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[BlockValidator], Cannnot find prevHeader..")
	}
	if prevHeader == nil {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], Cannnot find previous block.")
	}

	if prevHeader.Blockdata.Height+1 != bd.Height {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block height is incorrect.")
	}

	if prevHeader.Blockdata.Timestamp >= bd.Timestamp {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block timestamp is incorrect.")
	}
	// TODO temp powLimit

	bigOne := big.NewInt(1)
	powLimit := new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	isAuxPow := config.Parameters.PowConfiguration.CoMining
	if isAuxPow && !bd.AuxPow.Check(bd.Hash(), auxpow.AuxPowChainID) {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block check proof is failed.")
	}
	if CheckProofOfWork(bd, powLimit, isAuxPow) != nil {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block check proof is failed.")
	}

	return nil
}

func CheckProofOfWork(bd *ledger.Blockdata, powLimit *big.Int, isAuxPow bool) error {
	// The target difficulty must be larger than zero.
	target := CompactToBig(bd.Bits)
	if target.Sign() <= 0 {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block target difficulty is too low.")
	}
	fmt.Printf("target:%x\n", target)

	// The target difficulty must be less than the maximum allowed.
	if target.Cmp(powLimit) > 0 {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block target difficulty is higher than max of limit.")
	}

	// The block hash must be less than the claimed target.
	var hash Uint256
	if isAuxPow {
		hash = bd.AuxPow.ParBlockHeader.Hash()
	} else {
		hash = bd.Hash()
	}
	hashNum := HashToBig(&hash)
	log.Tracef("hash %x\n", hash)
	log.Tracef("hashNum %x\n", hashNum)
	if hashNum.Cmp(target) > 0 {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block target difficulty is higher than expected difficulty.")
	}

	return nil
}
