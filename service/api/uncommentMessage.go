package api

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 5. Check that the message exists in the conversation
	msgExists, err := rt.db.MessageExists(messageID, conversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !msgExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 6. Remove the user's reaction from the message.
	//    If the reaction does not exist, DeleteReaction returns an error
	//    which is mapped to 404.
	err = rt.db.DeleteReaction(messageID, currentUser.Username)
	if err != nil {
		if err.Error() == "reaction not found" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 7. Respond with 204 No Content (no body)
	w.WriteHeader(http.StatusNoContent)
}
