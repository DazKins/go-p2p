package network

import (
	"fmt"
	"net"
	"p2p/config"
)

const MAX_CONNECTIONS = 10
const MAX_MESSAGE_SIZE = 256

type ConnectionEvent interface{}

type ConnectionEventNew struct{}
type ConnectionEventMessage struct {
	Message []byte
}
type ConnectionEventClosed struct{}

type ConnectionHandler struct {
	config *config.Config

	conns [MAX_CONNECTIONS]net.Conn

	connectionEventChannels [MAX_CONNECTIONS](chan ConnectionEvent)
}

func NewConnectionHandler(config *config.Config) *ConnectionHandler {
	connectionEventChannels := [MAX_CONNECTIONS](chan ConnectionEvent){}

	for i := range connectionEventChannels {
		connectionEventChannels[i] = make(chan ConnectionEvent)
	}

	return &ConnectionHandler{
		config: config,

		conns: [MAX_CONNECTIONS]net.Conn{},

		connectionEventChannels: connectionEventChannels,
	}
}

func (ch *ConnectionHandler) Start() {
	go ch.StartNewConnectionListener(ch.config.Port)

	for _, connectionAddress := range ch.config.InitialConnectionAddresses {
		go ch.attemptConnectToAddress(connectionAddress)
	}
}

func (ch *ConnectionHandler) attemptConnectToAddress(connectionAddress string) {
	conn, err := net.Dial("tcp", connectionAddress)
	if err != nil {
		fmt.Printf("Error connecting to %s: %s\n", connectionAddress, err.Error())
		return
	}

	ch.handleNewConnection(conn)
}

func (ch *ConnectionHandler) SendMessageToConnection(connection int, message []byte) error {
	if ch.conns[connection] == nil {
		return ErrorConnectionClosed
	}

	if len(message) > MAX_MESSAGE_SIZE {
		return ErrorMessageTooLarge
	}

	ch.conns[connection].Write(message)

	return nil
}

func (ch *ConnectionHandler) StartNewConnectionListener(port string) {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()

		if err != nil {
			fmt.Println(err)
			return
		}

		ch.handleNewConnection(conn)
	}
}

func (ch *ConnectionHandler) GetConnectionEventChannels() [10]chan ConnectionEvent {
	return ch.connectionEventChannels
}

func (ch *ConnectionHandler) findFreeConnectionSlot() int {
	for i, conn := range ch.conns {
		if conn != nil {
			continue
		}

		return i
	}

	return -1
}

func (ch *ConnectionHandler) handleNewConnection(conn net.Conn) {
	index := ch.findFreeConnectionSlot()
	ch.conns[index] = conn
	ch.connectionEventChannels[index] <- ConnectionEventNew{}

	go ch.connectionReader(index)
}

func (ch *ConnectionHandler) connectionReader(index int) {
	for {
		message := make([]byte, MAX_MESSAGE_SIZE)
		_, err := ch.conns[index].Read(message)

		if err != nil {
			ch.conns[index] = nil
			ch.connectionEventChannels[index] <- ConnectionEventClosed{}
			break
		}

		// go func() {
		ch.connectionEventChannels[index] <- ConnectionEventMessage{
			Message: message,
		}
		// }()
	}
}
