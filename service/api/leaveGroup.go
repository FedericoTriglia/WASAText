package api

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 4. Check that the user is a member of the group
	isMember, err := rt.db.IsParticipant(groupID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 5. Leave the group (and delete it if the user was the last member)
	if err := rt.db.LeaveGroup(groupID, currentUser.Username); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Respond with 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
