package filter

import (
	"fmt"

	"github.com/elastos/Elastos.ELA/core"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

var _ TxFilter = (*TypeAddrFilter)(nil)

// TypeAddrFilter is a simple filter
type TypeAddrFilter struct {
	List []typeAddr
}

type typeAddr struct {
	Type core.TransactionType
	Addr common.Uint168
}

func (f *TypeAddrFilter) Load(filter []byte) error {
	if len(filter) < 23 {
		return fmt.Errorf("invalid filter length %d, expecting >%d",
			len(filter), 23)
	}

	count := filter[0]
	filter = filter[1:]
	if len(filter)%22 != 0 || len(filter)/22 != int(count) {
		return fmt.Errorf("invalid filter length")
	}

	f.List = make([]typeAddr, 0, count)
	for i := uint8(0); i < count; i++ {
		data := filter[i*22 : (i+1)*22]
		addr := common.Uint168{}
		copy(addr[:], data[1:])
		f.List = append(f.List, typeAddr{
			Type: core.TransactionType(data[0]),
			Addr: addr,
		})
	}

	return nil
}

func (f *TypeAddrFilter) Add(data []byte) error {
	if len(data) != 22 {
		return fmt.Errorf("invalid add data length %d, expecting %d",
			len(data), 22)
	}

	addr := common.Uint168{}
	copy(addr[:], data[1:])
	f.List = append(f.List, typeAddr{
		Type: core.TransactionType(data[0]),
		Addr: addr,
	})
	return nil
}

func (f *TypeAddrFilter) Match(tx *core.Transaction) bool {
	for _, f := range f.List {
		if f.Type != tx.TxType {
			continue
		}

		for _, output := range tx.Outputs {
			if output.ProgramHash.IsEqual(f.Addr) {
				return true
			}
		}
	}
	return false
}

func (f *TypeAddrFilter) ToMsg() *msg.TxFilterLoad {
	data := []byte{byte(len(f.List))}
	for _, f := range f.List {
		data = append(data, byte(f.Type))
		data = append(data, f.Addr[:]...)
	}

	return &msg.TxFilterLoad{
		Type: FTTypeAddr,
		Data: data,
	}
}

func (f *TypeAddrFilter) Append(txType core.TransactionType, addr *common.Uint168) {
	f.List = append(f.List, typeAddr{Type: txType, Addr: *addr})
}
