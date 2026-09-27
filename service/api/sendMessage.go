package api

import (
	"encoding/json"
	"fmt"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 4. Check that the user is a participant in the conversation
	isMember, err := rt.db.IsParticipant(conversationID, currentUser.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 5. Distinguish between text and image message based on Content-Type
	contentType := r.Header.Get("Content-Type")

	var result database.MessageCreatedResult

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// --- IMAGE MESSAGE ---
		result, err = rt.handleImageMessage(r, conversationID, currentUser.Username)
	} else {
		// --- TEXT MESSAGE ---
		result, err = rt.handleTextMessage(r, conversationID, currentUser.Username)
	}

	if err != nil {
		if err.Error() == "bad request" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Respond with 201 Created
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}

// handleTextMessage handles sending a text message
func (rt *_router) handleTextMessage(r *http.Request, conversationID int64, sender string) (database.MessageCreatedResult, error) {
	var req struct {
		Content          string `json:"content"`
		ReplyToMessageID *int64 `json:"replyToMessageId"` // pointer: field is optional
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return database.MessageCreatedResult{}, fmt.Errorf("bad request")
	}
	if len(req.Content) == 0 {
		return database.MessageCreatedResult{}, fmt.Errorf("bad request")
	}

	return rt.db.InsertTextMessage(conversationID, sender, req.Content, req.ReplyToMessageID)
}

// handleImageMessage handles sending an image message
func (rt *_router) handleImageMessage(r *http.Request, conversationID int64, sender string) (database.MessageCreatedResult, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return database.MessageCreatedResult{}, fmt.Errorf("bad request")
	}

	file, fileHeader, err := r.FormFile("photo")
	if err != nil {
		return database.MessageCreatedResult{}, fmt.Errorf("bad request")
	}
	defer file.Close()

	// Save the image file to disk
	if err := os.MkdirAll("/tmp/photos", os.ModePerm); err != nil {
		return database.MessageCreatedResult{}, err
	}
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	filename := fmt.Sprintf("msg-%d-%s%s", conversationID, sender, ext)
	savePath := filepath.Join("/tmp/photos", filename)

	dst, err := os.Create(savePath)
	if err != nil {
		return database.MessageCreatedResult{}, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return database.MessageCreatedResult{}, err
	}

	imageURL := fmt.Sprintf("/photos/%s", filename)

	return rt.db.InsertImageMessage(conversationID, sender, imageURL)
}
