package pow

import (
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/ledger"
	"fmt"

	zmq "github.com/pebbe/zmq4"
)

func ZMQClientSend(MsgBlock ledger.Block) {
	requester, _ := zmq.NewSocket(zmq.REQ)
	defer requester.Close()

	serverIP := fmt.Sprintf("tcp://%s:%d", config.Parameters.PowConfiguration.MiningServerIP,
		config.Parameters.PowConfiguration.MiningServerPort)

	requester.Connect(serverIP)
	requester.Send("Hello world", 0)
}

func ZMQServer() {
	//  Socket to talk to clients
	log.Info("ZMQ Service Start")
	responder, _ := zmq.NewSocket(zmq.REP)
	defer responder.Close()

	bindIP := fmt.Sprintf("tcp://*:%d", config.Parameters.PowConfiguration.MiningSelfPort)
	responder.Bind(bindIP)
	for {
		responder.Recv(0)
		//TODO transfer to verify and save block handling process
	}
}
