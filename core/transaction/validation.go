package transaction

import (
	"errors"
	"crypto/sha256"

	"Elastos.ELA/crypto"
	. "Elastos.ELA/core/signature"
	"Elastos.ELA/common"
)

func VerifySignature(txn *Transaction) error {
	hashes, err := txn.GetProgramHashes()
	if err != nil {
		return err
	}

	programs := txn.GetPrograms()
	Length := len(hashes)
	if Length != len(programs) {
		return errors.New("The number of data hashes is different with number of programs.")
	}

	data := txn.GetDataContent()

	for i := 0; i < len(programs); i++ {

		code := programs[i].Code
		param := programs[i].Parameter

		programHash, err := ToProgramHash(code)
		if err != nil {
			return err
		}

		if hashes[i] != programHash {
			return errors.New("The data hashes is different with corresponding program code.")
		}
		// Get transaction type
		signType, err := txn.GetOpCode(i)
		if err != nil {
			return err
		}
		if signType == STANDARD {
			// Remove length byte and sign type byte
			publicKeyBytes := code[1:len(code)-1]
			if err = checkStandardSignature(publicKeyBytes, data, param); err != nil {
				return err
			}

		} else if signType == MULTISIG {
			publicKeys, err := txn.GetMultiSignPublicKeys(i)
			if err != nil {
				return err
			}
			if err = checkMultiSignSignatures(code, param, data, publicKeys); err != nil {
				return err
			}

		} else {
			return errors.New("unknown signature type")
		}
	}

	return nil
}

func checkStandardSignature(publicKeyBytes, content, signature []byte) error {
	if len(signature) != SignatureScriptLength {
		return errors.New("Invalid signature length")
	}

	publicKey, err := crypto.DecodePoint(publicKeyBytes)
	if err != nil {
		return err
	}

	return crypto.Verify(*publicKey, content, signature[1:])
}

func checkMultiSignSignatures(code, param, content []byte, publicKeys [][]byte) error {
	// Get N parameter
	n := int(code[len(code)-2]) - PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - PUSH1 + 1
	if m < 1 || m > n {
		return errors.New("invalid multi sign script code")
	}
	if len(publicKeys) != n {
		return errors.New("invalid multi sign public key script count")
	}
	if len(param)%SignatureScriptLength != 0 {
		return errors.New("invalid multi sign signatures, length not match")
	}
	if len(param)/SignatureScriptLength < m {
		return errors.New("invalid signatures, not enough signatures")
	}
	if len(param)/SignatureScriptLength > n {
		return errors.New("invalid signatures, too many signatures")
	}

	var verified = make(map[common.Uint256]struct{})
	for i := 0; i < len(param); i += SignatureScriptLength {
		// Remove length byte
		sign := param[i: i+SignatureScriptLength][1:]
		// Match public key with signature
		for _, publicKey := range publicKeys {
			pubKey, err := crypto.DecodePoint(publicKey[1:])
			if err != nil {
				return err
			}
			err = crypto.Verify(*pubKey, content, sign)
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
