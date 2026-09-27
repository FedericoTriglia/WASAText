package api

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Authenticate the request via Bearer token
	token, ok := ExtractBearer(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	currentUser, ok := GetUserByToken(token)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 2. Decode the JSON request body
	var req struct {
		Name            string   `json:"name"`
		MemberUsernames []string `json:"memberUsernames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validate: the group name must not be empty and at least one member is required
	if len(req.Name) < 1 || len(req.MemberUsernames) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Build the full member list:
	//    the authenticated user is automatically included if not already present
	allMembers := req.MemberUsernames
	alreadyIn := false
	for _, m := range allMembers {
		if m == currentUser.Username {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		allMembers = append(allMembers, currentUser.Username)
	}

	// 5. Check that every member exists in the database
	for _, member := range allMembers {
		exists, err := rt.db.UserExists(member)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !exists {
			w.WriteHeader(http.StatusNotFound) // 404 if a member does not exist
			return
		}
	}

	// 6. Create the group in the database
	result, err := rt.db.CreateGroup(req.Name, allMembers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 7. Respond with 201 Created
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}
