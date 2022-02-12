package handlers

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/superc03/carp/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Other struct {
	l         *zap.Logger
	templates *embed.FS
	db        *mongo.Database
}

func NewOther(
	l *zap.Logger,
	templates *embed.FS,
	db *mongo.Database,
) *Other {
	return &Other{
		l, templates, db,
	}
}

func (o *Other) StatisticsPage(w http.ResponseWriter, r *http.Request) {
	mongoContext, mongoCancel := context.WithTimeout(r.Context(), time.Second*15)
	defer mongoCancel()
	// Only Admin Access Allowed
	user := r.Context().Value(userFromContext{}).(models.User)
	if user.IsAdmin == false {
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	csvWriter := csv.NewWriter(w)
	// Accumulate all Article Codes
	cursor, err := o.db.Collection("articles").Find(mongoContext, bson.D{})
	if err != nil {
		o.l.Error("Unable to Accumulate all Article Codes", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	articleIDs := make([]string, 0, 10)
	for cursor.Next(mongoContext) {
		var article models.Article
		if err = cursor.Decode(&article); err != nil {
			o.l.Error("Unable to Accumulate all Article Codes", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
		articleIDs = append(articleIDs, article.ID.Hex())
	}
	err = csvWriter.Write(append([]string{"imagePresent"}, articleIDs...))
	if err != nil {
		o.l.Error("Unable to Accumulate all Article Codes", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	// Anomynously Accumulate all Article Scores
	cursor, err = o.db.Collection("users").Find(mongoContext, bson.M{"is_admin": bson.M{"$eq": false}})
	if err != nil {
		o.l.Error("Unable to Accumulate all Users", zap.Error(err))
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
	for cursor.Next(mongoContext) {
		var user models.User
		data := make([]string, 0, 11)
		if err = cursor.Decode(&user); err != nil {
			o.l.Error("Unable to Accumulate all Users", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
		if user.SurveyType == models.SurveyWithImage {
			data = append(data, "true")
		} else {
			data = append(data, "false")
		}
		for _, v := range articleIDs {
			data = append(data, fmt.Sprintf("%d", user.Data[v]))
		}
		err := csvWriter.Write(data)
		if err != nil {
			o.l.Error("Unable to Accumulate all Users", zap.Error(err))
			http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
			return
		}
	}

	csvWriter.Flush()
}

func (o *Other) WrongAccountPage(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("wrong-account-page").ParseFS(*o.templates, "templates/wrong_account.html"))
	err := t.ExecuteTemplate(w, "wrong_account.html", nil)
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}
