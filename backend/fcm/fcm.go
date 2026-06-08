package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2/google"

	"violin-quest-api/db"
)

// Send fires a notification to each FCM token using the HTTP v1 API.
// Silently no-ops when FCM_PROJECT_ID is not set (e.g. in tests).
// Tokens returning 404 (unregistered device) are auto-deleted from fcm_tokens.
func Send(tokens []string, title, body string) {
	if len(tokens) == 0 {
		return
	}
	projectID := os.Getenv("FCM_PROJECT_ID")
	if projectID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tok, err := accessToken(ctx)
	if err != nil {
		log.Printf("fcm: get access token: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, token := range tokens {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			if err := sendOne(ctx, projectID, tok, t, title, body); err != nil {
				log.Printf("fcm: send failed: %v", err)
			}
		}(token)
	}
	wg.Wait()
}

func sendOne(ctx context.Context, projectID, accessTok, token, title, body string) error {
	payload := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
		},
	}
	b, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", projectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessTok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		db.DeleteFCMToken(token)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func accessToken(ctx context.Context) (string, error) {
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		return "", fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	data, err := os.ReadFile(credPath)
	if err != nil {
		return "", fmt.Errorf("read credentials: %w", err)
	}
	creds, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return "", fmt.Errorf("parse credentials: %w", err)
	}
	tok, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	return tok.AccessToken, nil
}
