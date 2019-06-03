package p2p

import (
	"io"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/stretchr/testify/assert"
)

var _ Message = (*msg)(nil)

type msg struct{ i uint32 }

func (m *msg) CMD() string                   { return "msg" }
func (m *msg) MaxLength() uint32             { return 4 }
func (m *msg) Serialize(w io.Writer) error   { return common.WriteUint32(w, m.i) }
func (m *msg) Deserialize(r io.Reader) error { return common.ReadElement(r, &m.i) }

func TestBuffer_GetPayload(t *testing.T) {
	msgs := make([]*msg, 100)
	for i := uint32(0); i < 100; i++ {
		msg := &msg{i}
		msgs[i] = msg
		_, err := bufPool.GetPayload(msg)
		assert.NoError(t, err)

		size := len(bufPool.payloads)
		if size > maxBufferedPayloads {
			t.Fatalf("Too many buffered payloads(%d)", size)
		}

		if i > 9 {
			_, ok := bufPool.payloads[msgs[i-10]]
			assert.False(t, ok)
		}
	}

	quit := make(chan struct{})
	go func() {
		for i := 0; i < 10000; i++ {
			_, err := bufPool.GetPayload(msgs[i%100])
			assert.NoError(t, err)
		}
		quit <- struct{}{}
	}()

	go func() {
		for i := 0; i < 10000; i++ {
			_, err := bufPool.GetPayload(&msg{i: uint32(i)})
			assert.NoError(t, err)
		}
		quit <- struct{}{}
	}()

	<-quit
	<-quit
}
