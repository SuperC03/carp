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
	"github.com/superc03/research_survey/handlers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Embeded Content
//go:embed templates/*
var templates embed.FS

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
		panic("Environmental variable `HOST has not been set.`")
	}
	if host = os.Getenv("HOST"); host == "" {
		host = "localhost"
	}
	if dbUrl = os.Getenv("MONGODB_URL"); dbUrl == "" {
		panic("Environmental variable `MONGODB_URL has not been set.`")
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

	// Initialize Routes
	sm := mux.NewRouter()

	ah := handlers.NewHome(l, db, sess, &templates, googleKey, googleSecret, sessionKey, host, port)
	sm.HandleFunc("/", ah.LandingPage).Methods(http.MethodGet)
	sm.HandleFunc("/auth", ah.GoogleAuth).Methods(http.MethodGet)

	// Start HTTP Server
	s := http.Server{
		Addr:         port,
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
	// Handle Graceful Server Shutdown
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	sig := <-sigChan
	l.Info("Received Termination Signal, Shutting Down", zap.Any("Signal", sig))

	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(tc)
	cancel()
}
