package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"regexp"
)

// User represents a logged-in user stored in memory at runtime.
type User struct {
	Username string `json:"username"`
	PhotoURL string `json:"photoUrl"`
	Token    string `json:"-"`
}

// var Users = []User{}
// Users is the in-memory store of authenticated users (token → User).
// A map is used so that token lookup is O(1).
var Users = map[string]User{}

func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Decode the JSON request body
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. Validate the username format against the pattern defined in the OpenAPI spec
	pattern := "^[A-Za-z0-9]{3,16}$"
	matched, err := regexp.MatchString(pattern, req.Name)
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Look up the user in the database
	existingUsername, err := rt.db.DoLogin(req.Name)

	if errors.Is(err, sql.ErrNoRows) {
		// User does not exist yet —> create it
		if err2 := rt.db.CreateUser(req.Name); err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		// Unexpected database error
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		// User already exists — use the canonical name stored in the DB
		req.Name = existingUsername
	}

	// 4. Retrieve the user photo URL (optional — may be empty)
	photoURL := ""
	if pic, err := rt.db.GetUserPhoto(req.Name); err == nil {
		photoURL = pic
	}

	// 5. Generate the Bearer token and store the user in the in-memory map
	token := generateToken(req.Name)
	Users[token] = User{
		Username: req.Name,
		PhotoURL: photoURL,
		Token:    token,
	}

	// 6. Respond with the token as the user identifier
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"identifier": token,
	}); err != nil {
		_ = err
	}
}

// generateToken creates a deterministic Bearer token from the username.
func generateToken(username string) string {
	return fmt.Sprintf("user-%s", username)
}
