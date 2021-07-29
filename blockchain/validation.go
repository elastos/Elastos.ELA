// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"math/big"
	"sort"

	//"github.com/btcsuite/btcd/btcec"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	. "github.com/elastos/Elastos.ELA/core/contract/program"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/crypto"
)

var (
	// Curve is a KoblitzCurve which implements secp256k1.

	//Curve = btcec.S256()
	Curve = elliptic.P256()
	// One holds a big integer of 1
	One = new(big.Int).SetInt64(1)
	// Two holds a big integer of 2
	Two = new(big.Int).SetInt64(2)
	// Three holds a big integer of 3
	Three = new(big.Int).SetInt64(3)
	// Four holds a big integer of 4
	Four = new(big.Int).SetInt64(4)
	// N2 holds a big integer of N-2
	N2 = new(big.Int).Sub(Curve.Params().N, Two)
)

func RunPrograms(data []byte, programHashes []common.Uint168, programs []*Program) error {
	if len(programHashes) != len(programs) {
		return errors.New("the number of data hashes is different with number of programs")
	}

	for i, program := range programs {
		programHash := programHashes[i]
		prefixType := contract.GetPrefixType(programHash)

		// TODO: this implementation will be deprecated
		if prefixType == contract.PrefixCrossChain {
			if err := checkCrossChainSignatures(*program, data); err != nil {
				return err
			}
			continue
		}

		codeHash := common.ToCodeHash(program.Code)
		ownerHash := programHash.ToCodeHash()

		if !ownerHash.IsEqual(*codeHash) {
			return errors.New("the data hashes is different with corresponding program code")
		}

		if prefixType == contract.PrefixStandard || prefixType == contract.PrefixDeposit {
			if contract.IsSchnorr(program.Code) {
				if ok, err := checkSchnorrSignatures(*program, data); !ok {
					return errors.New("check schnorr signature failed:" + err.Error())
				}
			} else {
				if err := checkStandardSignature(*program, data); err != nil {
					return err
				}
			}
		} else if prefixType == contract.PrefixMultiSig {
			if err := checkMultiSigSignatures(*program, data); err != nil {
				return err
			}
		} else {
			return errors.New("unknown signature type")
		}
	}

	return nil
}

func GetTxProgramHashes(tx *Transaction, references map[*Input]Output) ([]common.Uint168, error) {
	if tx == nil {
		return nil, errors.New("[Transaction],GetProgramHashes transaction is nil")
	}
	hashes := make([]common.Uint168, 0)
	uniqueHashes := make([]common.Uint168, 0)
	// add inputUTXO's transaction
	for _, output := range references {
		programHash := output.ProgramHash
		hashes = append(hashes, programHash)
	}
	for _, attribute := range tx.Attributes {
		if attribute.Usage == Script {
			dataHash, err := common.Uint168FromBytes(attribute.Data)
			if err != nil {
				return nil, errors.New("[Transaction], GetProgramHashes err")
			}
			hashes = append(hashes, *dataHash)
		}
	}

	//remove duplicated hashes
	unique := make(map[common.Uint168]bool)
	for _, v := range hashes {
		unique[v] = true
	}
	for k := range unique {
		uniqueHashes = append(uniqueHashes, k)
	}
	return uniqueHashes, nil
}

func checkStandardSignature(program Program, data []byte) error {
	if len(program.Parameter) != crypto.SignatureScriptLength {
		return errors.New("invalid signature length")
	}

	publicKey, err := crypto.DecodePoint(program.Code[1 : len(program.Code)-1])
	if err != nil {
		return err
	}

	return crypto.Verify(*publicKey, data, program.Parameter[1:])
}

func checkMultiSigSignatures(program Program, data []byte) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1
	if m < 1 || m > n {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	return verifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
}

