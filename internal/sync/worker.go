// Package sync OnlinePBX history'ni periodik ravishda DB'ga tortib oladi.
package sync

import (
	"context"
	"log"
	"time"

	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
	"github.com/salesdoc/monitoring-api/internal/store"
)

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
	n := 0
	for _, u := range users {
		if u.Num == "" || u.Name == "" {
			continue
		}
		company := onlinepbx.CompanyByQueue(u.Queue)
		if err := w.store.UpsertOperator(ctx, u.Name, u.Num, company); err == nil {
			n++
		}
	}
	if n > 0 {
		log.Printf("sync: %d ta operator employees'ga yangilandi", n)
	}
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
