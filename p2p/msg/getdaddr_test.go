package msg

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDAddr_Serialize(t *testing.T) {
	msg1 := GetDAddr{PIDs: map[[33]byte][][33]byte{}}
	msg2 := GetDAddr{PIDs: map[[33]byte][][33]byte{}}

	for i := 0; i < 3; i++ {
		var pid [33]byte
		rand.Read(pid[:])

		ids := make([][33]byte, 3)
		for i := range ids {
			rand.Read(ids[i][:])
		}
		msg1.PIDs[pid] = ids
	}

	buf := new(bytes.Buffer)
	err := msg1.Serialize(buf)
	assert.NoError(t, err)

	err = msg2.Deserialize(buf)
	assert.NoError(t, err)

	assert.Equal(t, msg1, msg2)
	t.Log(msg1)
	t.Log(msg2)
}
