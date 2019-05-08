package msg

import (
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA/p2p"
)

// Ensure GetAddr implement p2p.Message interface.
var _ p2p.Message = (*GetAddr)(nil)

// GetAddr implements the Message interface and represents a getaddr message.
type GetAddr struct {
	PIDs [][33]byte
}

// AddPID adds a known DPoS peer ID to the message.
func (msg *GetAddr) AddPID(pids ...[33]byte) {
	for _, pid := range pids {
		msg.PIDs = append(msg.PIDs, pid)
	}
}

func (msg *GetAddr) CMD() string {
	return CmdGetAddr
}

func (msg *GetAddr) MaxLength() uint32 {
	return 1 + addrmgr.MaxPeerAddrSize*maxPeerAddrs
}

func (msg *GetAddr) Serialize(w io.Writer) error {
	count := len(msg.PIDs)
	if count > maxPeerAddrs {
		return fmt.Errorf("too many addresses for message "+
			"[count %v, max %v]", count, maxPeerAddrs)
	}

	if err := common.WriteVarUint(w, uint64(count)); err != nil {
		return err
	}

	for _, pa := range msg.PIDs {
		if _, err := w.Write(pa[:]); err != nil {
			return err
		}
	}

	return nil
}

func (msg *GetAddr) Deserialize(r io.Reader) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	// Limit to max addresses per message.
	if count > maxPeerAddrs {
		return fmt.Errorf("too many addresses for message "+
			"[count %v, max %v]", count, maxPeerAddrs)
	}

	msg.PIDs = make([][33]byte, count)
	for i := uint64(0); i < count; i++ {
		if _, err := io.ReadFull(r, msg.PIDs[i][:]); err != nil {
			return err
		}
	}
	return nil
}

// NewGetAddr returns a new bitcoin getaddr message that conforms to the
// Message interface.  See MsgGetAddr for details.
func NewGetAddr() *GetAddr {
	return &GetAddr{
		PIDs: make([][33]byte, 0, maxPeerAddrs),
	}
}
