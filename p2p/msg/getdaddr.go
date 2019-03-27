package msg

import (
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p"
)

const (
	// maxPIDsPerMsg defines the maximum number of PIDs in this message.
	maxPIDsPerMsg = 72
)

// Ensure GetData implement p2p.Message interface.
var _ p2p.Message = (*GetDAddr)(nil)

func serialize(ids [][33]byte, w io.Writer) error {
	count := len(ids)
	if count > maxPIDsPerMsg {
		str := fmt.Sprintf("too many PID in message [%v]", count)
		return common.FuncError("GetDAddr.Serialize", str)
	}

	if err := common.WriteVarUint(w, uint64(count)); err != nil {
		return err
	}

	for _, pid := range ids {
		if _, err := w.Write(pid[:]); err != nil {
			return err
		}
	}

	return nil
}

func deserialize(r io.Reader) ([][33]byte, error) {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return nil, err
	}

	// Limit to max PIDs per message.
	if count > maxPIDsPerMsg {
		str := fmt.Sprintf("too many PID in message [%v]", count)
		return nil, common.FuncError("GetDAddr.Deserialize", str)
	}

	// Create a contiguous slice of PIDs to deserialize into in order to
	// reduce the number of allocations.
	ids := make([][33]byte, count)
	for i := uint64(0); i < count; i++ {
		_, err := io.ReadFull(r, ids[i][:])
		if err != nil {
			return nil, err
		}
	}

	return ids, nil
}

// GetDAddr defines a message to request DPOS addresses.
type GetDAddr struct {
	PIDs map[[33]byte][][33]byte
}

func (msg *GetDAddr) CMD() string {
	return p2p.CmdGetDAddr
}

func (msg *GetDAddr) MaxLength() uint32 {
	return (maxPIDsPerMsg + 1) * (2 + maxPIDsPerMsg*33)
}

func (msg *GetDAddr) Serialize(w io.Writer) error {
	count := len(msg.PIDs)
	if count > maxPIDsPerMsg {
		str := fmt.Sprintf("too many PID in message [%v]", count)
		return common.FuncError("GetDAddr.Serialize", str)
	}

	if err := common.WriteVarUint(w, uint64(count)); err != nil {
		return err
	}

	for pid, pids := range msg.PIDs {
		if _, err := w.Write(pid[:]); err != nil {
			return err
		}

		if err := serialize(pids, w); err != nil {
			return err
		}
	}

	return nil
}

func (msg *GetDAddr) Deserialize(r io.Reader) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	// Limit to max PIDs per message.
	if count > maxPIDsPerMsg {
		str := fmt.Sprintf("too many PID in message [%v]", count)
		return common.FuncError("GetDAddr.Deserialize", str)
	}

	// Create a contiguous slice of PIDs to deserialize into in order to
	// reduce the number of allocations.
	msg.PIDs = make(map[[33]byte][][33]byte, count)
	for i := uint64(0); i < count; i++ {
		var pid [33]byte
		_, err := io.ReadFull(r, pid[:])
		if err != nil {
			return err
		}

		ids, err := deserialize(r)
		if err != nil {
			return err
		}

		msg.PIDs[pid] = ids
	}

	return nil
}

func NewGetDAddr(pids map[[33]byte][][33]byte) *GetDAddr {
	return &GetDAddr{PIDs: pids}
}
