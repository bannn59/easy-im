// prepare_data seeds load-test fixtures over the HTTP API: users, friends,
// conversations, and history messages, then writes per-user session tokens to
// a JSON file for the wrk scripts.
//
// Usage:
//
//	go run ./scripts/prepare_data -base http://localhost:8080 -users 10 -out /tmp/loadtest_tokens.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	base  = flag.String("base", "http://localhost:8080", "API base URL")
	users = flag.Int("users", 10, "number of load-test users to create")
	out   = flag.String("out", "/tmp/loadtest_tokens.json", "output token file")
)

const password = "pass1234"

type client struct {
	hc     *http.Client
	base   string
	cookie string
}

func (c *client) do(method, path string, body any) (*http.Response, []byte) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		log.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Fatalf("%s %s -> %d: %s", method, path, resp.StatusCode, data)
	}
	return resp, data
}

// doRaw is like do but does not log.Fatal on 4xx/5xx — the caller decides.
func (c *client) doRaw(method, path string, body any) (*http.Response, []byte) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		log.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func main() {
	flag.Parse()
	hc := &http.Client{Timeout: 30 * time.Second}

	emails := make([]string, 0, *users)
	usersSet := make([]*client, 0, *users)
	for i := 0; i < *users; i++ {
		email := fmt.Sprintf("lt_%03d@test.local", i)
		emails = append(emails, email)
		c := &client{hc: hc, base: *base}
		// Registration is idempotent across runs: 409 (email exists) is fine.
		if resp, body := c.doRaw("POST", "/v1/auth/register", map[string]string{
			"email": email, "password": password, "display_name": fmt.Sprintf("LoadTester %d", i),
		}); resp.StatusCode != 201 && resp.StatusCode != 200 && resp.StatusCode != 409 {
			log.Fatalf("register %s -> %d: %s", email, resp.StatusCode, body)
		}
		resp, _ := c.do("POST", "/v1/auth/login", map[string]string{"email": email, "password": password})
		var cookie string
		for _, sc := range resp.Header["Set-Cookie"] {
			if strings.HasPrefix(sc, "easyim_session=") {
				cookie = strings.SplitN(sc, ";", 2)[0]
				break
			}
		}
		if cookie == "" {
			log.Fatalf("no session cookie for %s", email)
		}
		c.cookie = cookie
		usersSet = append(usersSet, c)
		log.Printf("user %d ready: %s", i, email)
	}

	// Friendship fan: user0 befriends everyone else. Send request by email from
	// user0, then accept from the peer's incoming list.
	hub := usersSet[0]
	for i := 1; i < *users; i++ {
		peer := usersSet[i]
		// Friend request is idempotent: 409 (already sent) is fine.
		if resp, body := hub.doRaw("POST", "/v1/friends/requests", map[string]string{"email": emails[i]}); resp.StatusCode != 201 && resp.StatusCode != 200 && resp.StatusCode != 409 {
			log.Fatalf("friend request to %s -> %d: %s", emails[i], resp.StatusCode, body)
		}
		_, incoming := peer.do("GET", "/v1/friends/requests/incoming", nil)
		var parsed struct {
			Requests []struct {
				ID string `json:"id"`
			} `json:"requests"`
		}
		if err := json.Unmarshal(incoming, &parsed); err != nil {
			log.Fatalf("parse incoming for %s: %v", emails[i], err)
		}
		// If a pending request exists, accept it. If none, the friendship
		// already exists from a prior run — that is fine.
		if len(parsed.Requests) > 0 {
			peer.do("POST", "/v1/friends/requests/"+parsed.Requests[0].ID+"/accept", nil)
			log.Printf("friendship %s <-> %s", emails[0], emails[i])
		} else {
			log.Printf("friendship %s <-> %s already exists", emails[0], emails[i])
		}
	}

	// Conversations: hub opens a 1:1 conversation with each peer, then seeds
	// history by sending 10 messages into each.
	historyPer := 10
	for i := 1; i < *users; i++ {
		_, openResp := hub.do("POST", "/v1/friends/"+peerEmailToID(hub, emails[i])+"/conversation", nil)
		var conv struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(openResp, &conv); err != nil {
			log.Fatalf("open conversation: %s", openResp)
		}
		for m := 0; m < historyPer; m++ {
			hub.do("POST", "/v1/conversations/"+conv.ID+"/messages", map[string]string{
				"client_msg_id": fmt.Sprintf("seed-%s-%d-%d", emails[i], m, time.Now().UnixNano()),
				"body":          fmt.Sprintf("history message %d from hub", m),
			})
		}
		log.Printf("conversation %s seeded with %d messages", conv.ID, historyPer)
	}

	// Write tokens: for each user a session cookie value.
	type entry struct {
		Email  string `json:"email"`
		Cookie string `json:"cookie"`
	}
	var entries []entry
	for i := 0; i < *users; i++ {
		entries = append(entries, entry{Email: emails[i], Cookie: usersSet[i].cookie})
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d tokens to %s", len(entries), *out)
}

// peerEmailToID resolves a peer's user id by calling GET /v1/friends and
// matching display email. The friends list returns user objects.
func peerEmailToID(hub *client, email string) string {
	_, body := hub.do("GET", "/v1/friends", nil)
	var parsed struct {
		Friends []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"friends"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Fatalf("list friends: %s", body)
	}
	for _, u := range parsed.Friends {
		if u.Email == email {
			return u.ID
		}
	}
	log.Fatalf("peer %s not in hub's friends list", email)
	return ""
}
