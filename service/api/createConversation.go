package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 2. Decode the JSON request body  (targetUsername and content)
	var req struct {
		TargetUsername string `json:"targetUsername"`
		Content        string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validate the target username format
	matched, err := regexp.MatchString(`^[A-Za-z0-9]{3,16}$`, req.TargetUsername)
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Validate the first message content
	if len(req.Content) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 5. A user cannot start a conversation with themselves
	if req.TargetUsername == currentUser.Username {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 6. Check that the target user exists
	targetExists, err := rt.db.UserExists(req.TargetUsername)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !targetExists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("User '%s' not found", req.TargetUsername),
		}); err != nil {
			_ = err
		}
		return
	}

	// 7. Check that no private conversation between these two users already exists
	alreadyExists, err := rt.db.PrivateConversationExists(currentUser.Username, req.TargetUsername)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if alreadyExists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("A conversation with '%s' already exists", req.TargetUsername),
		}); err != nil {
			_ = err
		}
		return
	}

	// 8. Create the conversation and the first message atomically
	result, err := rt.db.CreateConversation(currentUser.Username, req.TargetUsername, req.Content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 9. Respond with 201 Created
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}