func checkSchnorrSignatures(program Program, data []byte) (bool, error) {
	publicKey := [33]byte{}
	copy(publicKey[:], program.Code[1:len(program.Code)-1])

	signature := [64]byte{}
	copy(signature[:], program.Parameter[:64])

	return verify(publicKey, data, signature)
}

func verify(publicKey [33]byte, message []byte, signature [64]byte) (bool, error) {
	Px, Py := Unmarshal(Curve, publicKey[:])

	if Px == nil || Py == nil || !Curve.IsOnCurve(Px, Py) {
		return false, errors.New("signature verification failed")
	}
	r := new(big.Int).SetBytes(signature[:32])
	if r.Cmp(Curve.Params().P) >= 0 {
		return false, errors.New("r is larger than or equal to field size")
	}
	s := new(big.Int).SetBytes(signature[32:])
	if s.Cmp(Curve.Params().P) >= 0 {

		return false, errors.New("s is larger than or equal to curve order")
	}

	e := getE(Px, Py, intToByte(r), message)
	sGx, sGy := Curve.ScalarBaseMult(intToByte(s))
	ePx, ePy := Curve.ScalarMult(Px, Py, intToByte(e))
	ePy.Sub(Curve.Params().P, ePy)
	Rx, Ry := Curve.Add(sGx, sGy, ePx, ePy)

	if (Rx.Sign() == 0 && Ry.Sign() == 0) || big.Jacobi(Ry, Curve.Params().P) != 1 || Rx.Cmp(r) != 0 {

		return false, errors.New("signature verification failed")
	}
	return true, nil
}

func getK(Ry, k0 *big.Int) *big.Int {
	if big.Jacobi(Ry, Curve.Params().P) == 1 {
		return k0
	}
	return k0.Sub(Curve.Params().N, k0)
}

func getE(Px, Py *big.Int, rX []byte, m []byte) *big.Int {
	r := append(rX, Marshal(Curve, Px, Py)...)
	r = append(r, m[:]...)
	h := sha256.Sum256(r)
	i := new(big.Int).SetBytes(h[:])
	return i.Mod(i, Curve.Params().P)
}

func intToByte(i *big.Int) []byte {
	b1, b2 := [32]byte{}, i.Bytes()
	copy(b1[32-len(b2):], b2)
	return b1[:]
}

func Marshal(curve elliptic.Curve, x, y *big.Int) []byte {
	byteLen := (curve.Params().BitSize + 7) >> 3

	ret := make([]byte, 1+byteLen)
	ret[0] = 2

	xBytes := x.Bytes()
	copy(ret[1+byteLen-len(xBytes):], xBytes)
	ret[0] += byte(y.Bit(0))
	return ret
}

func Unmarshal(curve elliptic.Curve, data []byte) (x, y *big.Int) {
	byteLen := (curve.Params().BitSize + 7) >> 3
	if (data[0] &^ 1) != 2 {
		return
	}
	if len(data) != 1+byteLen {
		return
	}

	x0 := new(big.Int).SetBytes(data[1 : 1+byteLen])
	P := curve.Params().P
	ySq := new(big.Int)
	ySq.Exp(x0, Three, P)
	ySq.Add(ySq, curve.Params().B)
	ySq.Mod(ySq, P)
	y0 := new(big.Int)
	P1 := new(big.Int).Add(P, One)
	d := new(big.Int).Mod(P1, Four)
	P1.Sub(P1, d)
	P1.Div(P1, Four)
	y0.Exp(ySq, P1, P)

	if new(big.Int).Exp(y0, Two, P).Cmp(ySq) != 0 {
		return
	}
	if y0.Bit(0) != uint(data[0]&1) {
		y0.Sub(P, y0)
	}
	x, y = x0, y0
	return
}

