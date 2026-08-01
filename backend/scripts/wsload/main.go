// ws_load measures how many concurrent WebSocket connections a node (or the
// nginx LB) can sustain. It opens N connections at increasing concurrency,
// verifies each upgrade succeeds, and reports success rate + handshake latency.
//
// Usage:
//
//	go run ./scripts/ws_load -url ws://localhost:8081/v1/ws -cookie "<easyim_session=...>" -steps 50,100,200,500
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	url    = flag.String("url", "ws://localhost:8081/v1/ws", "WS endpoint")
	cookie = flag.String("cookie", "", "session cookie header value (easyim_session=...)")
	steps  = flag.String("steps", "50,100,200,500", "comma-separated concurrency levels")
)

func main() {
	flag.Parse()
	if *cookie == "" {
		fmt.Fprintln(os.Stderr, "provide -cookie (easyim_session=...); use a load-test user from prepare_data tokens")
		os.Exit(2)
	}
	levels := []int{}
	for _, s := range splitInts(*steps) {
		levels = append(levels, s)
	}

	for _, n := range levels {
		fmt.Printf("=== %d concurrent connections ===\n", n)
		runStep(n)
	}
}

func splitInts(s string) []int {
	var out []int
	cur := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			cur = cur*10 + int(c-'0')
		} else {
			out = append(out, cur)
			cur = 0
		}
	}
	out = append(out, cur)
	return out
}

func runStep(n int) {
	hdr := http.Header{}
	hdr.Set("Cookie", *cookie)
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, fail := 0, 0
	var latSum time.Duration

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t0 := time.Now()
			conn, _, err := dialer.Dial(*url, hdr)
			if err != nil {
				mu.Lock()
				fail++
				mu.Unlock()
				return
			}
			lat := time.Since(t0)
			// Hold the connection briefly to confirm it stays alive.
			conn.WriteMessage(websocket.PingMessage, nil)
			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			conn.ReadMessage()
			conn.Close()
			mu.Lock()
			ok++
			latSum += lat
			mu.Unlock()
		}()
	}
	wg.Wait()

	total := ok + fail
	fmt.Printf("  connected: %d/%d (%.1f%%)\n", ok, total, float64(ok)*100/float64(total))
	if ok > 0 {
		fmt.Printf("  avg handshake: %s\n", (latSum/time.Duration(ok)).Round(time.Microsecond))
	}
	fmt.Printf("  wall time: %s\n", time.Since(start).Round(time.Millisecond))
	if fail > 0 {
		log.Printf("  %d dial failures at concurrency %d", fail, n)
	}
}
