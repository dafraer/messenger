package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/dafraer/messenger/src/api"
	"github.com/dafraer/messenger/src/ws"
	"go.uber.org/zap"
)

func main() {
	//Create logger
	logger, err := zap.NewDevelopment()
	var sugar *zap.SugaredLogger
	if logger != nil {
		sugar = logger.Sugar()
	}

	if err != nil {
		panic(fmt.Errorf("error while creating new Logger, %v ", err))
	}
	//Create a websocket manager
	manager := ws.NewManager(sugar)

	//Create the server
	s := api.New(manager, sugar)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	//Run server
	if err := s.Run(ctx, "localhost:8080"); err != nil {
		panic(err)
	}

}
