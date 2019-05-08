package msg

import (
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA/p2p"
)

const (
	// maxPeerAddrs indicates the maximum PeerAddr number of an addr message.
	maxPeerAddrs = 72
)

// Ensure Ping implement p2p.Message interface.
var _ p2p.Message = (*Addr)(nil)

// Addr implements the Message interface and represents a DPoS addr message.
type Addr struct {
	AddrList []*addrmgr.PeerAddr
}

// AddPeerAddr adds a known DPoS peer addr to the message.
func (msg *Addr) AddPeerAddr(pas ...*addrmgr.PeerAddr) {
	for _, pa := range pas {
		msg.AddrList = append(msg.AddrList, pa)
	}
}

func (msg *Addr) CMD() string {
	return CmdAddr
}

func (msg *Addr) MaxLength() uint32 {
	return 1 + addrmgr.MaxPeerAddrSize*maxPeerAddrs
}

func (msg *Addr) Serialize(w io.Writer) error {
	count := len(msg.AddrList)
	if count > maxPeerAddrs {
		return fmt.Errorf("too many addresses for message "+
			"[count %v, max %v]", count, maxPeerAddrs)
	}

	if err := common.WriteVarUint(w, uint64(count)); err != nil {
		return err
	}

	for _, pa := range msg.AddrList {
		if err := pa.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (msg *Addr) Deserialize(r io.Reader) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	// Limit to max addresses per message.
	if count > maxPeerAddrs {
		return fmt.Errorf("too many addresses for message "+
			"[count %v, max %v]", count, maxPeerAddrs)
	}

	addrList := make([]addrmgr.PeerAddr, count)
	msg.AddrList = make([]*addrmgr.PeerAddr, 0, count)
	for i := uint64(0); i < count; i++ {
		pa := &addrList[i]
		if err := pa.Deserialize(r); err != nil {
			return err
		}
		msg.AddPeerAddr(pa)
	}
	return nil
}

// NewAddr returns a new addr message that conforms to the Message interface.
func NewAddr() *Addr {
	return &Addr{
		AddrList: make([]*addrmgr.PeerAddr, 0, maxPeerAddrs),
	}
}
