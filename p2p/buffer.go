package p2p

import (
	"bytes"
	"container/list"
	"sync"
)

const (
	// bufferPoolSize defines the buffer pool size.
	bufferPoolSize = 10

	// maxBufferCap defines the maximum capacity of a buffer.
	maxBufferCap = 1024 * 100 // 100KB

	// maxBufferedPayloads defines the maximum buffered message payloads.
	maxBufferedPayloads = 10
)

var (
	bufPool = &buffer{
		bufChan:  make(chan *bytes.Buffer, bufferPoolSize),
		payloads: make(map[Message][]byte, maxBufferedPayloads),
		msgList:  list.New(),
	}
)

// buffer is a buffer pool to reduce memory use and allocation.
type buffer struct {
	mtx      sync.RWMutex
	bufChan  chan *bytes.Buffer
	payloads map[Message][]byte
	msgList  *list.List
}

// Get gets a Buffer from the buffer pool, or creates a new one if none are
// available in the pool. Buffers have a pre-allocated capacity.
func (b *buffer) Get() (buf *bytes.Buffer) {
	select {
	case buf = <-b.bufChan:
		// reuse existing buffer
	default:
		// create new buffer
		buf = bytes.NewBuffer(make([]byte, 0, maxBufferCap))
	}
	return
}

// Put returns the given Buffer to the buffer pool.
func (b *buffer) Put(buf *bytes.Buffer) {
	buf.Reset()

	// Release buffers over our maximum capacity and re-create a pre-sized
	// buffer to replace it.
	if cap(buf.Bytes()) > maxBufferCap {
		buf = bytes.NewBuffer(make([]byte, 0, maxBufferCap))
	}

	select {
	case b.bufChan <- buf:
	default: // Discard the buffer if the pool is full.
	}
}

// GetPayload returns the payload bytes of the message, the buffer will return
// buffered bytes if the same message pointer comes.
func (b *buffer) GetPayload(msg Message) ([]byte, error) {
	// Find payload from buffer.
	b.mtx.RLock()
	payload, ok := b.payloads[msg]
	b.mtx.RUnlock()
	if ok {
		return payload, nil
	}

	// Payload not found in buffer, create the payload and put it into buffer.
	buf := b.Get()
	if err := msg.Serialize(buf); err != nil {
		return nil, err
	}
	payload = make([]byte, buf.Len())
	copy(payload, buf.Bytes())
	b.Put(buf)

	// Evict the least recently used entry (back of the list) if the the new
	// entry would exceed the size limit for the map.  Also reuse the list
	// node so a new one doesn't have to be allocated.
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if len(b.payloads)+1 > maxBufferedPayloads {
		node := b.msgList.Back()

		// Evict least recently used item.
		delete(b.payloads, node.Value.(Message))

		// Reuse the list node of the item that was just evicted for the
		// new item.
		node.Value = msg
		b.msgList.MoveToFront(node)
		b.payloads[msg] = payload
		return payload, nil
	}

	// The limit hasn't been reached yet, so just add the new item.
	b.msgList.PushFront(msg)
	b.payloads[msg] = payload

	return payload, nil
}
