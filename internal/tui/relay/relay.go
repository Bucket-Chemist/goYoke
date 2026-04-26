package relay

import (
	"log"

	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// StartDiscordRelay subscribes to store and forwards meaningful snapshot
// changes to the Discord webhook at webhookURL.
//
// If webhookURL is empty the relay is disabled: the function returns
// immediately with a no-op unsubscribe function. This is the default path
// when GOYOKE_DISCORD_WEBHOOK is unset.
//
// The returned function unsubscribes from the store and must be called on
// application shutdown.
func StartDiscordRelay(store *observability.SnapshotStore, webhookURL string) func() {
	if webhookURL == "" {
		return func() {}
	}

	discordRelay := NewDiscordRelay(webhookURL)

	unsub := store.Subscribe(func(old, new harnessproto.SessionSnapshot) {
		message, ok := Summarize(old, new)
		if !ok {
			return
		}
		if err := discordRelay.Send(message); err != nil {
			log.Printf("[relay/discord] send error: %v", err)
		}
	})

	return unsub
}
