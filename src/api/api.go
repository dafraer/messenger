package api

import (
	"context"
	"net"
	"net/http"

	"github.com/dafraer/messenger/src/ws"
	"go.uber.org/zap"
)

type Server struct {
	manager *ws.Manager
	logger  *zap.SugaredLogger
}

func New(manager *ws.Manager, logger *zap.SugaredLogger) *Server {
	return &Server{
		manager: manager,
		logger:  logger,
	}
}

func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	http.HandleFunc("/ws", s.serveWS)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

// serveWS is an http handler that upgrades to a websocket connection
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	//Upgrade connection
	conn, err := s.manager.WSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorw("error upgrading connection", "err", err)
	}

	//Create a new client
	client := ws.NewClient(conn, s.manager)

	//Add client to client list
	s.manager.AddClient(client)

	//Start read/write proccesses in seperate gproutines
	go client.ReadMessages()
	go client.WriteMessages()
}
