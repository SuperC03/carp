package handlers

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/superc03/carp/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Survey struct {
	l         *zap.Logger
	db        *mongo.Database
	sess      *sessions.CookieStore
	templates *embed.FS
}

func NewSurvey(
	l *zap.Logger,
	db *mongo.Database,
	sess *sessions.CookieStore,
	templates *embed.FS,
) *Survey {
	return &Survey{
		l, db, sess, templates,
	}
}

func (s *Survey) StartPage(w http.ResponseWriter, r *http.Request) {
	_, err := utils.ExtractUserID(w, r, s.sess)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	t := template.Must(template.New("survey-start-page").ParseFS(*s.templates, "templates/start.html"))
	err = t.ExecuteTemplate(w, "start.html", struct {
		NextQuestionPath string
	}{NextQuestionPath: "/"})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}
