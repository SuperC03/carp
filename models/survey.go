package models

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Survey represents the single response a user can give
type Survey struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	UserId     primitive.ObjectID  `bson:"user_id"`
	SurveyType int                 `bson:"survey_type"`
	Data       bson.A              `bson:"survey_type"`
	CreatedOn  primitive.Timestamp `bson:"created_on"`
	UpdatedOn  primitive.Timestamp `bson:"updated_on"`
}

const (
	SurveyNoImage   = 0
	SurveyWithImage = 1
)
