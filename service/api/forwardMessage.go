package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 2. Parse path parameters
	sourceConversationID, err := strconv.ParseInt(ps.ByName("conversationId"), 10, 64)
	if err != nil || sourceConversationID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	messageID, err := strconv.ParseInt(ps.ByName("messageId"), 10, 64)
	if err != nil || messageID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Decode the JSON request body
	var req struct {
		TargetConversationID int64 `json:"targetConversationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetConversationID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Check that the source conversation exists
	sourceExists, err := rt.db.ConversationExists(sourceConversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !sourceExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 5. Check that the user is a participant in the source conversation
	isMemberSource, err := rt.db.IsParticipant(sourceConversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMemberSource {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 6. Check that the message exists in the source conversation
	msgExists, err := rt.db.MessageExists(messageID, sourceConversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !msgExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 7. Check that the target conversation exists
	targetExists, err := rt.db.ConversationExists(req.TargetConversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !targetExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 8. Check that the user is a participant in the target conversation
	isMemberTarget, err := rt.db.IsParticipant(req.TargetConversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMemberTarget {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 9. Forward the message in the database
	result, err := rt.db.ForwardMessage(messageID, req.TargetConversationID, currentUser.Username)
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
