package api

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/dafraer/messenger/src/ws"
	"go.uber.org/zap"
)

type Server struct {
	manager *ws.Manager
	logger  *zap.SugaredLogger
}

func New() *Server {
	//Create websocket manager
	manager := ws.NewManager()

	//Create logger
	logger, err := zap.NewDevelopment()
	var sugar *zap.SugaredLogger
	if logger != nil {
		sugar = logger.Sugar()
	}

	if err != nil {
		panic(fmt.Errorf("error while creating new Logger, %v ", err))
	}
	return &Server{
		manager: manager,
		logger:  sugar,
	}
}

func (s *Server) Run(ctx context.Context, addr string) {
	srv := &http.Server{
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	http.HandleFunc("/ws", s.ServeWS)
	srv.ListenAndServe()
}

func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	//Upgrade http request
	conn, err := s.manager.WSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorw("error upgrading connection", "err", err)
	}
	conn.Close()
}
