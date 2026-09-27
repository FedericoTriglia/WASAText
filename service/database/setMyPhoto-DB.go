package database

// SetUserPhoto updates the photo URL of a user in the database.
func (db *appdbimpl) SetUserPhoto(username string, photoURL string) error {
	_, err := db.c.Exec(
		`UPDATE User SET photo_url = ? WHERE username = ?`,
		photoURL, username,
	)
	return err
}
