package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
			"error": "You must be a group member to change the photo",
		}); err != nil {
			_ = err
		}
		return
	}

	// 5. Parse the multipart body (10MB limit)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		if err.Error() == "http: request body too large" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge) // 413
			if err := json.NewEncoder(w).Encode(map[string]string{
				"error": "File size exceeds maximum allowed (5MB)",
			}); err != nil {
				_ = err
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid image format. Supported: JPEG, PNG",
		}); err != nil {
			_ = err
		}
		return
	}

	// 6. Extract the file from the "photo" field
	file, _, err := r.FormFile("photo")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid image format. Supported: JPEG, PNG",
		}); err != nil {
			_ = err
		}
		return
	}
	defer file.Close()

	// 7. Save the file to disk
	if err := os.MkdirAll("/tmp/photos", os.ModePerm); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("group-%d.jpg", groupID)
	savePath := filepath.Join("/tmp/photos", filename)

	dst, err := os.Create(savePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	photoURL := fmt.Sprintf("/photos/%s", filename)

	// 8. Update the database with the new photo URL
	result, err := rt.db.SetGroupPhoto(groupID, photoURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 9. Respond with 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		_ = err
	}
}
