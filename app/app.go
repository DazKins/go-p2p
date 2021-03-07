package app

import (
	"fmt"
	"math/rand"
	"p2p/node"
	"time"

	"github.com/google/uuid"
)

type App struct {
	node                    *node.Node
	currentlyConnectedNodes []uuid.UUID
}

func NewApp(node *node.Node) *App {
	return &App{
		node:                    node,
		currentlyConnectedNodes: []uuid.UUID{},
	}
}

func (app *App) Run() {
	app.node.Run()

	go app.runNewNodeConnectionListener()

	for {
		time.Sleep(time.Second)

		nodeIds := app.node.GetConnectedNodes()
		length := len(nodeIds)

		if length == 0 {
			continue
		}

		nodeId := nodeIds[rand.Intn(len(nodeIds))]

		app.node.SendMessageTo(nodeId, []byte("Hello!"))
	}
}

func (app *App) runNewNodeConnectionListener() {
	nodeId := <-app.node.GetNewNodeConnectionChannel()

	app.currentlyConnectedNodes = append(app.currentlyConnectedNodes, nodeId)

	go app.runNodeMessageListener(nodeId)
}

func (app *App) runNodeMessageListener(nodeId uuid.UUID) {
	messageChannel, err := app.node.GetMessageChannel(nodeId)

	if err != nil {
		return
	}

	for {
		message := <-messageChannel

		fmt.Printf("%s : %s\n", nodeId.String(), string(message))
	}
}
