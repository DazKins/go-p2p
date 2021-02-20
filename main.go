package main

import (
	"fmt"
	"os"
	"p2p/app"
	"p2p/config"
	"p2p/network"
	"p2p/node"

	"go.uber.org/dig"
)

func main() {
	c := dig.New()

	c.Provide(func() *config.Config {
		return &config.Config{
			Port:                       os.Args[1],
			InitialConnectionAddresses: os.Args[2:],
		}
	})
	c.Provide(network.NewConnectionHandler)
	c.Provide(node.NewNode)
	c.Provide(app.NewApp)

	err := c.Invoke(func(app *app.App) {
		app.Run()
	})

	if err != nil {
		fmt.Println(err.Error())
	}
}
