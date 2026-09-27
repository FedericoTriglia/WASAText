package database

func (db *appdbimpl) DoLogin(username string) (string, error) {
	var name string
	err := db.c.QueryRow(`SELECT username FROM User WHERE username = ?`, username).Scan(&name)
	return name, err
}

func (db *appdbimpl) GetUserPhoto(username string) (string, error) {
	var out string
	err := db.c.QueryRow(`SELECT photo_url FROM User WHERE username = ?`, username).Scan(&out)
	return out, err
}

// CreateUser inserts a new user into the database with an empty photo URL.
func (db *appdbimpl) CreateUser(username string) error {
	_, err := db.c.Exec(
		`INSERT INTO User (username, photo_url) VALUES (?, ?)`,
		username, "",
	)
	return err
}
