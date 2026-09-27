package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 2. Parse the :groupId path parameter
	groupID, err := strconv.ParseInt(ps.ByName("groupId"), 10, 64)
	if err != nil || groupID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Check that the group exists
	exists, err := rt.db.GroupExists(groupID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Group not found",
		}); err != nil {
			_ = err
		}
		return
	}

	// 4. Check that the authenticated user is a member of the group (403 otherwise)
	isMember, err := rt.db.IsParticipant(groupID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Only group members can add other users",
		}); err != nil {
			_ = err
		}
		return
	}

	// 5. Decode the JSON request body  {"username": "Luigi"}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		}); err != nil {
			_ = err
		}
		return
	}

	// 6. Validate the username format
	matched, err := regexp.MatchString(`^[A-Za-z0-9]{3,16}$`, req.Username)
	if err != nil || !matched {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid username format",
		}); err != nil {
			_ = err
		}
		return
	}

	// 7. Check that the target user exists in the database
	userExists, err := rt.db.UserExists(req.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !userExists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("User '%s' not found", req.Username),
		}); err != nil {
			_ = err
		}
		return
	}

	// 8. Check that the user is not already a member of the group (409 otherwise)
	alreadyMember, err := rt.db.IsParticipant(groupID, req.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if alreadyMember {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("User '%s' is already a member of this group", req.Username),
		}); err != nil {
			_ = err
		}
		return
	}

	// 9. Add the user to the group in the database
	result, err := rt.db.AddUserToGroup(groupID, req.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 10. Respond with 201 Created
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}
