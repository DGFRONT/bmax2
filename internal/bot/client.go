// Package bot answers the MAX bot chat: it greets users with a description of
// the service and buttons that open the mini app. Updates are received by long
// polling (https://dev.max.ru/docs-api/methods/GET/updates).
package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// pollTimeout is how long the API holds a GET /updates request open.
const pollTimeout = 30 * time.Second

type client struct {
	base  string
	token string
	http  *http.Client
}

func newClient(base, token string) *client {
	return &client{
		base:  base,
		token: token,
		http:  &http.Client{Timeout: pollTimeout + 15*time.Second},
	}
}

type botInfo struct {
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type user struct {
	UserID    int64  `json:"user_id"`
	FirstName string `json:"first_name"`
	Name      string `json:"name"`
	IsBot     bool   `json:"is_bot"`
}

type update struct {
	UpdateType string   `json:"update_type"`
	User       *user    `json:"user"`
	Message    *message `json:"message"`
}

type message struct {
	Sender    *user `json:"sender"`
	Recipient struct {
		ChatType string `json:"chat_type"`
	} `json:"recipient"`
	Body struct {
		Text string `json:"text"`
	} `json:"body"`
}

type updatesResponse struct {
	Updates []update `json:"updates"`
	Marker  *int64   `json:"marker"`
}

type subscriptionsResponse struct {
	Subscriptions []struct {
		URL string `json:"url"`
	} `json:"subscriptions"`
}

type newMessage struct {
	Text        string       `json:"text"`
	Format      string       `json:"format,omitempty"`
	Attachments []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Type    string          `json:"type"`
	Payload keyboardPayload `json:"payload"`
}

type keyboardPayload struct {
	Buttons [][]button `json:"buttons"`
}

type button struct {
	Type string `json:"type"`
	Text string `json:"text"`
	// WebApp is the username of the bot whose mini app an open_app button launches.
	WebApp string `json:"web_app,omitempty"`
	// Payload of an open_app button reaches the mini app as start_param.
	Payload string `json:"payload,omitempty"`
}

func (c *client) me(ctx context.Context) (botInfo, error) {
	var info botInfo
	err := c.do(ctx, http.MethodGet, "/me", nil, nil, &info)
	return info, err
}

func (c *client) subscriptions(ctx context.Context) (subscriptionsResponse, error) {
	var resp subscriptionsResponse
	err := c.do(ctx, http.MethodGet, "/subscriptions", nil, nil, &resp)
	return resp, err
}

func (c *client) updates(ctx context.Context, marker *int64, types []string) (updatesResponse, error) {
	q := url.Values{}
	q.Set("timeout", strconv.Itoa(int(pollTimeout/time.Second)))
	q.Set("limit", "100")
	for _, t := range types {
		q.Add("types", t)
	}
	if marker != nil {
		q.Set("marker", strconv.FormatInt(*marker, 10))
	}
	var resp updatesResponse
	err := c.do(ctx, http.MethodGet, "/updates", q, nil, &resp)
	return resp, err
}

func (c *client) sendToUser(ctx context.Context, userID int64, msg newMessage) error {
	q := url.Values{"user_id": {strconv.FormatInt(userID, 10)}}
	return c.do(ctx, http.MethodPost, "/messages", q, msg, nil)
}

func (c *client) do(ctx context.Context, method, path string, q url.Values, body, out any) error {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
