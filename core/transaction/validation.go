package transaction

import (
	"errors"

	"Elastos.ELA/crypto"
	. "Elastos.ELA/core/signature"
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
		signType, err := txn.GetTransactionType()
		if err != nil {
			return err
		}
		if signType == STANDARD {
			// Remove length byte and sign type byte
			publicKeyBytes := code[1:len(code)-1]
			if err = checkStandardSignature(publicKeyBytes, txn.GetDataContent(), param); err != nil {
				return err
			}

		} else if signType == MULTISIG {
			publicKeys, err := txn.GetMultiSignPublicKeys()
			if err != nil {
				return err
			}
			if err = checkMultiSignSignatures(code, param, txn.GetDataContent(), publicKeys); err != nil {
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

	signatureCount := 0
	for i := 0; i < len(param); i += SignatureScriptLength {
		// Remove length byte
		sign := param[i: i+SignatureScriptLength][1:]
		// Get signature index, if signature exists index will not be -1
		index := -1
		for i, publicKey := range publicKeys {
			pubKey, err := crypto.DecodePoint(publicKey[1:])
			if err != nil {
				return err
			}
			err = crypto.Verify(*pubKey, content, sign)
			if err == nil {
				index = i
			}
		}
		if index != -1 {
			signatureCount++
		}
	}
	// Check signature count
	if signatureCount != m {
		return errors.New("invalid signature count")
	}

	return nil
}
