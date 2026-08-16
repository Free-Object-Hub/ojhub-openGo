package main

// МОДУЛЬ НАВАЙБКОЖЕН chatGpt!
// я его не верифал вообще

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var GlobalHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// TGWebhookLog отправляет технический лог в Telegram.
// Ошибка отправки лога не должна ломать основной запрос.
func TGWebhookLog(msg string) error {
	log.Println(msg)
	botToken := os.Getenv("TG_BOT_TOKEN")
	chatID := os.Getenv("TG_NEWS_RESENDER")

	if botToken == "" || chatID == "" {
		log.Printf("[TGWebhookLog] ENV MISSING: token_set=%v chat_set=%v", botToken != "", chatID != "")
		return fmt.Errorf("telegram configuration is missing")
	}

	message := msg + "\n\n" + HELPER_VER

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	endpoint := "https://api.telegram.org/bot" +
		botToken +
		"/sendMessage"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		endpoint,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	resp, err := GlobalHTTPClient.Do(req)
	if err != nil {
		rlog.Printf("[TGWebhookLog] http request failed: %v", err)
		eturn fmt.Errorf("failed to send telegram log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"telegram returned status %d",
			resp.StatusCode,
		)
	}

	return nil
}
