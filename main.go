package main

import (
	"context"
	"embed"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/superc03/carp/handlers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Embeded Content
//go:embed templates/*
var templates embed.FS

//go:embed static/*
var static embed.FS

var (
	host         string
	port         string
	dbUrl        string
	sessionKey   string
	googleKey    string
	googleSecret string
)

func init() {
	if port = os.Getenv("PORT"); port == "" {
		panic("Environmental variable `PORT` has not been set.")
	}
	if host = os.Getenv("HOST"); host == "" {
		host = "localhost"
	}
	if dbUrl = os.Getenv("MONGODB_URL"); dbUrl == "" {
		panic("Environmental variable `MONGODB_URL` has not been set.")
	}
	if sessionKey = os.Getenv("SESSION_KEY"); sessionKey == "" {
		panic("Enviornmental variable `SESSION_KEY` has not been set.")
	}
	if googleKey = os.Getenv("GOOGLE_KEY"); googleKey == "" {
		panic("Enviornmental variable `GOOGLE_KEY` has not been set.")
	}
	if googleSecret = os.Getenv("GOOGLE_SECRET"); googleSecret == "" {
		panic("Enviornmental variable `GOOGLE_SECRET` has not been set.")
	}
}

func main() {
	// Initialize Logger
	l, err := zap.NewProduction()
	if err != nil {
		panic("Could not initialize logger.")
	}

	// Initialize Sessions
	sess := &sessions.CookieStore{
		Options: &sessions.Options{
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteDefaultMode,
			MaxAge:   86400 * 7,
		},
		Codecs: securecookie.CodecsFromPairs([]byte(sessionKey)),
	}

	// Initialize Database
	dbOptions := options.Client().ApplyURI(dbUrl)
	dbContext, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()
	db, err := mongo.Connect(dbContext, dbOptions)
	if err != nil {
		l.Fatal("Could not connect to MongoDB", zap.Error(err))
	}
	err = db.Ping(dbContext, nil)
	if err != nil {
		l.Fatal("Could not connect to MongoDB", zap.Error(err))
	}

	// Initialize Routes
	sm := mux.NewRouter()

	hh := handlers.NewHome(l, db.Database("carp"), sess, &templates, googleKey, googleSecret, sessionKey, host, port)
	sm.HandleFunc("/", hh.LandingPage).Methods(http.MethodGet)
	sm.HandleFunc("/auth", hh.GoogleAuth).Methods(http.MethodGet)

	sh := handlers.NewSurvey(l, db.Database("carp"), sess, &templates)
	surveyRouter := sm.PathPrefix("/survey").Subrouter()
	surveyRouter.Use(sh.UserMiddleware)
	surveyRouter.HandleFunc("/start", sh.StartPage).Methods(http.MethodGet)
	surveyRouter.HandleFunc("/complete", sh.CompletePage).Methods(http.MethodGet, http.MethodPost)
	surveyRouter.HandleFunc("/{code}", sh.QuestionPage).Methods(http.MethodGet, http.MethodPost)

	oh := handlers.NewOther(l, &templates, db.Database("carp"))
	sm.HandleFunc("/wrong_account", oh.WrongAccountPage).Methods(http.MethodGet)
	statsRouter := sm.PathPrefix("/statistics.csv").Subrouter()
	statsRouter.Use(sh.UserMiddleware)
	statsRouter.HandleFunc("", oh.StatisticsPage)

	fileServer := http.FileServer(http.FS(static))
	sm.PathPrefix("/static").Handler(http.StripPrefix("/", fileServer))

	// Start HTTP Server
	s := http.Server{
		Addr:         ":" + port,
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			l.Fatal("A fatal server error has occured", zap.Error(err))
		}
	}()
	l.Info("Server Started")
	// Handle Graceful Server Shutdown
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	sig := <-sigChan
	l.Info("Received Termination Signal, Shutting Down", zap.Any("Signal", sig))

	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(tc)
	cancel()
}
