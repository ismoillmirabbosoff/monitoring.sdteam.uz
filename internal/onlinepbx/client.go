// Package onlinepbx OnlinePBX REST API bilan ishlash uchun klient.
//
// Auth oqimi (OnlinePBX hujjatiga muvofiq):
//
//	POST {base}/{domain}/auth.json   form: auth_key, auth_id
//	-> {"status":"1","data":{"key":"<64hex>","key_id":"<...>"}}
//
// Keyingi barcha so'rovlar `x-pbx-authentication: <key>:<key_id>` header bilan
// yuboriladi. Token muddati tugaganda (errorCode=API_KEY_CHECK_FAILED yoki
// isNotAuth) avtomatik qayta auth qilinadi.
package onlinepbx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	base        string
	domain      string
	apiKey      string
	apiID       string
	keysURL     string // bo'sh bo'lmasa: token/auth_key shu upstream'dan olinadi
	staticToken string // bo'sh bo'lmasa: shu key_and_id to'g'ridan-to'g'ri ishlatiladi
	http        *http.Client

	mu      sync.RWMutex
	token   string // "<key>:<key_id>"
	wsKey   string // websocket uchun kalit (auth_key)
	expires time.Time
}

func New(base, domain, apiKey, apiID string) *Client {
	return &Client{
		base:   strings.TrimRight(base, "/"),
		domain: domain,
		apiKey: apiKey,
		apiID:  apiID,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// SetKeysURL upstream keys rejimini yoqadi: backend o'zi auth.json qilmaydi,
// balki token va auth_key ni shu URL'dan oladi ({key_and_id, auth_key} JSON).
func (c *Client) SetKeysURL(u string) { c.keysURL = u }

// SetStaticToken to'g'ridan-to'g'ri (proxy/auth.json'siz) sessiya tokenini o'rnatadi.
// Hech qanday tashqi xizmatga bog'liq bo'lmaydi — faqat OnlinePBX bilan ishlaydi.
func (c *Client) SetStaticToken(token string) {
	c.staticToken = token
	c.mu.Lock()
	c.token = token
	c.expires = time.Now().Add(100 * 365 * 24 * time.Hour) // amalda muddatsiz
	c.mu.Unlock()
}

func (c *Client) endpoint(path string) string {
	return fmt.Sprintf("%s/%s/%s", c.base, c.domain, strings.TrimLeft(path, "/"))
}

// Token joriy (yoki yangilangan) `x-pbx-authentication` qiymatini qaytaradi.
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expires) {
		t := c.token
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()
	return c.authenticate(ctx)
}

// WSKey websocket uchun kalitni qaytaradi (zarur bo'lsa auth qiladi).
func (c *Client) WSKey(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expires) && c.wsKey != "" {
		k := c.wsKey
		c.mu.RUnlock()
		return k, nil
	}
	c.mu.RUnlock()
	if _, err := c.authenticate(ctx); err != nil {
		return "", err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wsKey, nil
}

type authResp struct {
	Status string `json:"status"`
	Comment string `json:"comment"`
	Data   struct {
		Key   string `json:"key"`
		KeyID string `json:"key_id"`
		// turli OnlinePBX versiyalarida websocket kaliti turlicha nomlanadi:
		AuthKey string `json:"auth_key"`
		WSKey   string `json:"ws_key"`
	} `json:"data"`
}

func (c *Client) authenticate(ctx context.Context) (string, error) {
	if c.staticToken != "" {
		return c.staticToken, nil
	}
	if c.keysURL != "" {
		return c.authenticateUpstream(ctx)
	}
	if c.apiKey == "" {
		return "", fmt.Errorf("ONPBX_API_KEY sozlanmagan — auth qilib bo'lmaydi")
	}
	form := url.Values{}
	form.Set("auth_key", c.apiKey)
	if c.apiID != "" {
		form.Set("auth_id", c.apiID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("auth.json"), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth so'rovi: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var ar authResp
	if err := json.Unmarshal(body, &ar); err != nil {
		return "", fmt.Errorf("auth javobini o'qish: %w (body=%.200s)", err, body)
	}
	if ar.Status != "1" || ar.Data.Key == "" {
		return "", fmt.Errorf("auth muvaffaqiyatsiz: status=%s comment=%s", ar.Status, ar.Comment)
	}

	// OnlinePBX token formati: "<key_id>:<key>"
	token := ar.Data.KeyID + ":" + ar.Data.Key
	// WebSocket kaliti: auth.json bermasa, API kalitning o'zi ishlatiladi.
	wsKey := firstNonEmpty(ar.Data.AuthKey, ar.Data.WSKey, c.apiKey)

	c.mu.Lock()
	c.token = token
	if wsKey != "" {
		c.wsKey = wsKey
	}
	// OnlinePBX sessiya kaliti odatda bir necha soat amal qiladi; xavfsizlik
	// uchun 30 daqiqada bir yangilaymiz.
	c.expires = time.Now().Add(30 * time.Minute)
	c.mu.Unlock()
	return token, nil
}

// authenticateUpstream tokenni boshqa backend'dan oladi (keysURL rejimi).
// Kutilgan javob: {"key_and_id":"<key>:<key_id>","auth_key":"<ws>"}.
func (c *Client) authenticateUpstream(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.keysURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upstream keys so'rovi: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r struct {
		KeyAndID string `json:"key_and_id"`
		AuthKey  string `json:"auth_key"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("upstream keys javobi: %w (body=%.200s)", err, body)
	}
	if r.KeyAndID == "" {
		return "", fmt.Errorf("upstream keys bo'sh key_and_id qaytardi (body=%.200s)", body)
	}

	c.mu.Lock()
	c.token = r.KeyAndID
	if r.AuthKey != "" {
		c.wsKey = r.AuthKey
	}
	c.expires = time.Now().Add(30 * time.Minute)
	c.mu.Unlock()
	return r.KeyAndID, nil
}

// SetWSKey websocket kalitini qo'lda o'rnatish (ONPBX_WS_KEY override uchun).
func (c *Client) SetWSKey(k string) {
	c.mu.Lock()
	c.wsKey = k
	c.mu.Unlock()
}

// doAuthed token bilan POST yuboradi; auth tugagan bo'lsa bir marta qayta urinadi.
func (c *Client) doAuthed(ctx context.Context, path string, form url.Values) ([]byte, error) {
	token, err := c.Token(ctx)
	if err != nil {
		return nil, err
	}
	body, retry, err := c.post(ctx, path, form, token)
	if err != nil {
		return nil, err
	}
	if retry {
		newToken, err := c.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		body, _, err = c.post(ctx, path, form, newToken)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// post bitta so'rov yuboradi; retry=true bo'lsa token yangilash kerak.
func (c *Client) post(ctx context.Context, path string, form url.Values, token string) (body []byte, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-pbx-authentication", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)

	// Token muddati tugaganini aniqlash
	var probe struct {
		IsNotAuth bool   `json:"isNotAuth"`
		ErrorCode string `json:"errorCode"`
		Status    string `json:"status"`
	}
	_ = json.Unmarshal(body, &probe)
	if probe.IsNotAuth || probe.ErrorCode == "API_KEY_CHECK_FAILED" {
		return body, true, nil
	}
	return body, false, nil
}

// ---- Fifo (navbatlar / hozir ishlayotgan operatorlar) ----

type Fifo struct {
	Name  string `json:"name"`
	Num   string `json:"num"`
	Users string `json:"users"` // "201:1;202:1;..." (extension:online)
}

func (c *Client) FifoGet(ctx context.Context) ([]Fifo, error) {
	body, err := c.doAuthed(ctx, "fifo/get.json", url.Values{})
	if err != nil {
		return nil, err
	}
	var r struct {
		Status string `json:"status"`
		Data   []Fifo `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("fifo javobi: %w (body=%.200s)", err, body)
	}
	return r.Data, nil
}

// ---- Users (operator ismlari) ----

type User struct {
	Num     string `json:"num"`     // extension (masalan "206")
	Name    string `json:"name"`    // operator ismi
	Queue   string `json:"tr1"`     // asosiy navbat (fifo)
	Enabled bool   `json:"enabled"` // faolmi
}

func (c *Client) UserGet(ctx context.Context) ([]User, error) {
	body, err := c.doAuthed(ctx, "user/get.json", url.Values{})
	if err != nil {
		return nil, err
	}
	var r struct {
		Status string `json:"status"`
		Data   []User `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("user javobi: %w (body=%.200s)", err, body)
	}
	return r.Data, nil
}

// ---- Call history (mongo_history/search.json) ----

type Call struct {
	UUID              string `json:"uuid"`
	CallerIDNumber    string `json:"caller_id_number"`
	CallerIDName      string `json:"caller_id_name"`
	DestinationNumber string `json:"destination_number"`
	Direction         string `json:"-"` // accountcode dan olinadi
	Accountcode       string `json:"accountcode"`
	Gateway           string `json:"gateway"`
	StartStamp        int64  `json:"start_stamp"`
	EndStamp          int64  `json:"end_stamp"`
	Duration          int64  `json:"duration"`
	UserTalkTime      int64  `json:"user_talk_time"`
	HangupCause       string `json:"hangup_cause"`
}

// gateway maydoni OnlinePBX javobida ba'zan son, ba'zan satr bo'ladi.
func (c *Call) UnmarshalJSON(b []byte) error {
	type alias Call
	aux := &struct {
		Gateway json.RawMessage `json:"gateway"`
		*alias
	}{alias: (*alias)(c)}
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	c.Gateway = rawToString(aux.Gateway)
	c.Direction = c.Accountcode
	return nil
}

// SearchHistory berilgan vaqt oralig'idagi qo'ng'iroqlarni qaytaradi (paging bilan).
func (c *Client) SearchHistory(ctx context.Context, from, to int64, limit int) ([]Call, error) {
	var all []Call
	const page = 500
	for offset := 0; ; offset += page {
		form := url.Values{}
		form.Set("start_stamp_from", strconv.FormatInt(from, 10))
		form.Set("start_stamp_to", strconv.FormatInt(to, 10))
		form.Set("limit", strconv.Itoa(page))
		form.Set("start", strconv.Itoa(offset))

		body, err := c.doAuthed(ctx, "mongo_history/search.json", form)
		if err != nil {
			return nil, err
		}
		var r struct {
			Status string `json:"status"`
			Data   []Call `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("history javobi: %w (body=%.200s)", err, body)
		}
		all = append(all, r.Data...)
		if len(r.Data) < page || (limit > 0 && len(all) >= limit) {
			break
		}
	}
	return all, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func rawToString(r json.RawMessage) string {
	s := strings.TrimSpace(string(r))
	if s == "" || s == "null" {
		return ""
	}
	s = strings.Trim(s, `"`)
	return s
}
