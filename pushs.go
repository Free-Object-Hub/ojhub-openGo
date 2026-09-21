package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
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

// SendPushToUser отправляет пуш на ВСЕ подписки юзера (все устройства/браузеры),
// а не только на первую найденную, как было в старой версии. Собирает ошибки
// по каждой подписке отдельно, чтобы одна мёртвая подписка не блокировала
// остальные; мёртвые (410/404) чистит из БД сразу.
func SendPushToUser(userId int, title, body string) error {
	subs, err := GetSubscriptionsByUser(userId)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return fmt.Errorf("no push subscriptions for user %d", userId)
	}
	var lastErr error
	for _, s := range subs {
		code := SendWebPush(SendWebPushRequest{
			Endpoint: s.Endpoint,
			P256dh:   s.P256dh,
			Auth:     s.Auth,
			Title:    title,
			Body:     body,
		})
		switch code {
		case "1":
			// ok
		case "410":
			if delErr := RemoveSubscription(s.Endpoint); delErr != nil {
				lastErr = fmt.Errorf("dead subscription cleanup failed for endpoint %q: %w", s.Endpoint, delErr)
			}
		default:
			lastErr = fmt.Errorf("push failed for endpoint %q: code %s", s.Endpoint, code)
		}
	}
	return lastErr
}

type SendWebPushRequest struct {
	Endpoint string
	P256dh   string
	Auth     string
	Title    string
	Body     string
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Коды возврата один в один с Rust-версией (см. push.rs):
// "1"    — успех
// "410"  — EndpointNotValid / EndpointNotFound (подписка мертва, надо чистить в БД)
// "-1"   — VAPID_PRIVATE_KEY отсутствует
// "-2"   — (не используется отдельно в Go-версии, см. ниже)
// "-3"   — ошибка сборки сообщения
// "-4"   — ошибка инициализации клиента (не применимо к webpush-go, оставлено для совместимости)
// "-5"   — прочая ошибка отправки
// "-6"   — ошибка конвертации/декодирования VAPID-ключа

func SendWebPush(req SendWebPushRequest) string {
	vapidRaw := os.Getenv("VAPID_PRIVATE_KEY")
	if vapidRaw == "" {
		fmt.Fprintln(os.Stderr, "VAPID_PRIVATE_KEY missing")
		return "-1"
	}
	vapidPublic := os.Getenv("VAPID_PUBLIC_KEY")
	if vapidPublic == "" {
		// webpush-go требует оба ключа для сборки VAPID Authorization header.
		fmt.Fprintln(os.Stderr, "VAPID_PUBLIC_KEY missing")
		return "-6"
	}
	sub := &webpush.Subscription{
		Endpoint: req.Endpoint,
		Keys: webpush.Keys{
			P256dh: req.P256dh,
			Auth:   req.Auth,
		},
	}
	payloadBytes, err := json.Marshal(pushPayload{Title: req.Title, Body: req.Body})
	if err != nil {
		fmt.Fprintln(os.Stderr, "payload marshal error:", err)
		return "-3"
	}
	resp, err := webpush.SendNotification(payloadBytes, sub, &webpush.Options{
		Subscriber:      "mailto:admin@objecthub.xyz",
		VAPIDPublicKey:  vapidPublic,
		VAPIDPrivateKey: vapidRaw,
		TTL:             60,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "send error:", err)
		return "-5"
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return "410"
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "1"
	}
	fmt.Fprintln(os.Stderr, "unexpected push status:", resp.StatusCode)
	return "-5"
}
