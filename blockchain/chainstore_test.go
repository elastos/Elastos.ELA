package blockchain

import (
	"container/list"
	"testing"

	ela "github.com/elastos/Elastos.ELA/core"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

var testChainStore *ChainStore
var sidechainTxHash common.Uint256

func newTestChainStore() (*ChainStore, error) {
	// TODO: read config file decide which db to use.
	st, err := NewLevelDB("Chain_UnitTest")
	if err != nil {
		return nil, err
	}

	store := &ChainStore{
		IStore:             st,
		headerIndex:        map[uint32]common.Uint256{},
		headerCache:        map[common.Uint256]*ela.Header{},
		headerIdx:          list.New(),
		currentBlockHeight: 0,
		storedHeaderCount:  0,
		taskCh:             make(chan persistTask, TaskChanCap),
		quit:               make(chan chan bool, 1),
	}

	go store.loop()
	store.NewBatch()

	return store, nil
}

func TestChainStoreInit(t *testing.T) {
	// Get new chainstore
	var err error
	testChainStore, err = newTestChainStore()
	if err != nil {
		t.Error("Create chainstore failed")
	}

	// Assume the sidechain Tx hash
	txHashStr := "39fc8ba05b0064381e51afed65b4cf91bb8db60efebc38242e965d1b1fed0701"
	txHashBytes, _ := common.HexStringToBytes(txHashStr)
	txHash, _ := common.Uint256FromBytes(txHashBytes)
	sidechainTxHash = *txHash
}

func TestChainStore_PersisSidechainTx(t *testing.T) {
	if testChainStore == nil {
		t.Error("Chainstore init failed")
	}

	// 1. The sidechain Tx should not exist in DB.
	_, err := testChainStore.GetSidechainTx(sidechainTxHash)
	if err == nil {
		t.Error("Found the sidechain Tx which should not exist in DB")
	}

	// 2. Run PersistSidechainTx
	testChainStore.PersistSidechainTx(sidechainTxHash)

	// Need batch commit here because PersistSidechainTx use BatchPut
	testChainStore.BatchCommit()

	// 3. Verify PersistSidechainTx
	exist, err := testChainStore.GetSidechainTx(sidechainTxHash)
	if err != nil {
		t.Error("Not found the sidechain Tx")
	}
	if exist != ValueExist {
		t.Error("Sidechian Tx matched wrong value")
	}
}

func TestChainStore_RollbackSidechainTx(t *testing.T) {
	if testChainStore == nil {
		t.Error("Chainstore init failed")
	}

	// 1. The sidechain Tx hash should exist in DB.
	exist, err := testChainStore.GetSidechainTx(sidechainTxHash)
	if err != nil {
		t.Error("Not found the sidechain Tx")
	}
	if exist != ValueExist {
		t.Error("Sidechian Tx matched wrong value")
	}

	// 2. Run Rollback
	err = testChainStore.RollbackSidechainTx(sidechainTxHash)
	if err != nil {
		t.Error("Rollback the sidechain Tx failed")
	}

	// Need batch commit here because RollbackSidechainTx use BatchDelete
	testChainStore.BatchCommit()

	// 3. Verify RollbackSidechainTx
	_, err = testChainStore.GetSidechainTx(sidechainTxHash)
	if err == nil {
		t.Error("Found the sidechain Tx which should been deleted")
	}
}

func TestChainStore_IsSidechainTxHashDuplicate(t *testing.T) {
	if testChainStore == nil {
		t.Error("Chainstore init failed")
	}

	// 1. The sidechain Tx should not exist in DB.
	_, err := testChainStore.GetSidechainTx(sidechainTxHash)
	if err == nil {
		t.Error("Found the sidechain Tx which should not exist in DB")
	}

	// 2. Persist the sidechain Tx hash
	testChainStore.PersistSidechainTx(sidechainTxHash)

	// Need batch commit here because PersistSidechainTx use BatchPut
	testChainStore.BatchCommit()

	// 3. Verify PersistSidechainTx
	exist, err := testChainStore.GetSidechainTx(sidechainTxHash)
	if err != nil {
		t.Error("Not found the sidechain Tx")
	}
	if exist != ValueExist {
		t.Error("Sidechian Tx matched wrong value")
	}

	// 4. Run IsSidechainTxHashDuplicate
	isDuplicate := testChainStore.IsSidechainTxHashDuplicate(sidechainTxHash)
	if !isDuplicate {
		t.Error("Sidechain Tx hash should be checked to be duplicated")
	}
}

func TestChainStore_PersistSidechianRegInfo(t *testing.T) {
	// Assume the sidechain Tx hash
	hashBytes, _ := common.HexStringToBytes("8d43921dbd91a9c9f46fe562fb14f274fec98cd647be54002f41cd4d20ae4cc7")
	genesisHash, _ := common.Uint256FromBytes(hashBytes)
	coinIndex := uint32(0)
	name := "test_chain"
	payload := []byte{byte(10)}

	// 1. The register info should not exist in DB.
	_, err := testChainStore.GetSidechainRegInfo(*genesisHash)
	if err == nil {
		t.Error("Found the sidechain register info which should not exist in DB")
	}

	// 2. Run PersistSidechianRegInfo
	testChainStore.PersistSidechianRegInfo(*genesisHash, coinIndex, name, payload)
	testChainStore.BatchCommit()

	// 3. Verify PersistSidechianRegInfo
	_, err = testChainStore.GetSidechainRegInfo(*genesisHash)
	if err != nil {
		t.Error("Sidechain register info is not found")
	}
}

func TestChainStore_RollbackSidechainRegInfo(t *testing.T) {
	// Assume the sidechain Tx hash
	hashBytes, _ := common.HexStringToBytes("8d43921dbd91a9c9f46fe562fb14f274fec98cd647be54002f41cd4d20ae4cc7")
	genesisHash, _ := common.Uint256FromBytes(hashBytes)
	coinIndex := uint32(0)
	name := "test_chain"
	//payload := []byte{byte(10)}

	// 1. The register info should exist in DB.
	data, err := testChainStore.GetSidechainRegInfo(*genesisHash)
	if err != nil {
		t.Error("Not found the sidechain info")
	}
	if data[0] != byte(10) {
		t.Error("Sidechain info matched wrong value")
	}

	// 2. Run Rollback
	err = testChainStore.RollbackSidechainRegInfo(*genesisHash, coinIndex, name)
	if err != nil {
		t.Error("Rollback the sidechain register info failed")
	}
	testChainStore.BatchCommit()

	// 3. Verify GetSidechainRegInfo
	_, err = testChainStore.GetSidechainRegInfo(*genesisHash)
	if err == nil {
		t.Error("Found the sidechain register info which should been deleted")
	}
}

func TestChainStoreDone(t *testing.T) {
	if testChainStore == nil {
		t.Error("Chainstore init failed")
	}

	err := testChainStore.RollbackSidechainTx(sidechainTxHash)
	if err != nil {
		t.Error("Rollback the sidechain Tx failed")
	}

	hashBytes, _ := common.HexStringToBytes("8d43921dbd91a9c9f46fe562fb14f274fec98cd647be54002f41cd4d20ae4cc7")
	genesisHash, _ := common.Uint256FromBytes(hashBytes)
	coinIndex := uint32(0)
	name := "test_chain"

	err = testChainStore.RollbackSidechainRegInfo(*genesisHash, coinIndex, name)
	if err != nil {
		t.Error("Rollback the sidechain Tx failed")
	}

	testChainStore.BatchCommit()
}
