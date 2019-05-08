package addrmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// AddrManager provides a concurrency safe address manager for caching potential
// peers on the network.
type AddrManager struct {
	mtx       sync.Mutex
	peersFile string
	addrIndex map[[33]byte]*PeerAddr // address key to ka for all addrs.
	started   int32
	shutdown  int32
	wg        sync.WaitGroup
	quit      chan struct{}
}

type serializedAddress struct {
	PID  [33]byte
	Addr string
}

type serializedAddrManager struct {
	Addresses []*serializedAddress
}

const (
	// dumpAddressInterval is the interval used to dump the address
	// cache to disk for future use.
	dumpAddressInterval = time.Minute * 10
)

// addressHandler is the main handler for the address manager.  It must be run
// as a goroutine.
func (a *AddrManager) addressHandler() {
	dumpAddressTicker := time.NewTicker(dumpAddressInterval)
	defer dumpAddressTicker.Stop()
out:
	for {
		select {
		case <-dumpAddressTicker.C:
			a.savePeers()

		case <-a.quit:
			break out
		}
	}
	a.savePeers()
	a.wg.Done()
}

// savePeers saves all the known addresses to a file so they can be read back
// in at next run.
func (a *AddrManager) savePeers() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	w, err := os.Create(a.peersFile)
	if err != nil {
		log.Errorf("Error opening file %s: %v", a.peersFile, err)
		return
	}
	defer w.Close()

	// First we make a serialisable datastructure so we can encode it to
	// json.
	sam := new(serializedAddrManager)
	sam.Addresses = make([]*serializedAddress, 0, len(a.addrIndex))
	for k, v := range a.addrIndex {
		ska := new(serializedAddress)
		copy(ska.PID[:], k[:])
		ska.Addr = v.String()
		// Tried and refs are implicit in the rest of the structure
		// and will be worked out from context on unserialisation.
		sam.Addresses = append(sam.Addresses, ska)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&sam); err != nil {
		log.Errorf("Failed to encode file %s: %v", a.peersFile, err)
		return
	}
}

// loadPeers loads the known address from the saved file.  If empty, missing, or
// malformed file, just don't load anything and start fresh
func (a *AddrManager) loadPeers() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	err := a.deserializePeers(a.peersFile)
	if err != nil {
		log.Errorf("Failed to parse file %s: %v", a.peersFile, err)
		// if it is invalid we nuke the old one unconditionally.
		err = os.Remove(a.peersFile)
		if err != nil {
			log.Warnf("Failed to remove corrupt peers file %s: %v",
				a.peersFile, err)
		}
		return
	}
	log.Infof("Loaded %d addresses from file '%s'", len(a.addrIndex), a.peersFile)
}

func (a *AddrManager) deserializePeers(filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	r, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("%s error opening file: %v", filePath, err)
	}
	defer r.Close()

	var sam serializedAddrManager
	dec := json.NewDecoder(r)
	err = dec.Decode(&sam)
	if err != nil {
		return fmt.Errorf("error reading %s: %v", filePath, err)
	}

	for _, v := range sam.Addresses {
		na, err := AddrStringToPeerAddr(v.PID, v.Addr)
		if err != nil {
			return fmt.Errorf("failed to deserialize netaddress "+
				"%s: %v", v.Addr, err)
		}
		a.addrIndex[v.PID] = na
	}

	return nil
}

// Start begins the core address handler which manages a pool of known
// addresses, timeouts, and interval based writes.
func (a *AddrManager) Start() {
	// Already started?
	if atomic.AddInt32(&a.started, 1) != 1 {
		return
	}

	// Load peers we already know about from file.
	a.loadPeers()

	// Start the address ticker to save addresses periodically.
	a.wg.Add(1)
	go a.addressHandler()
}

// Stop gracefully shuts down the address manager by stopping the main handler.
func (a *AddrManager) Stop() {
	if atomic.AddInt32(&a.shutdown, 1) != 1 {
		log.Warnf("Address manager is already in the process of " +
			"shutting down")
		return
	}

	log.Infof("Address manager shutting down")
	close(a.quit)
	a.wg.Wait()
}

// AddAddress adds a new address to the address manager.  It enforces a max
// number of addresses and silently ignores duplicate addresses.  It is
// safe for concurrent access.
func (a *AddrManager) AddAddress(addr *PeerAddr) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.addrIndex[addr.PID] = addr
}

// GetAddress returns the network address according to the given PID and Encode.
func (a *AddrManager) GetAddress(pid [33]byte) *PeerAddr {
	// Protect concurrent access.
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.addrIndex[pid]
}

// New returns a new address manager.
// Use Start to begin processing asynchronous address updates.
func New(dataDir string) *AddrManager {
	am := AddrManager{
		peersFile: filepath.Join(dataDir, "peers.json"),
		addrIndex: make(map[[33]byte]*PeerAddr),
		quit:      make(chan struct{}),
	}
	return &am
}
