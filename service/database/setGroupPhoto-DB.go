package database

// GroupPhotoUpdatedResult matches the GroupPhotoUpdated schema in the OpenAPI spec.
type GroupPhotoUpdatedResult struct {
	GroupID  int64  `json:"groupId"`
	PhotoURL string `json:"photoUrl"`
}

// SetGroupPhoto updates the group photo URL in the database.
func (db *appdbimpl) SetGroupPhoto(groupID int64, photoURL string) (GroupPhotoUpdatedResult, error) {
	var result GroupPhotoUpdatedResult

	_, err := db.c.Exec(`
		UPDATE GroupChat SET photo_url = ? WHERE chat = ?
	`, photoURL, groupID)
	if err != nil {
		return result, err
	}

	result.GroupID = groupID
	result.PhotoURL = photoURL

	return result, nil
}
