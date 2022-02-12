package utils

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func SubmitRating(ctx context.Context, db *mongo.Database, userID primitive.ObjectID, articleID primitive.ObjectID, score int) error {
	res, err := db.Collection("users").UpdateByID(ctx, userID, bson.D{
		{"$set", bson.D{{"survey_data." + articleID.Hex(), score}}},
	})
	if err != nil {
		return err
	}
	if res.ModifiedCount != 1 {
		return fmt.Errorf("Unable to locate user record to update score")
	}
	return nil
}
