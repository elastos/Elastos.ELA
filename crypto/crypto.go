// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"io"
	"math/big"
	"sort"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto/ecies"
)

const (
	SignerLength      = 32
	SignatureLength   = 64
	NegativeBigLength = 33
)

var (
	DefaultCurve  = elliptic.P256()
	DefaultParams = DefaultCurve.Params()
)

type PublicKey struct {
	X, Y *big.Int
}

func GenerateKeyPair() ([]byte, *PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(DefaultCurve, rand.Reader)
	if err != nil {
		return nil, nil, errors.New("Generate key pair error")
	}

	publicKey := PublicKey{}
	publicKey.X = privateKey.PublicKey.X
	publicKey.Y = privateKey.PublicKey.Y

	return privateKey.D.Bytes(), &publicKey, nil
}

func GenerateSubKeyPair(index int, chainCode, parentPrivateKey []byte) ([]byte, *PublicKey, error) {

	if len(chainCode) != 32 {
		return nil, nil, errors.New("invalid chain code, length not equal to 32")
	}

	digest := parentPrivateKey
	for i := 0; i < index; i++ {
		temp := sha256.Sum256(append(chainCode, digest...))
		digest = temp[:]
	}

	publicKey := PublicKey{}
	publicKey.X, publicKey.Y = DefaultCurve.ScalarBaseMult(digest)

	return digest, &publicKey, nil
}

func SignDigest(priKey []byte, digest []byte) ([]byte, error) {
	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = DefaultCurve
	privateKey.D = big.NewInt(0)
	privateKey.D.SetBytes(priKey)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
	if err != nil {
		return nil, err
	}

	signature := make([]byte, SignatureLength)

	lenR := len(r.Bytes())
	lenS := len(s.Bytes())
	copy(signature[SignerLength-lenR:], r.Bytes())
	copy(signature[SignatureLength-lenS:], s.Bytes())
	return signature, nil
}

func VerifyDigest(pubkey PublicKey, digest []byte, signature []byte) error {
	len := len(signature)
	if len != SignatureLength {
		fmt.Printf("Unknown signature length %d\n", len)
		return errors.New("Unknown signature length")
	}

	r := new(big.Int).SetBytes(signature[:len/2])
	s := new(big.Int).SetBytes(signature[len/2:])

	pub := ecdsa.PublicKey{}
	pub.Curve = DefaultCurve
	pub.X = pubkey.X
	pub.Y = pubkey.Y

	if !ecdsa.Verify(&pub, digest, r, s) {
		return errors.New("[Validation], Verify failed.")
	}

	return nil
}

func Sign(priKey []byte, data []byte) ([]byte, error) {

	digest := sha256.Sum256(data)

	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = DefaultCurve
	privateKey.D = big.NewInt(0)
	privateKey.D.SetBytes(priKey)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		return nil, err
	}

	signature := make([]byte, SignatureLength)

	lenR := len(r.Bytes())
	lenS := len(s.Bytes())
	copy(signature[SignerLength-lenR:], r.Bytes())
	copy(signature[SignatureLength-lenS:], s.Bytes())
	return signature, nil
}

func Verify(publicKey PublicKey, data []byte, signature []byte) error {
	len := len(signature)
	if len != SignatureLength {
		fmt.Printf("Unknown signature length %d\n", len)
		return errors.New("Unknown signature length")
	}

	r := new(big.Int).SetBytes(signature[:len/2])
	s := new(big.Int).SetBytes(signature[len/2:])

	digest := sha256.Sum256(data)

	pub := ecdsa.PublicKey{}
	pub.Curve = DefaultCurve
	pub.X = publicKey.X
	pub.Y = publicKey.Y

	if !ecdsa.Verify(&pub, digest[:], r, s) {
		return errors.New("[Validation], Verify failed.")
	}

	return nil
}

