package handlers

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/superc03/carp/models"
	"github.com/superc03/carp/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

type userFromContext struct{}

func (s *Survey) UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId, err := utils.ExtractUserID(r, s.sess)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*5)
		defer mongoCancel()
		res := s.db.Collection("users").FindOne(mongoContext, primitive.M{"_id": userId})
		if res.Err() != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		user := models.User{}
		if err := res.Decode(&user); err != nil {
			s.l.Error("Unable to decode user record into struct", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), userFromContext{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Survey) StartPage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userFromContext{}).(models.User)
	mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*5)
	defer mongoCancel()

	nextQuestionPath, err := user.NextArticlePath(mongoContext, s.db)
	if err != nil {
		s.l.Error("Unable to determine user's next article path", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	t := template.Must(template.New("survey-start-page").ParseFS(*s.templates, "templates/start.html"))
	err = t.ExecuteTemplate(w, "start.html", struct {
		NextQuestionPath string
	}{NextQuestionPath: nextQuestionPath})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}

func (s *Survey) QuestionPage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userFromContext{}).(models.User)
	mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*5)
	defer mongoCancel()

	nextQuestionPath, err := user.NextArticlePath(mongoContext, s.db)
	if err != nil {
		s.l.Error("Unable to determine user's next article path", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	t := template.Must(template.New("survey-question-page").ParseFS(*s.templates, "templates/question.html"))
	err = t.ExecuteTemplate(w, "question.html", struct {
		LikenScaleValues   []int
		UserImageShown     bool
		ArticleHeadline    string
		ArticlePictureCode string
		NextPath           string
		ArticleID          string
	}{
		LikenScaleValues: []int{1, 2, 3, 4, 5},
		NextPath:         nextQuestionPath,
	})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}
