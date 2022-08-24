// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package crypto

import (
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"math/big"
	"math/rand"
)

var (
	// Curve is a KoblitzCurve which implements secp256r1.
	Curve = DefaultCurve
	// One holds a big integer of 1
	One = new(big.Int).SetInt64(1)
	// Two holds a big integer of 2
	Two = new(big.Int).SetInt64(2)
	// Three holds a big integer of 3
	Three = new(big.Int).SetInt64(3)
	// Four holds a big integer of 4
	Four = new(big.Int).SetInt64(4)
	// The order of the base point
	N = DefaultParams.N
	// The order of the underlying field
	P = DefaultParams.P
	// The constant of the curve equation
	B = DefaultParams.B
	// The size of the underlying field
	BitSize = DefaultParams.BitSize
)

func SchnorrVerify(publicKey [33]byte, message [32]byte, signature [64]byte) (bool, error) {
	Px, Py := Unmarshal(Curve, publicKey[:])

	if Px == nil || Py == nil || !Curve.IsOnCurve(Px, Py) {
		return false, errors.New("signature verification failed")
	}
	r := new(big.Int).SetBytes(signature[:32])
	if r.Cmp(P) >= 0 {
		return false, errors.New("r is larger than or equal to field size")
	}
	s := new(big.Int).SetBytes(signature[32:])
	if s.Cmp(N) >= 0 {

		return false, errors.New("s is larger than or equal to curve order")
	}

	e := getE(Px, Py, intToByte(r), message)
	sGx, sGy := Curve.ScalarBaseMult(intToByte(s))
	ePx, ePy := Curve.ScalarMult(Px, Py, intToByte(e))
	ePy.Sub(P, ePy)
	Rx, Ry := Curve.Add(sGx, sGy, ePx, ePy)

	if Rx.Sign() == 0 && Ry.Sign() == 0 {
		return false, errors.New("signature verification failed")
	}
	if big.Jacobi(Ry, P) != 1 {
		return false, errors.New("signature Ry verification failed")
	}
	if Rx.Cmp(r) != 0 {
		return false, errors.New("signature Rx verification failed")
	}
	return true, nil
}

func getK(Ry, k0 *big.Int) *big.Int {
	if big.Jacobi(Ry, P) == 1 {
		return k0
	}
	return k0.Sub(N, k0)
}

func getE(Px, Py *big.Int, rX []byte, m [32]byte) *big.Int {
	r := append(rX, Marshal(Curve, Px, Py)...)
	r = append(r, m[:]...)
	h := sha256.Sum256(r)
	i := new(big.Int).SetBytes(h[:])
	return i.Mod(i, N)
}

func intToByte(i *big.Int) []byte {
	b1, b2 := [32]byte{}, i.Bytes()
	copy(b1[32-len(b2):], b2)
	return b1[:]
}

func Marshal(curve elliptic.Curve, x, y *big.Int) []byte {
	byteLen := (BitSize + 7) >> 3

	ret := make([]byte, 1+byteLen)
	ret[0] = 2

	xBytes := x.Bytes()
	copy(ret[1+byteLen-len(xBytes):], xBytes)
	ret[0] += byte(y.Bit(0))
	return ret
}

func Unmarshal(curve elliptic.Curve, data []byte) (x, y *big.Int) {
	byteLen := (BitSize + 7) >> 3
	if (data[0] &^ 1) != 2 {
		return
	}
	if len(data) != 1+byteLen {
		return
	}
	paramA := big.NewInt(-3)
	x0 := new(big.Int).SetBytes(data[1 : 1+byteLen])

	ax := new(big.Int)
	ax.Add(ax, paramA)
	ax.Mul(ax, x0)

	ySq := new(big.Int)
	ySq.Exp(x0, Three, P)
	ySq.Add(ySq, ax)
	ySq.Add(ySq, B)
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
func AggregateSignatures(privateKeys []*big.Int, message [32]byte) ([64]byte, error) {
	sig := [64]byte{}
	if privateKeys == nil || len(privateKeys) == 0 {
		return sig, errors.New("privateKeys must be an array with one or more elements")
	}

	k0s := []*big.Int{}
	Px, Py := new(big.Int), new(big.Int)
	Rx, Ry := new(big.Int), new(big.Int)
	for _, privateKey := range privateKeys {
		if privateKey.Cmp(One) < 0 || privateKey.Cmp(new(big.Int).Sub(N, One)) > 0 {
			return sig, errors.New("the private key must be an integer in the range 1..n-1")
		}

		d := intToByte(privateKey)
		k0i, err := deterministicGetK0(d)
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
	e := getE(Px, Py, rX, message)
	s := new(big.Int).SetInt64(0)

	for i, k0 := range k0s {
		k := getK(Ry, k0)
		k.Add(k, new(big.Int).Mul(e, privateKeys[i]))
		s.Add(s, k)
	}

	copy(sig[:32], rX)
	copy(sig[32:], intToByte(s.Mod(s, N)))
	return sig, nil
}

func randomBytes(len int) []byte {
	a := make([]byte, len)
	rand.Read(a)
	return a
}

func deterministicGetK0(d []byte) (*big.Int, error) {
	for {
		message := randomBytes(32)
		h := sha256.Sum256(append(d, message[:]...))
		i := new(big.Int).SetBytes(h[:])
		k0 := i.Mod(i, N)
		if k0.Sign() == 0 {
			return nil, errors.New("k0 is zero")
		}
		return k0, nil
	}
}
