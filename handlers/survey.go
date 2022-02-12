package handlers

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/superc03/carp/models"
	"github.com/superc03/carp/utils"
	"go.mongodb.org/mongo-driver/bson"
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

	nextQuestionPath, err := user.NextArticlePath(mongoContext, s.db, nil)
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

	articleCode := mux.Vars(r)["code"]

	if r.Method == http.MethodPost {
		scoredArticleCode := r.FormValue("articleID")
		scoredArticleRating := r.FormValue("score")
		scoredArticleNumericRating, err := strconv.Atoi(scoredArticleRating)
		if err != nil {
			s.l.Error("Unable to decode article's score", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusBadRequest)
			return
		}
		scoredArticleId, err := primitive.ObjectIDFromHex(scoredArticleCode)
		if err != nil {
			s.l.Error("Unable to decode article's id code", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusBadRequest)
			return
		}
		err = utils.SubmitRating(mongoContext, s.db, user.ID, scoredArticleId, scoredArticleNumericRating)
		if err != nil {
			s.l.Error("Unable to submit user rating", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
	}

	// Find and confirm article's existance
	articleId, err := primitive.ObjectIDFromHex(articleCode)
	if err != nil {
		s.l.Error("Unable to decode article's id code", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusBadRequest)
		return
	}
	res := s.db.Collection("articles").FindOne(mongoContext, bson.M{"_id": articleId})
	if res.Err() == mongo.ErrNoDocuments {
		http.Redirect(w, r, "/", http.StatusNotFound)
		return
	} else if res.Err() != nil {
		s.l.Error("Unable to locate article", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	article := models.Article{}
	res.Decode(&article)

	nextQuestionPath, err := user.NextArticlePath(mongoContext, s.db, &articleId)
	if err != nil {
		s.l.Error("Unable to determine user's next article path", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}

	// Check if user already answered question
	if user.Data[articleCode] == 0 || user.Data[articleCode] == "" {
		http.Redirect(w, r, "/survey/"+nextQuestionPath, http.StatusFound)
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
		UserImageShown:     user.SurveyType == models.SurveyWithImage,
		ArticleHeadline:    article.Title,
		ArticlePictureCode: article.PictureCode,
		ArticleID:          articleCode,
		LikenScaleValues:   []int{1, 2, 3, 4, 5},
		NextPath:           nextQuestionPath,
	})
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}

func (s *Survey) CompletePage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userFromContext{}).(models.User)
	mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*5)
	defer mongoCancel()
	if r.Method == http.MethodPost {
		scoredArticleCode := r.FormValue("articleID")
		scoredArticleRating := r.FormValue("score")
		scoredArticleNumericRating, err := strconv.Atoi(scoredArticleRating)
		if err != nil {
			s.l.Error("Unable to decode article's score", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusBadRequest)
			return
		}
		scoredArticleId, err := primitive.ObjectIDFromHex(scoredArticleCode)
		if err != nil {
			s.l.Error("Unable to decode article's id code", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusBadRequest)
			return
		}
		err = utils.SubmitRating(mongoContext, s.db, user.ID, scoredArticleId, scoredArticleNumericRating)
		if err != nil {
			s.l.Error("Unable to submit user rating", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
	}
	// Delete the User Cookie upon completion
	session, err := s.sess.Get(r, "carp")
	session.Options.MaxAge = -1
	err = session.Save(r, w)
	if err != nil {
		s.l.Error("Unable to assign session token to user", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	t := template.Must(template.New("survey-complete-page").ParseFS(*s.templates, "templates/complete.html"))
	err = t.ExecuteTemplate(w, "complete.html", nil)
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}
