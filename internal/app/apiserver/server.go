package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"github.com/zlyaptica/http-rest-api/internal/app/model"
	"github.com/zlyaptica/http-rest-api/internal/app/store"
)

const (
	sessionName        = "booklib"
	ctxKeyUser  ctxKey = iota
	ctxKeyRequestID
)

var (
	errIncorrectEmailOrPassword = errors.New("incorrect email or password")
	errNotAuthenticated         = errors.New("not authenticated")
	errNoPermission             = errors.New("no permission")
)

type ctxKey int8

type server struct {
	router       *mux.Router
	logger       *logrus.Logger
	store        store.Store
	sessionStore sessions.Store
}

func newServer(store store.Store, sessionStore sessions.Store) *server {
	s := &server{
		router:       mux.NewRouter(),
		logger:       logrus.New(),
		store:        store,
		sessionStore: sessionStore,
	}

	s.configureRouter()

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) configureRouter() {
	s.router.Use(s.setRequestID)
	s.router.Use(s.logRequest)
	s.router.Use(s.setCORS)
	s.router.Use(s.authenticateUser)
	s.router.HandleFunc("/users", s.handleUsersCreate()).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/sessions", s.handleSessionsCreate()).Methods("POST", "OPTIONS")

	s.router.HandleFunc("/posts", s.handlePostsGet()).Methods("GET")
	s.router.HandleFunc("/user/{id}", s.handleGetUserByID()).Methods("GET")
	s.router.HandleFunc("/user/{id}/posts", s.handlePostsGetByUserID()).Methods("GET")

	private := s.router.PathPrefix("/private").Subrouter()
	private.Use(s.authorizeUser)

	private.HandleFunc("/whoami", s.handleWhoami())
	private.HandleFunc("/posts", s.handlePostsCreate()).Methods("POST", "OPTIONS")
	private.HandleFunc("/posts/{id}", s.handlePostGet()).Methods("GET", "OPTIONS")
	private.HandleFunc("/posts/{id}", s.handlePostDelete()).Methods("DELETE", "OPTIONS")
	private.HandleFunc("/posts/{id}", s.handlePostUpdate()).Methods("PUT", "OPTIONS")

	private.HandleFunc("/posts/{id}/star", s.handleStarGive()).Methods("POST", "OPTIONS")
	private.HandleFunc("/posts/{id}/star", s.handleStarTake()).Methods("DELETE", "OPTIONS")
}

func (s *server) setCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *server) setRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, id)))
	})
}

func (s *server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"request_id":  r.Context().Value(ctxKeyRequestID),
		})
		logger.Infof("started %s %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		var level logrus.Level
		switch {
		case rw.code >= 500:
			level = logrus.ErrorLevel
		case rw.code >= 400:
			level = logrus.WarnLevel
		default:
			level = logrus.InfoLevel
		}
		logger.Logf(
			level,
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Now().Sub(start),
		)
	})
}

func (s *server) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		id, ok := session.Values["user_id"]
		if !ok {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, nil)))
			return
		}

		u, err := s.store.User().Find(id.(int))
		if err != nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, nil)))
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u)))
	})
}

func (s *server) authorizeUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(ctxKeyUser).(*model.User)
		if !ok {
			s.error(w, r, http.StatusUnauthorized, errNotAuthenticated)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u)))
	})
}

func (s *server) handleUsersCreate() http.HandlerFunc {
	type request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u := &model.User{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		}
		if err := s.store.User().Create(u); err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		u.Sanitize()
		s.respond(w, r, http.StatusCreated, u)
	}
}

func (s *server) handleSessionsCreate() http.HandlerFunc {
	type request struct {
		Username   string `json:"username"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe bool   `json:"rememberMe"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u, err := s.store.User().FindByEmail(req.Email)
		if err != nil || !u.ComparePassword(req.Password) {
			s.error(w, r, http.StatusUnauthorized, errIncorrectEmailOrPassword)
			return
		}

		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		if !req.RememberMe {
			session.Options.MaxAge = 0
		}
		session.Values["user_id"] = u.ID
		if err := s.sessionStore.Save(r, w, session); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.respond(w, r, http.StatusOK, nil)
	}
}

