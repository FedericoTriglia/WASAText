package api

import (
	"encoding/json"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"regexp"
)

func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Authenticate the request via Bearer token
	token, ok := ExtractBearer(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, ok = GetUserByToken(token)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 2. Read the "randomUser" query parameter from the URL
	//    e.g. GET /users?randomUser=Mari
	query := r.URL.Query().Get("randomUser")

	// 3. Validate the parameter format (as per spec: 3-100 alphanumeric characters)
	matched, err := regexp.MatchString(`^[A-Za-z0-9]{3,100}$`, query)
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Search for matching users in the database
	users, err := rt.db.SearchUsers(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Ensure the response is always a JSON array (never null)
	if users == nil {
		users = []database.UserResult{}
	}

	// 6. Respond with 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(users); err != nil {
		_ = err
	}
}
