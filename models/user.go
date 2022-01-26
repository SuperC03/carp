package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User represents a survey participant who has signed-in with their Google account
type User struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	Email      string              `bson:"title"`
	IsAdmin    bool                `bson:"is_admin,"`
	SurveyType int                 `bson:"survey_type,"`
	CreatedOn  primitive.Timestamp `bson:"created_on"`
	UpdatedOn  primitive.Timestamp `bson:"updated_on"`
}
