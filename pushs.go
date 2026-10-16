package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

var pushHostsExact = map[string]bool{
	"fcm.googleapis.com": true,
	"web.push.apple.com": true,
}

var pushHostSuffixes = []string{
	".push.services.mozilla.com",
	".notify.windows.com",
}

// ValidPushEndpoint: only https, port 443, no userinfo, host from whitelist.
// We check Hostname(), not Host: otherwise it can be bypassed via "host:port" and "user@host".
func ValidPushEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	if p := u.Port(); p != "" && p != "443" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if pushHostsExact[host] {
		return true
	}
	for _, s := range pushHostSuffixes {
		if strings.HasSuffix(host, s) { // dot at the start of the suffix is mandatory
			return true
		}
	}
	return false
}

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

// Separate client: timeout and redirect ban (3xx will go into "-5").
var pushHTTPClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: GlobalHTTPClient.Transport,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// SendPushToUser sends a push to ALL of the user's subscriptions (all devices/browsers),
// and not only to the first found one, as was in the old version. Collects errors
// for each subscription separately, so that one dead subscription doesn't block
// the rest; dead ones (410/404) are cleaned from the DB immediately.
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

// Limit of subscriptions per user.
func CountSubscriptions(userId int, endpoint string) (int, error) {
	var n int
	err := DB.Get(&n, `SELECT COUNT(*) FROM pushs WHERE userId = ? AND endpoint <> ?`, userId, endpoint)
	return n, err
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

// Return codes are one-to-one with the Rust version (see push.rs):
// "1"    — success
// "410"  — EndpointNotValid / EndpointNotFound (subscription is dead, needs cleanup in DB)
// "-1"   — VAPID_PRIVATE_KEY missing
// "-2"   — (not used separately in the Go version, see below)
// "-3"   — message assembly error
// "-4"   — client initialization error (not applicable to webpush-go, kept for compatibility)
// "-5"   — other send error
// "-6"   — VAPID key conversion/decoding error

func SendWebPush(req SendWebPushRequest) string {
	if !ValidPushEndpoint(req.Endpoint) {
		return "410" // foreign/garbage endpoint: clean from DB as dead
	}
	vapidRaw := os.Getenv("VAPID_PRIVATE_KEY")
	if vapidRaw == "" {
		fmt.Fprintln(os.Stderr, "VAPID_PRIVATE_KEY missing")
		return "-1"
	}
	vapidPublic := os.Getenv("VAPID_PUBLIC_KEY")
	if vapidPublic == "" {
		// webpush-go requires both keys to assemble the VAPID Authorization header.
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
		HTTPClient:      pushHTTPClient,
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