// Encrypt encrypts a message using ECIES as specified in SEC 1, 5.1.
func Encrypt(publicKey *PublicKey, message []byte) (ct []byte, err error) {
	pubKey := ecies.PublicKey{
		X:      publicKey.X,
		Y:      publicKey.Y,
		Curve:  DefaultCurve,
		Params: ecies.ParamsFromCurve(DefaultCurve),
	}

	return ecies.Encrypt(rand.Reader, &pubKey, message, nil, nil)
}

// Decrypt decrypts an ECIES ciphertext.
func Decrypt(privateKey, cipher []byte) (m []byte, err error) {
	priKey := ecies.PrivateKey{}
	priKey.D = new(big.Int).SetBytes(privateKey)
	priKey.Curve = DefaultCurve
	priKey.Params = ecies.ParamsFromCurve(DefaultCurve)

	return priKey.Decrypt(cipher, nil, nil)
}

func (e *PublicKey) Serialize(w io.Writer) error {
	bufX := make([]byte, 0, NegativeBigLength)
	if e.X.Sign() == -1 {
		// prefix 0x00 means the big number X is negative
		bufX = append(bufX, 0x00)
	}
	bufX = append(bufX, e.X.Bytes()...)

	if err := common.WriteVarBytes(w, bufX); err != nil {
		return err
	}

	bufY := make([]byte, 0, NegativeBigLength)
	if e.Y.Sign() == -1 {
		// prefix 0x00 means the big number Y is negative
		bufY = append(bufY, 0x00)
	}
	bufY = append(bufY, e.Y.Bytes()...)
	if err := common.WriteVarBytes(w, bufY); err != nil {
		return err
	}
	return nil
}

func (e *PublicKey) Deserialize(r io.Reader) error {
	bufX, err := common.ReadVarBytes(r, NegativeBigLength,
		"public key X")
	if err != nil {
		return err
	}
	e.X = big.NewInt(0)
	e.X = e.X.SetBytes(bufX)
	if len(bufX) == NegativeBigLength {
		e.X.Neg(e.X)
	}
	bufY, err := common.ReadVarBytes(r, NegativeBigLength,
		"public key Y")
	if err != nil {
		return err
	}
	e.Y = big.NewInt(0)
	e.Y = e.Y.SetBytes(bufY)
	if len(bufY) == NegativeBigLength {
		e.Y.Neg(e.Y)
	}
	return nil
}

func SortPublicKeys(publicKeys []*PublicKey) {
	sort.Sort(byX(publicKeys))
}

type byX []*PublicKey

func (p byX) Len() int           { return len(p) }
func (p byX) Less(i, j int) bool { return p[i].X.Cmp(p[j].X) <= 0 }
func (p byX) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func Equal(e1 *PublicKey, e2 *PublicKey) bool {
	if e1.X.Cmp(e2.X) != 0 {
		return false
	}
	if e1.Y.Cmp(e2.Y) != 0 {
		return false
	}
	return true
}

func CheckMultiSigSignatures(program program.Program, data []byte) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - PUSH1 + 1
	if m < 1 || m > n {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := ParseMultisigScript(code)
	if err != nil {
		return err
	}

	return VerifyMultisigSignatures(m, n, publicKeys, program.Parameter, data)
}
func VerifyMultisigSignatures(m, n int, publicKeys [][]byte, signatures, data []byte) error {
	if len(publicKeys) != n {
		return errors.New("invalid multi sign public key script count")
	}
	if len(signatures)%SignatureScriptLength != 0 {
		return errors.New("invalid multi sign signatures, length not match")
	}
	if len(signatures)/SignatureScriptLength < m {
		return errors.New("invalid signatures, not enough signatures")
	}
	if len(signatures)/SignatureScriptLength > n {
		return errors.New("invalid signatures, too many signatures")
	}

	var verified = make(map[common.Uint256]struct{})
	for i := 0; i < len(signatures); i += SignatureScriptLength {
		// Remove length byte
		sign := signatures[i : i+SignatureScriptLength][1:]
		// Match public key with signature
		for _, publicKey := range publicKeys {
			pubKey, err := DecodePoint(publicKey[1:])
			if err != nil {
				return err
			}
			err = Verify(*pubKey, data, sign)
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
