package handlers

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/superc03/carp/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Home struct {
	l         *zap.Logger
	db        *mongo.Database
	conf      *oauth2.Config
	sess      *sessions.CookieStore
	templates *embed.FS
}

func NewHome(
	l *zap.Logger,
	db *mongo.Database,
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
	newSession, err := h.sess.Get(r, "carp")
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	// TODO Conditional Render 'Login with Google' or 'Continue Survey' depending on logged in status
	newSession.Values["state"] = randToken
	newSession.Save(r, w)
	loginURL := h.conf.AuthCodeURL(randToken)
	t := template.Must(template.New("landing-page").ParseFS(*h.templates, "templates/home.html"))
	err = t.ExecuteTemplate(w, "home.html", struct {
		GoogleLoginURL string
	}{GoogleLoginURL: loginURL})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}

func (h *Home) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	session, err := h.sess.Get(r, "carp")
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
	googleData, err := parseGoogleData(email.Body)
	if err != nil {
		http.Error(w, "Authorization Unsuccessful", http.StatusBadRequest)
		return
	}
	// Confirm student is use `student.dodea.edu` account
	if googleData["hd"] != "student.dodea.edu" {
		http.Redirect(w, r, "/wrong_account", http.StatusFound)
		return
	}
	// Check if User Already Exists
	mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*5)
	defer mongoCancel()
	var userId primitive.ObjectID
	res := h.db.Collection("users").FindOne(mongoContext, bson.M{"email": googleData["email"]})
	if res.Err() == mongo.ErrNoDocuments {
		newUser := models.User{
			Email:      googleData["email"].(string),
			IsAdmin:    false,
			SurveyType: imageGroup(),
			Data:       primitive.M{},
			CreatedOn:  time.Now(),
			UpdatedOn:  time.Now(),
		}
		res, err := h.db.Collection("users").InsertOne(mongoContext, newUser)
		if err != nil {
			h.l.Error("Could not inset new user into database", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
		userId = res.InsertedID.(primitive.ObjectID)
	} else if res.Err() != nil {
		if err != nil {
			h.l.Error("Could not search for user in database", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
	} else {
		user := models.User{}
		err = res.Decode(&user)
		if err != nil {
			h.l.Error("Unable to convert user from database record to object", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
		userId = user.ID
	}
	// Assign a session token
	session.Values["_id"] = userId.Hex()
	err = session.Save(r, w)
	if err != nil {
		h.l.Error("Unable to assign session token to user", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/survey/start", http.StatusFound)
}

func imageGroup() int {
	return rand.Intn(2)
}

func parseGoogleData(res io.ReadCloser) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(res)
	if err != nil {
		return nil, err
	}
	output := make(map[string]interface{})
	if err = json.Unmarshal(data, &output); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return output, nil
}

func randStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
