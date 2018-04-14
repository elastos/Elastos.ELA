package SideChainStore

import (
	"database/sql"
	"math"
	"os"
	"sync"

	"github.com/elastos/Elastos.ELA/common/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DriverName      = "sqlite3"
	DBName          = "./sideChainCache.db"
	QueryHeightCode = 0
	ResetHeightCode = math.MaxUint32
)

const (
	CreateSideChainMiningTable = `CREATE TABLE IF NOT EXISTS SideChainMining (
				GenesisBlockAddress VARCHAR(34) NOT NULL PRIMARY KEY,
				MainHeight INTEGER,
				SideHeight INTEGER,
				Offset INTEGER
			);`
	CreateSideChainTxsTable = `CREATE TABLE IF NOT EXISTS SideChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34)
			);`
)

var (
	DbCache DataStore
)

type DataStore interface {
	SetMiningRecord(genesisBlockAddress string, mainHeight uint32, sideHeight uint32, offset uint8) error
	GetMiningRecord(genesisBlockAddress string, mainHeight *uint32, sideHeight *uint32, offset *uint8) (bool, error)

	AddSideChainTx(transactionHash, genesisBlockAddress string) error
	HashSideChainTx(transactionHash string) (bool, error)

	ResetDataStore() error
}

type DataStoreImpl struct {
	mainMux   *sync.Mutex
	sideMux   *sync.Mutex
	miningMux *sync.Mutex

	*sql.DB
}

func OpenDataStore() (DataStore, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{DB: db, mainMux: new(sync.Mutex), sideMux: new(sync.Mutex), miningMux: new(sync.Mutex)}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open(DriverName, DBName)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create SideChainMining table
	_, err = db.Exec(CreateSideChainMiningTable)
	if err != nil {
		return nil, err
	}
	// Create SideChainTxs table
	_, err = db.Exec(CreateSideChainTxsTable)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (store *DataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mainMux.Lock()
		store.sideMux.Lock()
		store.miningMux.Lock()
		store.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreImpl) ResetDataStore() error {

	store.DB.Close()
	os.Remove(DBName)

	var err error
	store.DB, err = initDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreImpl) SetMiningRecord(genesisBlockAddress string, mainHeight uint32, sideHeight uint32, offset uint8) error {
	store.miningMux.Lock()
	defer store.miningMux.Unlock()

	rows, err := store.Query(`SELECT * FROM SideChainMining WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	if err != nil {
		return err
	}

	if rows.Next() {
		err = rows.Close()
		if err != nil {
			return err
		}

		stmt, err := store.Prepare("UPDATE SideChainMining SET MainHeight=?, SideHeight=?, Offset=? WHERE GenesisBlockAddress=?")
		if err != nil {
			return err
		}
		_, err = stmt.Exec(mainHeight, sideHeight, offset, genesisBlockAddress)
		if err != nil {
			return err
		}

	} else {
		rows.Close()

		// Prepare sql statement
		stmt, err := store.Prepare("INSERT INTO SideChainMining(GenesisBlockAddress, MainHeight, SideHeight, Offset) values(?,?,?,?)")
		if err != nil {
			return err
		}
		// Do insert
		_, err = stmt.Exec(genesisBlockAddress, mainHeight, sideHeight, offset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (store *DataStoreImpl) GetMiningRecord(genesisBlockAddress string, mainHeight *uint32, sideHeight *uint32, offset *uint8) (bool, error) {
	store.miningMux.Lock()
	defer store.miningMux.Unlock()

	rows, err := store.Query(`SELECT MainHeight, SideHeight, Offset FROM SideChainMining WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	if rows.Next() {
		err = rows.Scan(mainHeight, sideHeight, offset)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (store *DataStoreImpl) AddSideChainTx(transactionHash, genesisBlockAddress string) error {
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress) values(?,?)")
	if err != nil {
		return err
	}
	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) HashSideChainTx(transactionHash string) (bool, error) {
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

	rows, err := store.Query(`SELECT GenesisBlockAddress FROM SideChainTxs WHERE TransactionHash=?`, transactionHash)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}