func (s *server) handlePostsCreate() http.HandlerFunc {
	type request struct {
		Header   string `json:"header"`
		TextPost string `json:"text_post"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		author := r.Context().Value(ctxKeyUser).(*model.User)

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		p := &model.Post{
			Header:   req.Header,
			TextPost: req.TextPost,
			Author:   author,
		}
		if err := s.store.Post().Create(p); err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		s.respond(w, r, http.StatusCreated, p)
	}
}

func (s *server) handlePostDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		user := r.Context().Value(ctxKeyUser).(*model.User)

		post, err := s.store.Post().Find(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		if post.Author.ID != user.ID {
			s.error(w, r, http.StatusUnauthorized, errNoPermission)
			return
		}
		s.store.Post().Delete(id)
	}
}

func (s *server) handlePostUpdate() http.HandlerFunc {
	type request struct {
		Header   string `json:"header"`
		TextPost string `json:"text_post"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		user := r.Context().Value(ctxKeyUser).(*model.User)

		post, err := s.store.Post().Find(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		if post.Author.ID != user.ID {
			s.error(w, r, http.StatusUnauthorized, errNoPermission)
			return
		}
		s.store.Post().Update(req.Header, req.TextPost, id)
	}
}

func (s *server) handleStarGive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		postID, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		starer := r.Context().Value(ctxKeyUser).(*model.User)

		isStarred, err := s.store.Post().IsStarredByUser(starer.ID, postID)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		star := &model.Star{
			Starer: starer,
			Post: &model.Post{
				ID: postID,
			},
		}

		if isStarred {
			star.Post.StarsCount, err = s.store.Post().GetStarsCount(postID)
			s.respond(w, r, http.StatusAccepted, star)
			return
		}
		s.store.Star().Create(star)
		star.Post.StarsCount, err = s.store.Post().GetStarsCount(postID)

		if err != nil {
			s.respond(w, r, http.StatusInternalServerError, err)
			return
		}

		star.Post.IsStarred, err = s.store.Post().IsStarredByUser(starer.ID, postID)

		s.respond(w, r, http.StatusCreated, star)
	}
}

func (s *server) handleStarTake() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		postID, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		starer := r.Context().Value(ctxKeyUser).(*model.User)

		isStarred, err := s.store.Post().IsStarredByUser(starer.ID, postID)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		star := &model.Star{
			Starer: starer,
			Post: &model.Post{
				ID: postID,
			},
		}

		if !isStarred {
			s.respond(w, r, http.StatusAccepted, star)
			return
		}

		s.store.Star().Delete(starer.ID, postID)
		star.Post.IsStarred, err = s.store.Post().IsStarredByUser(starer.ID, postID)
		star.Post.StarsCount, err = s.store.Post().GetStarsCount(postID)

		s.respond(w, r, http.StatusOK, star)
	}
}

func (s *server) handlePostsGet() http.HandlerFunc {
	type response struct {
		Items []model.Post `json:"items"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		posts, err := s.store.Post().FindAll()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		u, ok := r.Context().Value(ctxKeyUser).(*model.User)
		if ok {
			for i := 0; i < len(posts); i++ {
				posts[i].IsStarred, err = s.store.Post().IsStarredByUser(u.ID, posts[i].ID)
				if err != nil {
					s.error(w, r, http.StatusInternalServerError, err)
					return
				}

			}
		}

		resp := &response{
			Items: posts,
		}

		s.respond(w, r, http.StatusOK, resp)
	}
}

func (s *server) handlePostGet() http.HandlerFunc {
	type response struct {
		Item *model.Post `json:"items"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		post, err := s.store.Post().Find(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		resp := &response{
			Item: post,
		}

		s.respond(w, r, http.StatusOK, resp)
	}
}

func (s *server) handleGetUserByID() http.HandlerFunc {
	type response struct {
		User *model.User `json:"user"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		user, err := s.store.User().FindByID(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		resp := &response{
			User: user,
		}

		s.respond(w, r, http.StatusOK, resp)
	}
}

func (s *server) handlePostsGetByUserID() http.HandlerFunc {
	type response struct {
		Items []model.Post `json:"items"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		posts, err := s.store.Post().FindByAuthor(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		u, ok := r.Context().Value(ctxKeyUser).(*model.User)
		if ok {
			for i := 0; i < len(posts); i++ {
				posts[i].IsStarred, err = s.store.Post().IsStarredByUser(u.ID, posts[i].ID)
				if err != nil {
					s.error(w, r, http.StatusInternalServerError, err)
					return
				}

			}
		}

		resp := &response{
			Items: posts,
		}

		s.respond(w, r, http.StatusOK, resp)
	}
}

func (s *server) handleWhoami() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, http.StatusOK, r.Context().Value(ctxKeyUser).(*model.User))
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
