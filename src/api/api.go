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

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Server struct {
	manager      *ws.Manager
	logger       *zap.SugaredLogger
	tokenManager *token.JWTManager
	store        *store.Storage
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
	//Serves websocket connections
	http.HandleFunc("/ws", s.authorize(s.serveWS))
	//handles registering logic
	http.HandleFunc("/register", s.handleRegister)
	//handles login logic
	http.HandleFunc("/login", s.handleLogin)
	//PUBLIC Returns non-sensitive user data
	http.HandleFunc("/user/{username}", s.handleUser)
	//PRIVATE returns list of user's chats
	http.HandleFunc("/chats/{username}", s.authorize(s.handleChats))
	//PRIVATE returns messages from a chat by id
	http.HandleFunc("/messages/{chatId}", s.authorize(s.handleMessages))
	//PUBLIC returns users with similar username
	http.HandleFunc("/search/{username}", s.handleSearch)
	//PRIVATE creates new chat
	http.HandleFunc("/newChat", s.authorize(s.handleNewChat))
	//TODO leave chat and delete chat handlers
	ch := make(chan error)
	go func() {
		defer close(ch)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	client := ws.NewClient(conn, s.manager, r.PathValue("username"))

	//Add client to client list
	if err := s.manager.AddClient(client); err != nil {
		s.logger.Errorw("error adding client", "error", err)
		http.Error(w, "Error adding client", http.StatusInternalServerError)
	}

	//Start read/write processes in separate goroutines
	go client.ReadMessages()
	go client.WriteMessages()
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	//Get user data from request
	var body authRequest
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
	if err := s.store.NewUser(r.Context(), body.Username, string(hash)); err != nil {
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
	var body authRequest
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
		s.logger.Debugw("auth header", "header", authHeader)
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
		r = r.WithContext(context.WithValue(r.Context(), "username", claims.Subject))
		s.logger.Debugw("authorizing user", "username", claims.Subject)
		fn(w, r)
	}
}

// handleUser writes User object as a response
func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	//Get username from the query
	username := r.PathValue("username")

	//Get user from the db
	user, err := s.store.GetUser(r.Context(), username)

	if err != nil {
		s.logger.Errorw("Error getting user from the database", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Marshal response body
	response, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
		return
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleChats handles requests at /chats/{username} endpoint. Writes list of Chat objects that are owned by the user as a response
func (s *Server) handleChats(w http.ResponseWriter, r *http.Request) {
	//Get username form the query
	username := r.PathValue("username")

	//Check that user is authorized
	if r.Context().Value("username") != username {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Get chat from the database
	chats, err := s.store.GetChats(r.Context(), username)
	if err != nil {
		s.logger.Errorw("Error getting chats", "error", err)
		http.Error(w, "Error getting chats", http.StatusInternalServerError)
		return
	}

	//Marshal response
	response, err := json.Marshal(chats)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleMessages handles requests at /{chatId}/messages endpoint. It is used to get messages from a specific chat
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	//get chatId from the query
	chatId := r.PathValue("chatId")

	//Check if user is authorised
	chat, err := s.store.GetChat(r.Context(), chatId)
	if err != nil {
		s.logger.Errorw("Error getting chat", "error", err)
		http.Error(w, "Error getting chat", http.StatusInternalServerError)
	}
	if chat.Owner != r.Context().Value("username") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Get messages from the database
	messages, err := s.store.GetMessages(r.Context(), chatId)
	if err != nil {
		s.logger.Errorw("Error getting messages from the database", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Marshal response
	response, err := json.Marshal(messages)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleNewChat receives a chat object and creates new chat with the owner and members specified and that object
func (s *Server) handleNewChat(w http.ResponseWriter, r *http.Request) {
	var body store.Chat
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.logger.Errorw("Error decoding json", "error", err)
		http.Error(w, "Error decoding json", http.StatusInternalServerError)
	}
	if body.Owner != r.Context().Value("username") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id, err := s.store.NewChat(r.Context(), &body)
	if err != nil {
		s.logger.Errorw("Error creating chat", "error", err)
		http.Error(w, "Error creating chat", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(id)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
	}

	if _, err = w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleSearch receives username to search for and writes a slice of similar usernames as a response
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	//Get username to search for from the query
	username := r.PathValue("username")

	//Find usernames in db
	usernames, err := s.store.FindSimilarUsers(r.Context(), username)
	if err != nil {
		s.logger.Errorw("Error getting users from the database", "error", err)
		http.Error(w, "Error getting users from the database", http.StatusInternalServerError)
	}

	//Marshal response body
	response, err := json.Marshal(usernames)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}
