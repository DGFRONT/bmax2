package bot

import (
	"context"
	"log/slog"
	"time"
)

// updateTypes are the updates the bot reacts to: the «Начать» button and
// messages written to the bot.
var updateTypes = []string{"bot_started", "message_created"}

type Bot struct {
	api *client
	log *slog.Logger
}

// New returns a bot using the MAX Bot API at apiURL.
func New(apiURL, token string, log *slog.Logger) *Bot {
	return &Bot{api: newClient(apiURL, token), log: log}
}

// Run receives updates and answers them until ctx is cancelled. Network and
// API errors are logged and retried, so Run returns only on shutdown.
func (b *Bot) Run(ctx context.Context) {
	var info botInfo
	for {
		var err error
		if info, err = b.api.me(ctx); err == nil {
			break
		}
		b.log.WarnContext(ctx, "bot: GET /me failed", "err", err)
		if !sleep(ctx, 10*time.Second) {
			return
		}
	}
	if subs, err := b.api.subscriptions(ctx); err == nil && len(subs.Subscriptions) > 0 {
		// MAX does not deliver updates by polling while a webhook is set.
		b.log.WarnContext(ctx, "bot: webhook subscriptions exist, the welcome message will not be sent; remove them with DELETE /subscriptions",
			"url", subs.Subscriptions[0].URL)
	}
	b.log.InfoContext(ctx, "bot: polling updates", "username", info.Username)

	var marker *int64
	backoff := time.Second
	for ctx.Err() == nil {
		resp, err := b.api.updates(ctx, marker, updateTypes)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.log.WarnContext(ctx, "bot: GET /updates failed", "err", err, "retry_in", backoff)
			if !sleep(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, time.Minute)
			continue
		}
		backoff = time.Second
		if resp.Marker != nil {
			marker = resp.Marker
		}
		for _, u := range resp.Updates {
			b.handle(ctx, info, u)
		}
	}
}

func (b *Bot) handle(ctx context.Context, info botInfo, u update) {
	var to *user
	switch u.UpdateType {
	case "bot_started":
		to = u.User
	case "message_created":
		// Only private dialogs: in group chats the bot stays silent.
		if m := u.Message; m != nil && m.Recipient.ChatType == "dialog" {
			to = m.Sender
		}
	}
	if to == nil || to.IsBot || to.UserID == 0 {
		return
	}
	name := to.FirstName
	if name == "" {
		name = to.Name
	}
	if err := b.api.sendToUser(ctx, to.UserID, welcomeMessage(name, info.Username)); err != nil {
		b.log.WarnContext(ctx, "bot: send welcome failed", "user", to.UserID, "err", err)
		return
	}
	b.log.InfoContext(ctx, "bot: welcome sent", "user", to.UserID, "update", u.UpdateType)
}

// sleep waits for d and reports false if ctx was cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
