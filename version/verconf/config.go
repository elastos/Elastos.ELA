package verconf

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/version"
)

type Config struct {
	Server       elanet.Server
	Chain        *blockchain.BlockChain
	ChainStore   blockchain.IChainStore
	ChainParams  *config.Params
	TxMemPool    *mempool.TxPool
	BlockMemPool *mempool.BlockPool
	DutyState    *state.DutyState
	Versions     version.HeightVersions
}
