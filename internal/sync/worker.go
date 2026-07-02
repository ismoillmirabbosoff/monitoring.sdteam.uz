// Package sync OnlinePBX history'ni periodik ravishda DB'ga tortib oladi.
package sync

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
	"github.com/salesdoc/monitoring-api/internal/store"
)

const pwAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// randomPassword operator akkaunti uchun tasodifiy 10 belgili parol.
func randomPassword() string {
	b := make([]byte, 10)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pwAlphabet))))
		b[i] = pwAlphabet[n.Int64()]
	}
	return string(b)
}

type Worker struct {
	pbx      *onlinepbx.Client
	store    *store.Store
	interval time.Duration
	lookback time.Duration
	limit    int
	now      func() time.Time
}

func New(pbx *onlinepbx.Client, st *store.Store, interval, lookback time.Duration, limit int) *Worker {
	return &Worker{
		pbx:      pbx,
		store:    st,
		interval: interval,
		lookback: lookback,
		limit:    limit,
		now:      time.Now,
	}
}

// Run blok qiladi va ctx bekor qilinmaguncha ishlaydi.
func (w *Worker) Run(ctx context.Context) {
	// Birinchi sinxni darhol bajaramiz.
	w.syncOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("sync: to'xtatildi")
			return
		case <-ticker.C:
			w.syncOnce(ctx)
		}
	}
}

// syncOperators OnlinePBX operatorlarini employees jadvaliga ko'chiradi (ism+kompaniya).
func (w *Worker) syncOperators(ctx context.Context) {
	users, err := w.pbx.UserGet(ctx)
	if err != nil {
		log.Printf("sync: operatorlarni olishda xato: %v", err)
		return
	}
	n, created := 0, 0
	for _, u := range users {
		if u.Num == "" || u.Name == "" {
			continue
		}
		company := onlinepbx.CompanyByQueue(u.Queue)
		if err := w.store.UpsertOperator(ctx, u.Name, u.Num, company); err == nil {
			n++
		}
		// Har operator uchun login akkaunti (agar ext bo'yicha yo'q bo'lsa).
		if w.ensureOperatorUser(ctx, u.Num, u.Name) {
			created++
		}
	}
	if n > 0 {
		log.Printf("sync: %d ta operator employees'ga yangilandi", n)
	}
	if created > 0 {
		log.Printf("sync: %d ta operatorga login akkaunti yaratildi", created)
	}
}

// ensureOperatorUser operator ext'i uchun login akkaunti yaratadi (yo'q bo'lsa).
// Tasodifiy parol bilan; email = {ext}@sdteam.uz. Yaratilsa true qaytaradi.
func (w *Worker) ensureOperatorUser(ctx context.Context, ext, name string) bool {
	if _, err := w.store.UserByExt(ctx, ext); err == nil {
		return false // allaqachon bor
	} else if !errors.Is(err, store.ErrNotFound) {
		return false // DB xatosi — keyingi sinxda urinadi
	}
	pw := randomPassword()
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return false
	}
	email := ext + "@sdteam.uz"
	if _, err := w.store.CreateUser(ctx, email, string(hash), name, "operator", ext, pw); err != nil {
		return false // email band bo'lishi mumkin — o'tkazamiz
	}
	return true
}

func (w *Worker) syncOnce(ctx context.Context) {
	w.syncOperators(ctx)
	now := w.now().Unix()

	// Boshlang'ich nuqta: oxirgi sinxdan beri (overlap bilan), aks holda lookback.
	from := now - int64(w.lookback.Seconds())
	if last, err := w.store.GetLastSyncedTo(ctx); err == nil && last > 0 {
		// kichik overlap — chala yozuvlarni qayta olish uchun
		overlap := int64(5 * 60)
		if last-overlap < from || from == 0 {
			from = last - overlap
		}
	}
	if from < 0 {
		from = 0
	}

	calls, err := w.pbx.SearchHistory(ctx, from, now, w.limit)
	if err != nil {
		log.Printf("sync: history tortishda xato: %v", err)
		return
	}
	n, err := w.store.UpsertCalls(ctx, calls)
	if err != nil {
		log.Printf("sync: DB upsert xatosi: %v", err)
		return
	}
	if err := w.store.SetLastSyncedTo(ctx, now); err != nil {
		log.Printf("sync: holatni saqlashda xato: %v", err)
	}
	if n > 0 {
		log.Printf("sync: %d ta qo'ng'iroq yangilandi (from=%d to=%d)", n, from, now)
	}
}
