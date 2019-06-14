package msg

import (
	"container/list"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p"
)

// ToMsg defines the function to convert a serializable object to a p2p.Message.
type ToMsg func(obj common.Serializable) p2p.Message

// cache is a message cache associate with the origin serializable object, it is
// used to re-use the message of a Tx or Block, so the message buffer pool can
// re-use the cached message payload bytes.
type cache struct {
	toMsg ToMsg
	mtx   sync.Mutex
	msgs  map[common.Serializable]p2p.Message
	list  *list.List
	limit int
}

// Get returns the cached message according to the passed item, creates and
// cache a new message if cached message found.
//
// This function is safe for concurrent access.
func (m *cache) Get(obj common.Serializable) p2p.Message {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	msg, ok := m.msgs[obj]
	if ok {
		return msg
	}

	msg = m.toMsg(obj)

	// When the limit is zero, nothing can be added to the map, so just
	// return.
	if m.limit == 0 {
		return msg
	}

	// Evict the least recently used entry (back of the list) if the the new
	// entry would exceed the size limit for the map.  Also reuse the list
	// node so a new one doesn't have to be allocated.
	if len(m.msgs)+1 > m.limit {
		node := m.list.Back()
		lru := node.Value.(p2p.Message)

		// Evict least recently used item.
		delete(m.msgs, lru)

		// Reuse the list node of the item that was just evicted for the
		// new item.
		node.Value = msg
		m.list.MoveToFront(node)
		m.msgs[obj] = msg
		return msg
	}

	// The limit hasn't been reached yet, so just add the new item.
	m.list.PushFront(msg)
	m.msgs[obj] = msg
	return msg
}

// NewCache creates a message cache with the given size limit and the message
// convert function.
func NewCache(limit int, toMsg ToMsg) *cache {
	return &cache{
		toMsg: toMsg,
		msgs:  make(map[common.Serializable]p2p.Message, limit),
		list:  list.New(),
		limit: limit,
	}
}
