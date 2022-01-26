package handlers

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Home struct {
	l         *zap.Logger
	db        *mongo.Client
	conf      *oauth2.Config
	sess      *sessions.CookieStore
	templates *embed.FS
}

func NewHome(
	l *zap.Logger,
	db *mongo.Client,
	sess *sessions.CookieStore,
	templates *embed.FS,
	googleKey string,
	googleSecret string,
	sessionKey string,
	host string, port string,
) *Home {
	conf := &oauth2.Config{
		ClientID:     googleKey,
		ClientSecret: googleSecret,
		RedirectURL:  fmt.Sprintf("http://%s%s/auth", host, port),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
	return &Home{l, db, conf, sess, templates}
}

func (h *Home) LandingPage(w http.ResponseWriter, r *http.Request) {
	// Create token to protect against CSRF attacks mid-signin
	randToken := randStateToken()
	// Assign the "mysterious" user a session
	newSession, err := h.sess.Get(r, "research_survey_session")
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	newSession.Values["state"] = randToken
	newSession.Save(r, w)
	loginURL := h.conf.AuthCodeURL(randToken)
	t := template.Must(template.New("landing-page").ParseFS(*h.templates, "templates/index.html"))
	err = t.ExecuteTemplate(w, "index.html", struct {
		GoogleLoginURL string
	}{GoogleLoginURL: loginURL})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}

func (h *Home) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	session, err := h.sess.Get(r, "research_survey_session")
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	if session.Values["state"] != r.URL.Query().Get("state") {
		http.Error(w, "Authorization Unsuccessful", http.StatusUnauthorized)
		return
	}
	tok, err := h.conf.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Authorization Unsuccessful", http.StatusBadRequest)
		return
	}
	client := h.conf.Client(oauth2.NoContext, tok)
	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "Authorization Unsuccessful", http.StatusBadRequest)
		return
	}
	defer email.Body.Close()
	data, _ := ioutil.ReadAll(email.Body)
	h.l.Info("Email Login", zap.String("email", string(data)))
	w.Write(data)
}

func randStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
