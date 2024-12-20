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

	//Create jwt token manager
	jwtManager := token.New("secret_key")

	//Create context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	//Create storage
	uri := "mongodb://localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	storage := store.New(client)
	if err := storage.Init(ctx); err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	//Create the server
	s := api.New(manager, sugar, jwtManager, storage)

	//Run server
	if err := s.Run(ctx, "localhost:8080"); err != nil {
		panic(err)
	}

}
