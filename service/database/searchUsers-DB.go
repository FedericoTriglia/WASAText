package database

import (
	"database/sql"
)

// UserResult matches a single element of the searchUsers response array
// defined in the OpenAPI spec.
type UserResult struct {
	Username string `json:"username"`
	PhotoURL string `json:"photoUrl,omitempty"`
}

// SearchUsers returns all users whose username contains the query string
// (partial match using SQL LIKE).
func (db *appdbimpl) SearchUsers(query string) ([]UserResult, error) {
	rows, err := db.c.Query(`
		SELECT username, photo_url
		FROM User
		WHERE username LIKE ?
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserResult
	for rows.Next() {
		var u UserResult
		var photoURL sql.NullString
		if err := rows.Scan(&u.Username, &photoURL); err != nil {
			return nil, err
		}
		if photoURL.Valid {
			u.PhotoURL = photoURL.String
		}
		users = append(users, u)
	}

	return users, rows.Err()
}
