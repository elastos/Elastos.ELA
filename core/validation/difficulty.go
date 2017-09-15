package validation

import (
	. "DNA_POW/common"
	"DNA_POW/common/log"
	"DNA_POW/core/ledger"
	. "DNA_POW/errors"
	"errors"
	"math/big"
	"time"
)

var (
	TargetTimespan     = time.Hour * 24 * 14 // 14 days
	TargetTimePerBlock = time.Minute * 10    // 10 minutes

	targetTimespan     = int64(TargetTimespan / time.Second)
	targetTimePerBlock = int64(TargetTimePerBlock / time.Second)
	blocksPerRetarget  = uint32(targetTimespan / targetTimePerBlock)

	adjustmentFactor    = int64(4) // 25% less, 400% more
	minRetargetTimespan = int64(targetTimespan / adjustmentFactor)
	maxRetargetTimespan = int64(targetTimespan * adjustmentFactor)

	// mainPowLimit is the highest proof of work value a Bitcoin block can
	// have for the main network.  It is the value 2^224 - 1.
	bigOne       = big.NewInt(1)
	PowLimit     = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)
	PowLimitBits = 0x1d00ffff

	//timeSource:          config.TimeSource,
)

func CalcNextRequiredDifficulty(block *ledger.Block, newBlockTime time.Time) (uint32, error) {
	// Genesis block.
	if block.Blockdata.Height == 0 {
		return uint32(PowLimitBits), nil
	}

	// Return the previous block's difficulty requirements if this block
	// is not at a difficulty retarget interval.
	if (block.Blockdata.Height+1)%blocksPerRetarget != 0 {
		// For the main network (or any unrecognized networks), simply
		// return the previous block's difficulty requirements.
		return block.Blockdata.Bits, nil
	}

	// Get the block node at the previous retarget (targetTimespan days
	// worth of blocks).
	firstBlock, err := ledger.DefaultLedger.GetBlockWithHeight(block.Blockdata.Height - blocksPerRetarget + 1)
	if err != nil {
		return 0, NewDetailErr(errors.New("[CalcNextRequiredDifficulty] error"), ErrNoCode, "[CalcNextRequiredDifficulty], unable to obtain previous retarget block.")
	}

	// Limit the amount of adjustment that can occur to the previous
	// difficulty.
	actualTimespan := int64(block.Blockdata.Timestamp - firstBlock.Blockdata.Timestamp)
	adjustedTimespan := actualTimespan
	if actualTimespan < minRetargetTimespan {
		adjustedTimespan = minRetargetTimespan
	} else if actualTimespan > maxRetargetTimespan {
		adjustedTimespan = maxRetargetTimespan
	}

	// Calculate new target difficulty as:
	//  currentDifficulty * (adjustedTimespan / targetTimespan)
	// The result uses integer division which means it will be slightly
	// rounded down.  Bitcoind also uses integer division to calculate this
	// result.
	oldTarget := CompactToBig(block.Blockdata.Bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	targetTimeSpan := int64(TargetTimespan / time.Second)
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))

	// Limit new value to the proof of work limit.
	if newTarget.Cmp(PowLimit) > 0 {
		newTarget.Set(PowLimit)
	}

	// Log new target difficulty and return it.  The new target logging is
	// intentionally converting the bits back to a number instead of using
	// newTarget since conversion to the compact representation loses
	// precision.
	newTargetBits := BigToCompact(newTarget)
	log.Debug("Difficulty retarget at block height %d", block.Blockdata.Height+1)
	log.Debug("Old target %08x (%064x)", block.Blockdata.Bits, oldTarget)
	log.Debug("New target %08x (%064x)", newTargetBits, CompactToBig(newTargetBits))
	log.Debug("Actual timespan %v, adjusted timespan %v, target timespan %v",
		time.Duration(actualTimespan)*time.Second,
		time.Duration(adjustedTimespan)*time.Second,
		TargetTimespan)

	return newTargetBits, nil
}

func BigToCompact(n *big.Int) uint32 {
	// No need to do any work if it's zero.
	if n.Sign() == 0 {
		return 0
	}

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes.  So, shift the number right or left
	// accordingly.  This is equivalent to:
	// mantissa = mantissa / 256^(exponent-3)
	var mantissa uint32
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		// Use a copy to avoid modifying the caller's original number.
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	// When the mantissa already has the sign bit set, the number is too
	// large to fit into the available 23-bits, so divide the number by 256
	// and increment the exponent accordingly.
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		exponent++
	}

	// Pack the exponent, sign bit, and mantissa into an unsigned 32-bit
	// int and return it.
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}

func HashToBig(hash *Uint256) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}
