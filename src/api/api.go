package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/dafraer/messenger/src/store"
	"github.com/dafraer/messenger/src/token"
	"github.com/dafraer/messenger/src/ws"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

type Server struct {
	manager      *ws.Manager
	logger       *zap.SugaredLogger
	tokenManager *token.JWTManager
	store        *store.Storage
}

type authData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func New(manager *ws.Manager, logger *zap.SugaredLogger, tokenManager *token.JWTManager, store *store.Storage) *Server {
	return &Server{
		manager:      manager,
		logger:       logger,
		tokenManager: tokenManager,
		store:        store,
	}
}

func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	http.HandleFunc("/ws", s.authorize(s.serveWS))
	http.HandleFunc("/register", s.handleRegister)
	http.HandleFunc("/login", s.handleLogin)
	ch := make(chan error)
	go func() {
		defer close(ch)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ch <- err
			return
		}
		ch <- nil
	}()
	select {
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			return err
		}
		err := <-ch
		if err != nil {
			return err
		}
	case err := <-ch:
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

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	//Get user data from request
	var body authData
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error decoding json", "error", err)
		return
	}
	//Register user
	if len(body.Password) < minPasswordLength {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error generating hash", "error", err)
		return
	}

	//Save user to the db
	if err := s.store.NewUser(r.Context(), &store.User{Username: body.Username, Password: string(hash)}); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			http.Error(w, "user exists", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error saving user", "error", err)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	//Get user data from request
	var body authData
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error decoding json", "error", err)
		return
	}

	//Get user from the database
	user, err := s.store.GetUser(r.Context(), body.Username)
	if err != nil {
		s.logger.Errorw("Error getting user from the database", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Check password validity
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		http.Error(w, "wrong password", http.StatusBadRequest)
		return
	}

	//Create access token
	accessToken, err := s.tokenManager.NewToken(body.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error creating token:", "error", err)
		return
	}
	//Create a json object from tokens
	response, err := json.Marshal(accessToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error marshaling json:", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

func (s *Server) authorize(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims, err := s.tokenManager.Verify(tokenString)
		if err != nil {
			http.Error(w, "Error validating token", http.StatusUnauthorized)
		}
		if claims.Subject != r.FormValue("username") {
			http.Error(w, "Token subject and user dont match", http.StatusBadRequest)
		}
		fn(w, r)
	}
}
