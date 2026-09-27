package database

// GroupNameUpdatedResult matches the GroupNameUpdated schema in the OpenAPI spec.
type GroupNameUpdatedResult struct {
	GroupID int64  `json:"groupId"`
	Name    string `json:"name"`
}

// SetGroupName updates the group name in the database.
func (db *appdbimpl) SetGroupName(groupID int64, name string) (GroupNameUpdatedResult, error) {
	var result GroupNameUpdatedResult

	_, err := db.c.Exec(`
		UPDATE GroupChat SET chatName = ? WHERE chat = ?
	`, name, groupID)
	if err != nil {
		return result, err
	}

	result.GroupID = groupID
	result.Name = name

	return result, nil
}
