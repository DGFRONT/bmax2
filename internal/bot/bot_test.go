package bot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAPI serves one batch of updates and records sent messages.
type fakeAPI struct {
	mu     sync.Mutex
	polled int
	sent   map[string]newMessage
	done   chan struct{}
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "tok" {
		http.Error(w, `{"code":"verify.token"}`, http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method + " " + r.URL.Path {
	case "GET /me":
		io.WriteString(w, `{"user_id":1,"name":"Сервисы","username":"rso_bot","is_bot":true}`)
	case "GET /subscriptions":
		io.WriteString(w, `{"subscriptions":[]}`)
	case "GET /updates":
		f.polled++
		if f.polled > 1 {
			if r.URL.Query().Get("marker") != "7" {
				http.Error(w, "marker not passed", http.StatusBadRequest)
				return
			}
			io.WriteString(w, `{"updates":[],"marker":7}`)
			return
		}
		io.WriteString(w, `{"marker":7,"updates":[
			{"update_type":"bot_started","chat_id":5,"user":{"user_id":10,"first_name":"Алан","is_bot":false}},
			{"update_type":"message_created","message":{"sender":{"user_id":11,"name":"Зарина"},"recipient":{"chat_id":6,"chat_type":"dialog"},"body":{"text":"привет"}}},
			{"update_type":"message_created","message":{"sender":{"user_id":12,"name":"Группа"},"recipient":{"chat_id":-3,"chat_type":"chat"},"body":{"text":"всем"}}},
			{"update_type":"message_created","message":{"sender":{"user_id":13,"is_bot":true},"recipient":{"chat_type":"dialog"},"body":{"text":"bot"}}}
		]}`)
	case "POST /messages":
		var m newMessage
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.sent[r.URL.Query().Get("user_id")] = m
		io.WriteString(w, `{"message":{}}`)
		if len(f.sent) == 2 {
			close(f.done)
		}
	default:
		http.NotFound(w, r)
	}
}

func TestRunSendsWelcome(t *testing.T) {
	api := &fakeAPI{sent: map[string]newMessage{}, done: make(chan struct{})}
	ts := httptest.NewServer(api)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		New(ts.URL, "tok", slog.New(slog.NewTextHandler(io.Discard, nil))).Run(ctx)
		close(stopped)
	}()
	select {
	case <-api.done:
	case <-time.After(5 * time.Second):
		t.Fatal("welcome messages were not sent")
	}
	cancel()
	<-stopped

	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.sent) != 2 {
		t.Fatalf("sent to %d users, want 2 (no replies in groups or to bots)", len(api.sent))
	}
	if !strings.HasPrefix(api.sent["10"].Text, "Добро пожаловать в официальный каталог сервисов") {
		t.Errorf("text = %q", api.sent["10"].Text)
	}
	kb := api.sent["10"].Attachments
	if len(kb) != 1 || kb[0].Type != "inline_keyboard" || len(kb[0].Payload.Buttons) != 1 {
		t.Fatalf("attachments = %+v", kb)
	}
	want := button{Type: "open_app", Text: "Открыть", WebApp: "rso_bot"}
	if got := kb[0].Payload.Buttons[0]; len(got) != 1 || got[0] != want {
		t.Errorf("buttons = %+v, want [%+v]", got, want)
	}
}
