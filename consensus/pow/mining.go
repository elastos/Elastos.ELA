package pow

import (
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/ledger"
	"fmt"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const (
	MSGHASKBLOCK = "hashblock"
	MSGHASKTX    = "hashtx"
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
	publisher, _ := zmq.NewSocket(zmq.PUB)
	defer publisher.Close()

	bindIP := fmt.Sprintf("tcp://*:%d", config.Parameters.PowConfiguration.MiningSelfPort)
	publisher.Bind(bindIP)
	for {
		publisher.Send(MSGHASKTX+"==Coming from elacoin node, glad to see you, Timestamp:"+string(time.Now().Unix()), zmq.SNDMORE)
		time.Sleep(time.Second * 3)
		//TODO transfer to verify and save block handling process
	}
}
