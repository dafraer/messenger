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
	tokenManager token.Manager
	store        store.Storer
}

// New creates new server
func New(manager *ws.Manager, logger *zap.SugaredLogger, tokenManager token.Manager, store store.Storer) *Server {
	return &Server{
		manager:      manager,
		logger:       logger,
		tokenManager: tokenManager,
		store:        store,
	}
}

// Run runs the server
func (s *Server) Run(ctx context.Context, addr string) error {
	//Create an http server with provided address
	srv := &http.Server{
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Serve the frontend files
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	//Serves websocket connections
	http.HandleFunc("/ws", s.authorize(s.serveWS))
	//handles registering logic
	http.HandleFunc("/register", s.handleRegister)
	//handles login logic
	http.HandleFunc("/login", s.handleLogin)
	//Writes public user data as a response
	http.HandleFunc("/user/{username}", s.handleUser)
	//Writes list of user's chats as a response
	http.HandleFunc("/chats/{username}", s.authorize(s.handleChats))
	//Writes messages from a chat by id as a response
	http.HandleFunc("/messages/{chatId}", s.authorize(s.handleMessages))
	//Creates new chat
	http.HandleFunc("/newChat", s.authorize(s.handleNewChat))
	//Removes user from chat. User can only remove others if they are owner of the chat
	http.HandleFunc("/remove/{chatId}/{username}", s.authorize(s.handleRemove))

	//Run the server
	ch := make(chan error)
	go func() {
		defer close(ch)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
			return
		}
		ch <- nil
	}()

	//Handle graceful shutdown
	select {
	//If SIGINT is called shutdown the server
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		err := <-ch
		if err != nil {
			return err
		}
		//If we get an error from running the server return the error
	case err := <-ch:
		return err
	}
	return nil
}

// serveWS upgrades http request to a websocket connection
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	//Upgrade connection
	conn, err := s.manager.WSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorw("Error upgrading connection", "err", err)
	}

	//Create a new client
	username := r.Context().Value("username")
	client := ws.NewClient(conn, s.manager, username.(string))

	//Add client to client list
	if err := s.manager.AddClient(r.Context(), client); err != nil {
		s.logger.Errorw("Error adding client", "error", err)
		http.Error(w, "Error adding client", http.StatusInternalServerError)
	}

	//Start read/write processes in separate goroutines
	go client.ReadMessages(r.Context())
	go client.WriteMessages(r.Context())
}

// handleRegister registers user using username and password
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	//Get user data from request
	var body authRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Errorw("Error decoding json", "error", err)
		return
	}
	//Check password length
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

// handleLogin logs user in using username and password. It writes JWT token as a response
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
		http.Error(w, "Error getting user from the database", http.StatusInternalServerError)
		return
	}

	//Check password validity
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		http.Error(w, "Wrong password", http.StatusBadRequest)
		return
	}

	//Create access token
	accessToken, err := s.tokenManager.NewToken(body.Username)
	if err != nil {
		s.logger.Errorw("Error creating JWT token:", "error", err)
		http.Error(w, "Error creating JWT token", http.StatusInternalServerError)
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
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// authorize is a middleware that authorizes user by verifying JWT token
func (s *Server) authorize(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//Get authorization header
		authHeader := r.Header.Get("Authorization")

		//Check if the authorization header is empty
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		//Parse the header
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		//Verify the token
		claims, err := s.tokenManager.Verify(tokenString)
		if err != nil {
			http.Error(w, "Error validating token", http.StatusUnauthorized)
		}

		//Pass username of the user as a context value
		r = r.WithContext(context.WithValue(r.Context(), "username", claims.Subject))
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

	//Set password to nil so it's omitted when marshaling
	user.Password = ""

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

// handleChats writes list of Chat objects that are owned by the user as a response
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
		return
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleMessages writes messages from a specific chat as a response
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	//get chatId from the query
	chatId := r.PathValue("chatId")

	//Get chat data from the database
	chat, err := s.store.GetChat(r.Context(), chatId)
	if err != nil {
		s.logger.Errorw("Error getting chat from the database", "error", err)
		http.Error(w, "Error getting chat from the database", http.StatusInternalServerError)
	}

	//Check if user is a member of the chat
	present := false
	for _, v := range chat.Members {
		if v == r.Context().Value("username") {
			present = true
		}
	}
	if !present {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Get messages from the database
	messages, err := s.store.GetMessages(r.Context(), chatId)
	if err != nil {
		s.logger.Errorw("Error getting messages from the database", "error", err)
		http.Error(w, "Error getting messages from the database", http.StatusInternalServerError)
		return
	}

	//Marshal response
	response, err := json.Marshal(messages)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
		return
	}

	//Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleNewChat receives a chat object and creates new chat and writes chat id as a response
func (s *Server) handleNewChat(w http.ResponseWriter, r *http.Request) {
	//Decode request
	var body store.Chat
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.logger.Errorw("Error decoding json", "error", err)
		http.Error(w, "Error decoding json", http.StatusInternalServerError)
		return
	}

	//If user tries to create chat from someone else's name refuse
	if body.Owner != r.Context().Value("username") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Create new chat
	id, err := s.store.NewChat(r.Context(), body.Members, body.Owner)
	if err != nil {
		s.logger.Errorw("Error creating chat", "error", err)
		http.Error(w, "Error creating chat", http.StatusInternalServerError)
		return
	}

	//Set header
	w.Header().Set("Content-Type", "application/json")

	//Marshal response
	response, err := json.Marshal(id)
	if err != nil {
		s.logger.Errorw("Error marshaling json", "error", err)
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
		return
	}

	//Write response
	if _, err = w.Write(response); err != nil {
		s.logger.Errorw("Error writing a response", "error", err)
	}
}

// handleRemove removes user from the chat
func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	chatId := r.PathValue("chatId")

	//Check if user removes others
	if r.Context().Value("username").(string) != r.PathValue("username") {
		//Get chat from the DB to check if user can delete others
		chat, err := s.store.GetChat(r.Context(), chatId)
		if err != nil {
			s.logger.Errorw("Error getting chat", "error", err)
			http.Error(w, "Error getting chat", http.StatusInternalServerError)
			return
		}

		//If user isn't the chat owner refuse
		if chat.Owner != r.Context().Value("username") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	//Delete user from the chat
	if err := s.store.RemoveUserFromChat(r.Context(), r.PathValue("username"), chatId); err != nil {
		s.logger.Errorw("Error leaving chat", "error", err)
		http.Error(w, "Error leaving chat", http.StatusInternalServerError)
	}
}
