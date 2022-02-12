package models

import (
	"context"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// User represents a survey participant who has signed-in with their Google account
type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Email      string             `bson:"email"`
	IsAdmin    bool               `bson:"is_admin,"`
	SurveyType int                `bson:"survey_type,"`
	Data       bson.M             `bson:"survey_data"`
	CreatedOn  time.Time          `bson:"created_on,omitempty"`
	UpdatedOn  time.Time          `bson:"updated_on,omitempty"`
}

func (u *User) NextArticlePath(ctx context.Context, db *mongo.Database, notIncluding *primitive.ObjectID) (string, error) {
	articleIDs, err := u.RemainingArticles(ctx, db, notIncluding)
	if err != nil {
		return "", err
	}
	if len(articleIDs) == 0 {
		return "complete", nil
	}
	return articleIDs[rand.Intn(len(articleIDs))].Hex(), nil
}

func (u *User) RemainingArticles(ctx context.Context, db *mongo.Database, notIncluding *primitive.ObjectID) ([]primitive.ObjectID, error) {
	var (
		completedIDs        = make([]primitive.ObjectID, 0)
		remainingArticleIDs = make([]primitive.ObjectID, 0)
	)
	for k := range u.Data {
		id, err := primitive.ObjectIDFromHex(k)
		if err != nil {
			return nil, err
		}
		completedIDs = append(completedIDs, id)

	}
	if notIncluding != nil {
		completedIDs = append(completedIDs, *notIncluding)
	}
	cur, err := db.Collection("articles").Find(ctx, bson.M{"_id": bson.M{"$nin": completedIDs}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		article := Article{}
		if err := cur.Decode(&article); err != nil {
			return nil, err
		}
		remainingArticleIDs = append(remainingArticleIDs, article.ID)
	}
	return remainingArticleIDs, nil
}

const (
	SurveyNoImage   = 0
	SurveyWithImage = 1
)
