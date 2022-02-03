package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExtractUserID takes request and response objects along with gorilla session object to extract mongo ID from cookies.
// Errors if unable to extract ObjectID for whatever reason.
func ExtractUserID(
	w http.ResponseWriter,
	r *http.Request,
	sess *sessions.CookieStore,
) (*primitive.ObjectID, error) {
	session, err := sess.Get(r, "carp")
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%s", session.Values["_id"])
	if id == "" {
		return nil, errors.New("invalid session cookie")
	}
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return &objId, nil
}
