// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package wallet

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	config2 "github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils"
)

const (
	// utxoCheckPointKey defines key of utxo checkpoint.
	utxoCheckPointKey = "utxo"

	// dataExtension defines checkpoint file extension of utxo checkpoint.
	dataExtension = ".ucp"

	// dataPath indicates the path storing the chain data.
	dataPath = "data"

	// checkpointPath indicates the path storing the checkpoint data.
	checkpointPath = "checkpoints"

	// savePeriod defines interval height between two neighbor check
	// points.
	savePeriod = uint32(720)

	effectivePeriod = uint32(720)
)

type AddressInfo struct {
	address string
	code    []byte
}

type Wallet struct {
	*CoinsCheckPoint
	*account.Client
}

func (w *Wallet) LoadAddresses() error {
	storeAccounts, err := w.LoadAccountData()
	if err != nil {
		return err
	}
	for _, account := range storeAccounts {
		code, err := common.HexStringToBytes(account.RedeemScript)
		if err != nil {
			return err
		}
		SetWalletAccount(&AddressInfo{
			address: account.Address,
			code:    code,
		})
	}

	return nil
}

func (w *Wallet) ImportPubkey(pubKey []byte, enableUtxoDB bool) error {
	pk, err := crypto.DecodePoint(pubKey)
	if err != nil {
		return errors.New("invalid public key")
	}
	sc, err := contract.CreateStandardContract(pk)
	if err != nil {
		return err
	}
	address, err := sc.ToProgramHash().ToAddress()
	if err != nil {
		return err
	}
	if err := w.SaveAccountData(sc.ToProgramHash(), sc.Code, nil); err != nil {
		return err
	}
	SetWalletAccount(&AddressInfo{
		address: address,
		code:    sc.Code,
	})
	config := config2.DefaultParams
	ckpManager := checkpoint.NewManager(&config)
	ckpManager.SetDataPath(filepath.Join(dataPath, checkpointPath))
	ckpManager.Reset(func(point checkpoint.ICheckPoint) bool {
		return point.Key() == utxoCheckPointKey
	})

	if enableUtxoDB {
		return nil
	}

	return w.RescanWallet()
}

func (w *Wallet) ImportAddress(address string, enableUtxoDB bool) error {
	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		return errors.New("invalid address")
	}
	if err := w.SaveAccountData(programHash, nil, nil); err != nil {
		return err
	}
	SetWalletAccount(&AddressInfo{
		address: address,
		code:    nil,
	})
	config := config2.DefaultParams
	ckpManager := checkpoint.NewManager(&config)
	ckpManager.SetDataPath(filepath.Join(dataPath, checkpointPath))
	ckpManager.Reset(func(point checkpoint.ICheckPoint) bool {
		return point.Key() == utxoCheckPointKey
	})

	if enableUtxoDB {
		return nil
	}

	return w.RescanWallet()
}

func (w *Wallet) ListUnspent(address string, enableUtxoDB bool) (map[common.Uint256][]*common2.UTXO,
	error) {
	coins := w.ListCoins(address)
	utxos := make([]*common2.UTXO, 0)
	for op, coin := range coins {
		utxos = append(utxos, &common2.UTXO{
			TxID:  op.TxID,
			Index: op.Index,
			Value: coin.Output.Value,
		})
	}
	unspent := make(map[common.Uint256][]*common2.UTXO, 0)
	unspent[*account.SystemAssetID] = utxos

	return unspent, nil
}

func (w *Wallet) RescanWallet() error {
	return nil
}

func NewWallet() *Wallet {
	return &Wallet{
		CoinsCheckPoint: NewCoinCheckPoint(),
	}
}

func New(dataDir string) *Wallet {
	path := filepath.Join(dataDir, account.KeystoreFileName)
	wallet := Wallet{
		CoinsCheckPoint: NewCoinCheckPoint(),
	}

	exist := utils.FileExisted(path)
	if !exist {
		pwd, err := utils.GetConfirmedPassword()
		if err != nil {
			log.Warn("Get password failed, use an empty password. " + err.Error())
		}
		client, err := account.Create(path, pwd)
		if err != nil {
			log.Warn("Create wallet failed, " + err.Error())
			os.Exit(1)
		}
		wallet.Client = client
	} else {
		pwd, err := utils.GetPassword()
		if err != nil {
			log.Warn("Get password failed, use an empty password. " + err.Error())
		}
		client, err := account.Open(path, pwd)
		if err != nil {
			log.Warn("Open wallet failed, " + err.Error())
			os.Exit(1)
		}
		wallet.Client = client
		if err := wallet.LoadAddresses(); err != nil {
			log.Warn("Build wallet failed" + err.Error())
		}
	}

	return &wallet
}
