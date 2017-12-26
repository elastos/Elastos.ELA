package account

import (
	. "Elastos.ELA/common"
	ct "Elastos.ELA/core/contract"
)

type IClientStore interface {
	BuildDatabase(path string)

	SaveStoredData(name string, value []byte)
	LoadStoredData(name string) []byte

	CreateAccount() (*Account, error)
	CreateAccountByPrivateKey(privateKey []byte) (*Account, error)
	LoadAccounts() map[Uint168]*Account

	CreateContract(account *Account) error
	LoadContracts() map[Uint168]*ct.Contract

	SaveHeight(height uint32) error
	LoadHeight() (uint32, error)
}
