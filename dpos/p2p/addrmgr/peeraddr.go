package addrmgr

import (
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/elastos/Elastos.ELA/common"
)

const (
	// maxHostLength defines the maximum host name length.
	maxHostLength = 253

	// MaxPeerAddrSize indicates the maximum size of a serialized PeerAddr.
	MaxPeerAddrSize = 33 + maxHostLength + 2
)

// PeerAddr represents the network address to connect to a DPoS peer.
type PeerAddr struct {
	PID  [33]byte
	Host string
	Port uint16
}

func (pa *PeerAddr) Network() string {
	return "tcp"
}

func (pa *PeerAddr) String() string {
	return net.JoinHostPort(pa.Host, fmt.Sprint(pa.Port))
}

func (pa *PeerAddr) Serialize(w io.Writer) error {
	if _, err := w.Write(pa.PID[:]); err != nil {
		return err
	}

	if err := common.WriteVarString(w, pa.Host); err != nil {
		return err
	}

	return common.WriteElement(w, pa.Port)
}

func (pa *PeerAddr) Deserialize(r io.Reader) error {
	var err error
	if _, err = io.ReadFull(r, pa.PID[:]); err != nil {
		return err
	}

	if pa.Host, err = common.ReadVarString(r); err != nil {
		return err
	}

	pa.Port, err = common.ReadUint16(r)
	return err
}

// NewPeerAddr creates a new PeerAddr instance.
func NewPeerAddr(pid [33]byte, host string, port uint16) *PeerAddr {
	return &PeerAddr{PID: pid, Host: host, Port: port}
}

// AddrStringToPeerAddr converts a given address string to a *PeerAddr
func AddrStringToPeerAddr(pid [33]byte, addr string) (*PeerAddr, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}

	return NewPeerAddr(pid, host, uint16(port)), nil
}
