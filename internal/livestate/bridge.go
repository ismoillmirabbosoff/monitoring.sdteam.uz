// Package livestate OnlinePBX WebSocket'iga doimiy ulanib, operatorlarning jonli
// holatini (registratsiya + BLF) xotirada saqlaydi. OnlinePBX REST'da registratsiya
// statusi yo'q va WS subscribe'da snapshot bermaydi — shu sabab bu daemon 24/7 ishlab,
// har register hodisasini (telefonlar ~120s da qayta register bo'ladi) yig'ib boradi.
package livestate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	regTTL      = 150 * time.Second // registratsiya amal qilish muddati (expires 120 + margin)
	transientTTL = 60 * time.Second // talking/ringing kabi o'tkinchi holatlar
)

type opState struct {
	regState string    // register/registered/register_attempt/unregister/...
	regAt    time.Time // oxirgi registratsiya hodisasi
	blf      string    // talking/ringing/dnd/'' (blf/status hodisasidan)
	blfAt    time.Time
}

// Bridge OnlinePBX WS holatini saqlaydi.
type Bridge struct {
	wsURL string

	mu        sync.RWMutex
	state     map[string]*opState
	connected bool
	lastEvent time.Time
	version   int64
}

func New(domain, wsPort, key string) *Bridge {
	if wsPort == "" {
		wsPort = "3342"
	}
	return &Bridge{
		wsURL: fmt.Sprintf("wss://%s:%s/?key=%s", domain, wsPort, key),
		state: map[string]*opState{},
	}
}

// Run doimiy ulanish/qayta-ulanish siklini boshqaradi (ctx bekor qilinmaguncha).
func (b *Bridge) Run(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := b.connect(ctx); err != nil {
			log.Printf("livestate: ulanish xatosi: %v", err)
		}
		b.setConnected(false)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (b *Bridge) connect(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	c, _, err := dialer.DialContext(ctx, b.wsURL, nil)
	if err != nil {
		return err
	}
	defer c.Close()
	b.setConnected(true)
	log.Println("livestate: OnlinePBX WS'ga ulandi")

	sub := map[string]any{
		"command": "subscribe",
		"reqId":   fmt.Sprint(time.Now().Unix()),
		"data":    map[string]any{"eventGroups": []string{"user_blf", "user_registration", "user_status"}},
	}
	if err := c.WriteJSON(sub); err != nil {
		return err
	}

	// server bo'sh bo'lsa ham ulanishni tirik ushlab turish uchun ping
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				b.mu.RLock()
				conn := b.connected
				b.mu.RUnlock()
				if !conn {
					return
				}
				c.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, msg, err := c.ReadMessage()
		if err != nil {
			return err
		}
		b.ingest(msg)
	}
}

func (b *Bridge) ingest(msg []byte) {
	var e struct {
		Event string `json:"event"`
		Data  struct {
			UID    string `json:"uid"`
			State  string `json:"state"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(msg, &e) != nil {
		return
	}
	uid := e.Data.UID
	if uid == "" {
		return
	}
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastEvent = now
	st := b.state[uid]
	if st == nil {
		st = &opState{}
		b.state[uid] = st
	}
	switch e.Event {
	case "user_registration":
		st.regState = strings.ToLower(e.Data.State)
		st.regAt = now
	case "user_blf", "user_status":
		s := strings.ToLower(e.Data.Status)
		switch {
		case s == "talking" || s == "busy" || s == "answered":
			st.blf, st.blfAt = "talking", now
		case s == "ringing" || s == "early":
			st.blf, st.blfAt = "ringing", now
		case s == "dnd" || s == "do_not_disturb":
			st.blf, st.blfAt = "dnd", now
		case s == "idle" || s == "available" || s == "ready" || s == "hangup":
			st.blf, st.blfAt = "", now
		}
	}
	b.version++
}

// Snapshot har operator uchun yakuniy holatni qaytaradi:
// online / offline / talking / ringing / dnd.
func (b *Bridge) Snapshot() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	now := time.Now()
	out := make(map[string]string, len(b.state))
	for ext, st := range b.state {
		// registratsiya bo'yicha asosiy holat
		registered := st.regState != "" &&
			!strings.HasPrefix(st.regState, "unregister") &&
			now.Sub(st.regAt) < regTTL
		if !registered {
			// aniq unregister yoki muddat tugagan → offline
			if st.regState != "" {
				out[ext] = "offline"
			}
			continue
		}
		// BLF (talking/ringing o'tkinchi; dnd yopishqoq)
		switch st.blf {
		case "talking", "ringing":
			if now.Sub(st.blfAt) < transientTTL {
				out[ext] = st.blf
				continue
			}
		case "dnd":
			out[ext] = "dnd"
			continue
		}
		out[ext] = "online"
	}
	return out
}

func (b *Bridge) Connected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

func (b *Bridge) Version() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.version
}

func (b *Bridge) setConnected(v bool) {
	b.mu.Lock()
	b.connected = v
	b.mu.Unlock()
}
