package distributor

import (
	"crypto/rand"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDistributor_Mapping tests if the connection mapping working correctly.
func TestDistributor_Mapping(t *testing.T) {
	d := New()
	// Mapping 12345 to 23456.
	err := d.Mapping(12345, "localhost:23456")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	d.Start()
	defer d.Stop()

	conns := make(chan net.Conn)
	listener, err := net.Listen("tcp", ":23456")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	go func() {
		for {
			out, err := listener.Accept()
			if err != nil {
				t.Error(err)
				continue
			}
			conns <- out
		}
	}()

	in, err := net.Dial("tcp", "localhost:12345")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	var out net.Conn
	select {
	case out = <-conns:
	case <-time.After(time.Millisecond):
		t.Fatal("mapping connection timeout")
	}

	// Two way stream test.
	buf := make([]byte, 10240)
	_, _ = rand.Read(buf[:])
	_, err = in.Write(buf)
	if err != nil {
		t.FailNow()
	}

	data := make([]byte, 10240)
	_, err = io.ReadFull(out, data)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, buf, data) {
		t.FailNow()
	}

	buf = make([]byte, 10240)
	_, _ = rand.Read(buf[:])
	_, err = out.Write(buf)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	data = make([]byte, 10240)
	_, err = io.ReadFull(in, data)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, buf, data) {
		t.FailNow()
	}
}

// TestDistributor_Disconnect test if inlet or outlet connection can be closed
// as expected.
func TestDistributor_Disconnect(t *testing.T) {
	d := New()
	// Mapping 34567 to 45678.
	err := d.Mapping(34567, "localhost:45678")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	d.Start()
	defer d.Stop()

	// 1--start outlet connection not started.
	conn, err := net.Dial("tcp", "localhost:34567")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	_, err = conn.Read(make([]byte, 1024))
	if !assert.Error(t, err) {
		t.FailNow()
	}
	// 1--finish outlet connection not started.

	// 2--start outlet connection closed.
	conns := make(chan net.Conn)
	listener, err := net.Listen("tcp", ":45678")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	go func() {
		for {
			out, err := listener.Accept()
			if err != nil {
				t.Error(err)
				continue
			}
			conns <- out
		}
	}()
	in, err := net.Dial("tcp", "localhost:34567")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	var out net.Conn
	select {
	case out = <-conns:
	case <-time.After(time.Millisecond):
		t.Fatal("mapping connection timeout")
	}

	buf := make([]byte, 1024)
	_, _ = rand.Read(buf[:])
	_, err = in.Write(buf)
	if err != nil {
		t.FailNow()
	}

	data := make([]byte, 1024)
	_, err = io.ReadFull(out, data)
	if err != nil {
		t.FailNow()
	}

	if !assert.Equal(t, buf, data) {
		t.FailNow()
	}

	err = out.Close()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	_, err = in.Read(make([]byte, 1024))
	if !assert.Error(t, err) {
		t.FailNow()
	}
	// 2--finish outlet connection closed.

	// 3--start inlet connection closed.
	in, err = net.Dial("tcp", "localhost:34567")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	select {
	case out = <-conns:
	case <-time.After(time.Millisecond):
		t.Fatal("mapping connection timeout")
	}

	buf = make([]byte, 1024)
	_, _ = rand.Read(buf[:])
	_, err = in.Write(buf)
	if err != nil {
		t.FailNow()
	}

	data = make([]byte, 1024)
	_, err = io.ReadFull(out, data)
	if err != nil {
		t.FailNow()
	}

	if !assert.Equal(t, buf, data) {
		t.FailNow()
	}

	err = in.Close()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	_, err = out.Read(make([]byte, 1024))
	if !assert.Error(t, err) {
		t.FailNow()
	}
	// 3--finish inlet connection closed.
}
