package main

import (
	"fmt"
	"log"
	"strings"
)

const (
	ChannelWiki    = -1
	ChannelJoinLog = -4
)

// #region gdps owners
type GdpsOwner struct {
	ID       int    `db:"ID"`
	Username string `db:"username"`
	Channel  int    `db:"channel"`
	GdpsId   int    `db:"gdpsId"`
	UserId   int    `db:"userId"`
}

func FetchOwnedGdps(gdpsId int, channel int) ([]GdpsOwner, error) {
	var owners []GdpsOwner
	query := "SELECT * FROM soowners WHERE gdpsId = ? AND channel = ? ORDER BY ID DESC"
	err := DB.Select(&owners, query, gdpsId, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gdps owners: %w", err)
	}
	return owners, nil
}

func AddOwnerGdps(gdpsId, userId, channel int) error {
	_, err := DB.Exec(
		"INSERT INTO soowners (gdpsId, userId, channel) VALUES (?, ?, ?)",
		gdpsId, userId, channel,
	)
	if err != nil {
		return fmt.Errorf("failed to add gdps owner: %w", err)
	}
	return nil
}

func DeleteOwnerGdps(gdpsId, userId, channel int) error {
	_, err := DB.Exec(
		"DELETE FROM soowners WHERE gdpsId = ? AND userId = ? AND channel = ?",
		gdpsId, userId, channel,
	)
	if err != nil {
		return fmt.Errorf("failed to delete gdps owner: %w", err)
	}
	return nil
}

// #endregion
// #region wiki owners
type WikiOwner struct {
	ID     int `db:"ID"`
	WikiId int `db:"wikiId"`
	UserId int `db:"userId"`
	// неиспользуемые поля
	PermLevel int `db:"permLevel"`
}

func FetchOwnedWiki(wikiId int) ([]WikiOwner, error) {
	var owners []WikiOwner
	query := "SELECT * FROM wikisoowners WHERE wikiId = ? ORDER BY ID DESC"
	err := DB.Select(&owners, query, wikiId)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to fetch wiki owners: %w", err)
	}
	return owners, nil
}

func AddOwnerWiki(wikiId, userId int) error {
	_, err := DB.Exec(
		"INSERT INTO wikisoowners (wikiId, userId) VALUES (?, ?)",
		wikiId, userId,
	)
	if err != nil {
		return fmt.Errorf("failed to add wiki owner: %w", err)
	}
	return nil
}

func DeleteOwnerWiki(wikiId, userId int) error {
	_, err := DB.Exec(
		"DELETE FROM wikisoowners WHERE wikiId = ? AND userId = ?",
		wikiId, userId,
	)
	if err != nil {
		return fmt.Errorf("failed to delete wiki owner: %w", err)
	}
	return nil
}

// #endregion
// #region joinlog (type -4, только чтение через fetchOwned)
type JoinLogEntry struct {
	ID       int    `db:"ID"`
	GdpsId   int    `db:"gdpsId"`
	UserId   int    `db:"userId"`
	Username string `db:"username"`
	JoinDate int    `db:"joinDate"`
	JoinData string `db:"joinData"`
}

func FetchJoinLog(gdpsId int) ([]JoinLogEntry, error) {
	var entries []JoinLogEntry
	query := "SELECT * FROM joinlog WHERE gdpsId = ? ORDER BY ID DESC"
	err := DB.Select(&entries, query, gdpsId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch join log: %w", err)
	}
	return entries, nil
}

func FetchUsersByIds(ids []int) (map[int]*User, error) {
	if len(ids) == 0 {
		return map[int]*User{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT * FROM users WHERE userId IN (%s)`, strings.Join(placeholders, ","))
	var users []User
	if err := DB.Select(&users, query, args...); err != nil {
		return nil, err
	}
	result := make(map[int]*User, len(users))
	for i := range users {
		result[users[i].UserId] = &users[i]
	}
	return result, nil
}

func nicknameOrFallback(u *User) string {
	if u == nil {
		return "???"
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	if u.Username != "" {
		return u.Username
	}
	return "???"
}

// #endregion
