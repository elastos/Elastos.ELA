// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package msg

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/dtime"
	"github.com/elastos/Elastos.ELA/p2p"
)

// Ensure Version implement p2p.Message interface.
var _ p2p.Message = (*Version)(nil)

const DPoSV1Version = 0x00
const DPoSV2Version = 0x01

var PayloadVersionLock sync.RWMutex

var PayloadVersion uint32

func GetPayloadVersion() uint32 {
	PayloadVersionLock.RLock()
	defer PayloadVersionLock.RUnlock()
	return PayloadVersion
}

func SetPayloadVersion(version uint32) {
	PayloadVersionLock.Lock()
	defer PayloadVersionLock.Unlock()
	PayloadVersion = version
}

type Version struct {
	Version     uint32
	PID         [33]byte
	Target      [16]byte
	Nonce       [16]byte
	Port        uint16
	Timestamp   time.Time
	NodeVersion string
}

func (msg *Version) CMD() string {
	return p2p.CmdVersion
}

func (msg *Version) MaxLength() uint32 {
	return 128 // 33+16+16+2+8 + Version +NodeVersion so extend to 128
}

func (msg *Version) Serialize(w io.Writer) error {
	if GetPayloadVersion() >= DPoSV2Version {
		if err := common.WriteUint32(w, msg.Version); err != nil {
			return err
		}
	}
	err := common.WriteElements(w, msg.PID, msg.Target, msg.Nonce, msg.Port,
		msg.Timestamp.UnixNano())
	if GetPayloadVersion() >= DPoSV2Version {
		err = common.WriteVarString(w, msg.NodeVersion)
	}
	return err
}

func (msg *Version) Deserialize(r io.Reader) error {
	if GetPayloadVersion() >= DPoSV2Version {
		v, err := common.ReadUint32(r)
		if err != nil {
			return err
		}
		msg.Version = v
	}
	var timestamp int64
	err := common.ReadElements(r, &msg.PID, &msg.Target, &msg.Nonce, &msg.Port,
		&timestamp)
	if err != nil {
		return err
	}

	// A version timestamp must not have greater precision than one million
	// second.
	if timestamp%dtime.Millisecond > 0 {
		return errors.New("version timestamp have a higher precision" +
			" than one million second")
	}

	msg.Timestamp = dtime.Int64ToTime(timestamp)

	if GetPayloadVersion() >= DPoSV2Version {
		msg.NodeVersion, err = common.ReadVarString(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewVersion(version uint32, pid [33]byte, target, nonce [16]byte, port uint16, nver string) *Version {
	return &Version{Version: version, PID: pid, Target: target, Nonce: nonce, Port: port,
		Timestamp: dtime.Now(), NodeVersion: nver}
}
