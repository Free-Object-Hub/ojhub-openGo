package main

import "fmt"

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
}

func FetchOwnedWiki(wikiId int) ([]WikiOwner, error) {
	var owners []WikiOwner
	query := "SELECT * FROM wikisoowners WHERE wikiId = ? ORDER BY ID DESC"
	err := DB.Select(&owners, query, wikiId)
	if err != nil {
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
// FIXME: сделать нормальную структуру а не набросок клода
type JoinLogEntry struct {
	ID     int `db:"ID"`
	GdpsId int `db:"gdpsId"`
	UserId int `db:"userId"`
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

// #endregion
