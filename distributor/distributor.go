package distributor

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

const (
	// buffSize is the data buffer size for each pipe way, so is 2MB for each
	// pipe instance.  Most of messages in pipe is smaller than 1MB, so one
	// message can be distributed by one loop.  If there are 100 pipe instances,
	// they will take 200MB memory cache, is not too large for a computer that
	// have a 8GB(1024MB*8) or larger memory.
	buffSize = 1024 << 10 // 1MB
)

// simpleAddr implements the net.Addr interface with two struct fields
type simpleAddr struct {
	net, addr string
}

// String returns the address.
//
// This is part of the net.Addr interface.
func (a simpleAddr) String() string {
	return a.addr
}

// Network returns the network.
//
// This is part of the net.Addr interface.
func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}

// pipe represent a pipeline from the local connection to the mapping net
// address.
type pipe struct {
	inlet  net.Conn
	outlet net.Conn
}

// start creates the data pipeline between inlet and outlet.
func (p *pipe) start() {
	// Create two way flow between inlet and outlet.
	go p.flow(p.inlet, p.outlet)
	go p.flow(p.outlet, p.inlet)
}

// close closes the data pipeline between inlet and outlet.
func (p *pipe) close() {
	_ = p.inlet.Close()
	_ = p.outlet.Close()
}

// flow creates a one way flow between from and to.
func (p *pipe) flow(from net.Conn, to net.Conn) {
	// Catch panic message and close the flow.
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("pipe flow error %s", err)
		}
		p.close()
	}()
	buf := make([]byte, buffSize)
	for {
		n, err := from.Read(buf)
		if err != nil {
			panic(err)
		}
		_, err = to.Write(buf[:n])
		if err != nil {
			panic(err)
		}
	}
}

// Distributor will mapping connections according to the port number, and
// distribute them to the related net address.
type Distributor struct {
	mtx     sync.Mutex
	mapping map[int]net.Addr
	quit    chan struct{}
}

// Mapping add a port to net address mapping to the distributor, if the net
// address is invalid or the port was mapped returns error.
func (d *Distributor) Mapping(port int, addr string) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if na, ok := d.mapping[port]; ok {
		return fmt.Errorf("port %d was mapped to %s", port, na)
	}

	na, err := addrStringToNetAddr(addr)
	if err != nil {
		return fmt.Errorf("mapping invalid address %s, %s", addr, err)
	}

	d.mapping[port] = na
	return nil
}

// Start starts the distributor, it initialize local listeners according to the
// registered mapping.
func (d *Distributor) Start() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	netAddrs := make(map[net.Addr]net.Addr, len(d.mapping)*2)
	for port, na := range d.mapping {
		addr := fmt.Sprintf(":%d", port)
		netAddrs[simpleAddr{net: "tcp4", addr: addr}] = na
		netAddrs[simpleAddr{net: "tcp6", addr: addr}] = na
	}

	for in, out := range netAddrs {
		listener, err := net.Listen(in.Network(), in.String())
		if err != nil {
			log.Warnf("Can't listen on %s: %v", in, err)
			continue
		}
		go d.listenHandler(listener, out)
	}
}

// listenHandler accepts incoming connections on a given listener.  It must be
// run as a goroutine.
func (d *Distributor) listenHandler(listener net.Listener, addr net.Addr) {
out:
	for {
		select {
		default:
			inlet, err := listener.Accept()
			if err != nil {
				log.Errorf("Can't accept connection: %v", err)
				continue
			}
			// Attempt to connect to outlet address.
			outlet, err := net.Dial(addr.Network(), addr.String())
			if err != nil {
				// If the outlet address can not be connected, close the inlet
				// connection to signal the pipe can not be created.
				_ = inlet.Close()
				continue
			}
			go d.newPipe(inlet, outlet)

		case <-d.quit:
			break out
		}
	}

	_ = listener.Close()
}

// newPipe creates a new pipe between inlet connection and outlet address.  It
// must be run as a goroutine.
func (d *Distributor) newPipe(inlet net.Conn, outlet net.Conn) {
	p := pipe{inlet: inlet, outlet: outlet}
	p.start()
	select {
	case <-d.quit:
		p.close()
	}
}

// Stop stops the distributor, this method can only call once
func (d *Distributor) Stop() {
	d.mtx.Lock()
	close(d.quit)
}

// NewDistributor creates a new distributor instance.
func New() *Distributor {
	return &Distributor{
		mapping: make(map[int]net.Addr),
		quit:    make(chan struct{}),
	}
}

// addrStringToNetAddr takes an address in the form of 'host:port' and returns
// a net.Addr which maps to the original address with any host names resolved
// to IP addresses.  It also handles tor addresses properly by returning a
// net.Addr that encapsulates the address.
func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

	// Skip if host is already an IP address.
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

	// Attempt to look up an IP address associated with the parsed host.
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}
