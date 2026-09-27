package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 2. Parse the :conversationId path parameter
	conversationID, err := strconv.ParseInt(ps.ByName("conversationId"), 10, 64)
	if err != nil || conversationID < 1 {
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

	// 4. Check that the user is a participant in the conversation (403 otherwise)
	isMember, err := rt.db.IsParticipant(conversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 5. Retrieve the full conversation
	conversation, err := rt.db.GetConversation(conversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Respond with 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(conversation); err != nil {
		_ = err
	}
}
