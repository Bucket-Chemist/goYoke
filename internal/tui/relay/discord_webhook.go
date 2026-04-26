package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// defaultRateInterval is the minimum duration between consecutive Discord
// webhook POST requests (Discord allows ~30 req/min per webhook).
const defaultRateInterval = 2 * time.Second

// DiscordRelay sends messages to a Discord channel via an Incoming Webhook URL.
//
// Its zero value is not usable; use NewDiscordRelay.
type DiscordRelay struct {
	webhookURL   string
	client       *http.Client
	rateInterval time.Duration

	mu       sync.Mutex
	lastSent time.Time
}

// NewDiscordRelay returns a DiscordRelay configured to POST to webhookURL.
func NewDiscordRelay(webhookURL string) *DiscordRelay {
	return &DiscordRelay{
		webhookURL:   webhookURL,
		rateInterval: defaultRateInterval,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send posts message to the configured Discord webhook.
//
// An empty message is a no-op. Consecutive calls are rate-limited to
// rateInterval (defaultRateInterval by default) to stay well within Discord's
// per-webhook limit. Callers serialise through a mutex, so concurrent Send
// calls queue rather than spray the webhook simultaneously.
func (d *DiscordRelay) Send(message string) error {
	if message == "" {
		return nil
	}

	d.mu.Lock()
	if elapsed := time.Since(d.lastSent); elapsed < d.rateInterval {
		time.Sleep(d.rateInterval - elapsed)
	}
	d.lastSent = time.Now()
	d.mu.Unlock()

	body, err := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: message})
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord webhook POST: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
