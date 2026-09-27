package api

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// 2. Parse the multipart body (10MB limit as per spec)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		if err.Error() == "http: request body too large" {
			w.WriteHeader(http.StatusRequestEntityTooLarge) // 413
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Extract the file from the "photo" field (field name defined in the spec)
	file, fileHeader, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Create the "photos/" directory if it does not exist
	if err := os.MkdirAll("/tmp/photos", os.ModePerm); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Save the file to disk using the username as the filename
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	filename := fmt.Sprintf("%s%s", currentUser.Username, ext)
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

	// 6. Build the public URL for the photo
	photoURL := fmt.Sprintf("/photos/%s", filename)

	// 7. Update the database with the new photo URL
	if err := rt.db.SetUserPhoto(currentUser.Username, photoURL); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 8. Update the in-memory user map
	u := Users[token]
	u.PhotoURL = photoURL
	Users[token] = u

	// 9. Respond with the new photo URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"photoUrl": photoURL,
	}); err != nil {
		_ = err
	}
}
