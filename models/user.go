package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

const (
	SurveyNoImage   = 0
	SurveyWithImage = 1
)
