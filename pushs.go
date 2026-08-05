package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PushSubscription struct {
	ID        int    `db:"ID"`
	UserId    int    `db:"userId"`
	DeviceId  int    `db:"deviceId"`
	Endpoint  string `db:"endpoint"`
	P256dh    string `db:"p256dh"`
	Auth      string `db:"auth"`
	UserAgent string `db:"userAgent"`
	CreatedAt int64  `db:"createdAt"`
}

func SaveSubscription(userId, deviceId int, endpoint, p256dh, auth, userAgent string) error {
	_, err := DB.Exec(
		`INSERT INTO pushs (userId, deviceId, endpoint, p256dh, auth, userAgent, createdAt) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		userId = VALUES(userId),
		deviceId = VALUES(deviceId),
		p256dh = VALUES(p256dh),
		auth = VALUES(auth),
		userAgent = VALUES(userAgent)`,
		userId, deviceId, endpoint, p256dh, auth, userAgent, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}
	return nil
}

func RemoveSubscription(endpoint string) error {
	_, err := DB.Exec(`DELETE FROM pushs WHERE endpoint = ?`, endpoint)
	if err != nil {
		return fmt.Errorf("failed to remove subscription: %w", err)
	}
	return nil
}

func GetSubscriptionsByUser(userId int) ([]PushSubscription, error) {
	var subs []PushSubscription
	err := DB.Select(&subs, `SELECT * FROM pushs WHERE userId = ?`, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	return subs, nil
}

func GetDeviceIdByToken(deviceToken string, userId int) (int, error) {
	var id int
	err := DB.Get(&id, `SELECT ID FROM devices WHERE staticFp = ? AND userId = ?`, deviceToken, userId)
	if err != nil {
		return 0, fmt.Errorf("device not found: %w", err)
	}
	return id, nil
}

func SendPushToUser(userId int, title, body string) error {
	var endpoint, p256dh, auth string
	err := DB.Get(&endpoint, "SELECT endpoint FROM pushs WHERE userId = ?", userId)
	if err != nil {
		return fmt.Errorf("user push endpoint not found: %w", err)
	}

	err = DB.Get(&p256dh, "SELECT p256dh FROM pushs WHERE userId = ?", userId)
	if err != nil {
		return fmt.Errorf("user push p256dh not found: %w", err)
	}

	err = DB.Get(&auth, "SELECT auth FROM pushs WHERE userId = ?", userId)
	if err != nil {
		return fmt.Errorf("user push auth not found: %w", err)
	}

	payload := map[string]interface{}{
		"endpoint": endpoint,
		"p256dh":   p256dh,
		"auth":     auth,
		"title":    title,
		"body":     body,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post("http://localhost:8087/cli/send-push", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("push failed: %d", resp.StatusCode)
	}
	return nil
}
