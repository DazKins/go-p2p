package node

import (
	"errors"
	"fmt"
	"p2p/network"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	id                  uuid.UUID
	connectionHandler   *network.ConnectionHandler
	nodeIdConnectionMap map[uuid.UUID]int
	acceptedConnections map[int]bool

	messageChannelMap        map[uuid.UUID](chan []byte)
	newNodeConnectionChannel chan uuid.UUID
}

func NewNode(connectionHandler *network.ConnectionHandler) *Node {
	return &Node{
		id:                  uuid.New(),
		connectionHandler:   connectionHandler,
		nodeIdConnectionMap: make(map[uuid.UUID]int),
		acceptedConnections: make(map[int]bool),

		messageChannelMap:        make(map[uuid.UUID](chan []byte)),
		newNodeConnectionChannel: make(chan uuid.UUID),
	}
}

func (n *Node) Run() {
	fmt.Printf("------ STARTING %s ------\n", n.id.String())

	n.connectionHandler.Start()

	for connection := range n.connectionHandler.GetConnectionEventChannels() {
		go n.runConnectionEventListener(connection)
	}
}

func (n *Node) SendMessageTo(nodeId uuid.UUID, message []byte) error {
	connection, ok := n.nodeIdConnectionMap[nodeId]

	if !ok {
		return ErrorNodeNotConnected
	}

	connectionMessage := append([]byte("MSG:"), message...)

	err := n.connectionHandler.SendMessageToConnection(connection, connectionMessage)

	if err != nil {
		switch err {
		case network.ErrorMessageTooLarge:
			return ErrorMessageTooLarge
		case network.ErrorConnectionClosed:
			return ErrorNodeNotConnected
		default:
			return errors.New("Unexpected error sending message")
		}
	}

	return nil
}

func (n *Node) GetConnectedNodes() []uuid.UUID {
	nodeIds := []uuid.UUID{}
	for nodeId := range n.nodeIdConnectionMap {
		nodeIds = append(nodeIds, nodeId)
	}
	return nodeIds
}

func (n *Node) GetNewNodeConnectionChannel() <-chan uuid.UUID {
	return n.newNodeConnectionChannel
}

func (n *Node) runConnectionEventListener(connection int) {
	for {
		connectionEvent := <-n.connectionHandler.GetConnectionEventChannels()[connection]

		n.handleConnectionEvent(connection, connectionEvent)
	}
}

func (n *Node) handleConnectionEvent(connection int, connectionEvent network.ConnectionEvent) {
	if _, ok := connectionEvent.(network.ConnectionEventNew); ok {
		go n.initiateNewConnection(connection)
		return
	}

	if _, ok := connectionEvent.(network.ConnectionEventMessage); ok {
		message := connectionEvent.(network.ConnectionEventMessage).Message

		n.handleConnectionMessage(connection, message)

		return
	}

	if _, ok := connectionEvent.(network.ConnectionEventClosed); ok {
		nodeId, err := n.getNodeIdFromConnection(connection)

		if err != nil {
			fmt.Printf("Tried to disconnect from %d which doesn't have a Node ID\n", connection)
			return
		}

		n.acceptedConnections[connection] = false
		delete(n.nodeIdConnectionMap, nodeId)

		fmt.Printf("Disconnected from %s\n", nodeId.String())

		return
	}
}

func (n *Node) handleConnectionMessage(connection int, message []byte) {
	if string(message[:3]) == "ID:" {
		nodeIdBytes := message[3:19]
		nodeId, err := uuid.FromBytes(nodeIdBytes)

		if err != nil {
			fmt.Printf("Error parsing node ID\n")
			return
		}

		if _, ok := n.nodeIdConnectionMap[nodeId]; ok {
			return
		}

		fmt.Printf("Made node connection to %s\n", nodeId.String())

		n.nodeIdConnectionMap[nodeId] = connection

		n.connectionHandler.SendMessageToConnection(connection, []byte("ACC"))

		n.newNodeConnectionChannel <- nodeId
		n.messageChannelMap[nodeId] = make(chan []byte)
	} else if string(message[:3]) == "ACC" {
		n.acceptedConnections[connection] = true
	} else if string(message[:4]) == "MSG:" {
		nodeMessage := message[4:]

		nodeId, err := n.getNodeIdFromConnection(connection)

		if err != nil {
			fmt.Printf("Error publishing message, Node ID: %s doesn't exist\n", nodeId.String())
			return
		}

		n.publishNodeMessage(nodeId, nodeMessage)
	}
}

func (n *Node) publishNodeMessage(nodeId uuid.UUID, message []byte) {
	go func() { n.messageChannelMap[nodeId] <- message }()
}

func (n *Node) GetMessageChannel(nodeId uuid.UUID) (chan []byte, error) {
	if _, ok := n.messageChannelMap[nodeId]; !ok {
		return nil, fmt.Errorf("Couldn't get message channel for Node ID: %s\n", nodeId.String())
	}

	return n.messageChannelMap[nodeId], nil
}

func (n *Node) initiateNewConnection(connection int) {
	for {
		time.Sleep(time.Second)

		if n.acceptedConnections[connection] {
			break
		}

		message := []byte("ID:")
		message = append(message, n.id[:]...)

		n.connectionHandler.SendMessageToConnection(connection, message)
	}
}

func (n *Node) getNodeIdFromConnection(connection int) (uuid.UUID, error) {
	for nodeId, nodeIdConnection := range n.nodeIdConnectionMap {
		if nodeIdConnection != connection {
			continue
		}

		return nodeId, nil
	}

	return uuid.UUID{}, fmt.Errorf("No nodeId for connection: %d\n", connection)
}
