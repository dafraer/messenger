package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/dafraer/messenger/src/store"
	"github.com/dafraer/messenger/src/token"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dafraer/messenger/src/api"
	"github.com/dafraer/messenger/src/ws"
	"go.uber.org/zap"
)

func main() {
	//Check that we got 3 arguments
	if len(os.Args) != 4 {
		panic("Signing key, Server address and Mongo URI must be passed as arguments")
	}
	signingKey := os.Args[1]
	//localhost:8080
	serverAddress := os.Args[2]
	//mongodb://localhost:27017
	mongoURI := os.Args[3]

	//Create logger
	logger, err := zap.NewDevelopment()
	var sugar *zap.SugaredLogger
	if err != nil {
		panic(fmt.Errorf("error while creating new Logger, %v ", err))
	}

	//Create sugared logger
	if logger != nil {
		sugar = logger.Sugar()
	}

	//Create jwt token manager
	jwtManager := token.New(signingKey)

	//Create default context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	//Create storage
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(err)
	}
	storage := store.New(client)
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	//Create a websocket manager
	manager := ws.NewManager(sugar, storage)

	//Create the server
	s := api.New(manager, sugar, jwtManager, storage)

	//Run the server
	if err := s.Run(ctx, serverAddress); err != nil {
		panic(err)
	}

}