// AggregateSignatures aggregates multiple signatures of different private keys over
// the same message into a single 64 byte signature.
func AggregateSignatures(privateKeys []*big.Int, message []byte) ([64]byte, error) {
	sig := [64]byte{}
	if privateKeys == nil || len(privateKeys) == 0 {
		return sig, errors.New("privateKeys must be an array with one or more elements")
	}

	k0s := []*big.Int{}
	Px, Py := new(big.Int), new(big.Int)
	Rx, Ry := new(big.Int), new(big.Int)
	for _, privateKey := range privateKeys {
		if privateKey.Cmp(One) < 0 || privateKey.Cmp(new(big.Int).Sub(Curve.Params().N, One)) > 0 {
			return sig, errors.New("the private key must be an integer in the range 1..n-1")
		}

		d := intToByte(privateKey)
		k0i, err := deterministicGetK0(d, message)
		if err != nil {
			return sig, err
		}

		RiX, RiY := Curve.ScalarBaseMult(intToByte(k0i))
		PiX, PiY := Curve.ScalarBaseMult(d)

		k0s = append(k0s, k0i)

		Rx, Ry = Curve.Add(Rx, Ry, RiX, RiY)
		Px, Py = Curve.Add(Px, Py, PiX, PiY)
	}

	rX := intToByte(Rx)
	e := getE(Px, Py, rX, message[:])
	s := new(big.Int).SetInt64(0)

	for i, k0 := range k0s {
		k := getK(Ry, k0)
		k.Add(k, new(big.Int).Mul(e, privateKeys[i]))
		s.Add(s, k)
	}

	copy(sig[:32], rX)
	copy(sig[32:], intToByte(s.Mod(s, Curve.Params().N)))
	return sig, nil
}
func deterministicGetK0(d []byte, message []byte) (*big.Int, error) {
	h := sha256.Sum256(append(d, message[:]...))
	i := new(big.Int).SetBytes(h[:])
	k0 := i.Mod(i, Curve.Params().N)
	if k0.Sign() == 0 {
		return nil, errors.New("k0 is zero")
	}

	return k0, nil
}

func checkCrossChainSignatures(program Program, data []byte) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1
	publicKeys, err := crypto.ParseCrossChainScript(code)
	if err != nil {
		return err
	}

	return verifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
}

func verifyMultisigSignatures(m, n int, publicKeys [][]byte, signatures, data []byte) error {
	if len(publicKeys) != n {
		return errors.New("invalid multi sign public key script count")
	}
	if len(signatures)%crypto.SignatureScriptLength != 0 {
		return errors.New("invalid multi sign signatures, length not match")
	}
	if len(signatures)/crypto.SignatureScriptLength < m {
		return errors.New("invalid signatures, not enough signatures")
	}
	if len(signatures)/crypto.SignatureScriptLength > n {
		return errors.New("invalid signatures, too many signatures")
	}

	var verified = make(map[common.Uint256]struct{})
	for i := 0; i < len(signatures); i += crypto.SignatureScriptLength {
		// Remove length byte
		sign := signatures[i : i+crypto.SignatureScriptLength][1:]
		// Match public key with signature
		for _, publicKey := range publicKeys {
			pubKey, err := crypto.DecodePoint(publicKey[1:])
			if err != nil {
				return err
			}
			err = crypto.Verify(*pubKey, data, sign)
			if err == nil {
				hash := sha256.Sum256(publicKey)
				if _, ok := verified[hash]; ok {
					return errors.New("duplicated signatures")
				}
				verified[hash] = struct{}{}
				break // back to public keys loop
			}
		}
	}
	// Check signatures count
	if len(verified) < m {
		return errors.New("matched signatures not enough")
	}

	return nil
}

func SortPrograms(programs []*Program) {
	sort.Sort(byHash(programs))
}

type byHash []*Program

func (p byHash) Len() int      { return len(p) }
func (p byHash) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p byHash) Less(i, j int) bool {
	hashi := common.ToCodeHash(p[i].Code)
	hashj := common.ToCodeHash(p[j].Code)
	return hashi.Compare(*hashj) < 0
}
