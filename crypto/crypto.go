package crypto

import (
	"Elastos.ELA/common/serialization"
	"errors"
	"fmt"
	"io"
	"math/big"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/elliptic"
	"crypto/sha256"
)

const (
	SIGNRLEN     = 32
	SIGNATURELEN = 64
	NEGBIGNUMLEN = 33
)

type CryptoAlgSet struct {
	EccParams elliptic.CurveParams
	Curve     elliptic.Curve
}

var AlgChoice int

var algSet CryptoAlgSet

type PubKey struct {
	X, Y *big.Int
}

func init() {
	algSet.Curve = elliptic.P256()
	algSet.EccParams = *(algSet.Curve.Params())
}

func GenKeyPair() ([]byte, PubKey, error) {

	privateKey, err := ecdsa.GenerateKey(algSet.Curve, rand.Reader)
	if err != nil {
		return nil, PubKey{}, errors.New("Generate key pair error")
	}

	mPubKey := new(PubKey)
	mPubKey.X = new(big.Int).Set(privateKey.PublicKey.X)
	mPubKey.Y = new(big.Int).Set(privateKey.PublicKey.Y)

	return privateKey.D.Bytes(), *mPubKey, nil
}

func Sign(priKey []byte, data []byte) ([]byte, error) {

	digest := sha256.Sum256(data)

	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = algSet.Curve
	privateKey.D = big.NewInt(0)
	privateKey.D.SetBytes(priKey)

	r := big.NewInt(0)
	s := big.NewInt(0)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		fmt.Printf("Sign error\n")
		return nil, err
	}

	signature := make([]byte, SIGNATURELEN)

	lenR := len(r.Bytes())
	lenS := len(s.Bytes())
	copy(signature[SIGNRLEN-lenR:], r.Bytes())
	copy(signature[SIGNATURELEN-lenS:], s.Bytes())
	return signature, nil
}

func Verify(publicKey PubKey, data []byte, signature []byte) error {
	len := len(signature)
	if len != SIGNATURELEN {
		fmt.Printf("Unknown signature length %d\n", len)
		return errors.New("Unknown signature length")
	}

	r := new(big.Int).SetBytes(signature[:len/2])
	s := new(big.Int).SetBytes(signature[len/2:])

	digest := sha256.Sum256(data)

	pub := new(ecdsa.PublicKey)
	pub.Curve = algSet.Curve

	pub.X = new(big.Int).Set(publicKey.X)
	pub.Y = new(big.Int).Set(publicKey.Y)

	if ecdsa.Verify(pub, digest[:], r, s) {
		return nil
	} else {
		return errors.New("[Validation], Verify failed.")
	}

}

func (e *PubKey) Serialize(w io.Writer) error {
	bufX := []byte{}
	if e.X.Sign() == -1 {
		// prefix 0x00 means the big number X is negative
		bufX = append(bufX, 0x00)
	}
	bufX = append(bufX, e.X.Bytes()...)

	if err := serialization.WriteVarBytes(w, bufX); err != nil {
		return err
	}

	bufY := []byte{}
	if e.Y.Sign() == -1 {
		// prefix 0x00 means the big number Y is negative
		bufY = append(bufY, 0x00)
	}
	bufY = append(bufY, e.Y.Bytes()...)
	if err := serialization.WriteVarBytes(w, bufY); err != nil {
		return err
	}
	return nil
}

func (e *PubKey) Deserialize(r io.Reader) error {
	bufX, err := serialization.ReadVarBytes(r)
	if err != nil {
		return err
	}
	e.X = big.NewInt(0)
	e.X = e.X.SetBytes(bufX)
	if len(bufX) == NEGBIGNUMLEN {
		e.X.Neg(e.X)
	}
	bufY, err := serialization.ReadVarBytes(r)
	if err != nil {
		return err
	}
	e.Y = big.NewInt(0)
	e.Y = e.Y.SetBytes(bufY)
	if len(bufY) == NEGBIGNUMLEN {
		e.Y.Neg(e.Y)
	}
	return nil
}

type PubKeySlice []*PubKey

func (p PubKeySlice) Len() int { return len(p) }
func (p PubKeySlice) Less(i, j int) bool {
	r := p[i].X.Cmp(p[j].X)
	if r <= 0 {
		return true
	}
	return false
}
func (p PubKeySlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func Equal(e1 *PubKey, e2 *PubKey) bool {
	r := e1.X.Cmp(e2.X)
	if r != 0 {
		return false
	}
	r = e1.Y.Cmp(e2.Y)
	if r == 0 {
		return true
	}
	return false
}
