package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
		return
	}

	// 4. Check that the user is a member of the group (otherwise return 403)
	isMember, err := rt.db.IsParticipant(groupID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "You must be a group member to change the name",
		}); err != nil {
			_ = err
		}
		return
	}

	// 5. Decode the JSON request body
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 6. Validate the name format (as per spec: 3-50 characters)
	matched, err := regexp.MatchString(`^[A-Za-z0-9 àèìòù]{3,50}$`, req.Name)
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 7. Update the group name in the database
	result, err := rt.db.SetGroupName(groupID, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 8. Respond with 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}
