package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	conversationID, err := strconv.ParseInt(ps.ByName("conversationId"), 10, 64)
	if err != nil || conversationID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	messageID, err := strconv.ParseInt(ps.ByName("messageId"), 10, 64)
	if err != nil || messageID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Check that the conversation exists
	exists, err := rt.db.ConversationExists(conversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 4. Check that the user is a participant
	isMember, err := rt.db.IsParticipant(conversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 5. Check that the message exists and belongs to the conversation
	msgExists, err := rt.db.MessageExists(messageID, conversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !msgExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 6. Decode the JSON request body (the emoticon)
	var req struct {
		Emoticon string `json:"emoticon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(req.Emoticon) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 7. Add or update the reaction in the database
	//    (if the user already reacted to this message, the reaction is replaced)
	result, err := rt.db.UpsertReaction(messageID, currentUser.Username, req.Emoticon)
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
