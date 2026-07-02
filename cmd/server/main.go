package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/salesdoc/monitoring-api/internal/api"
	"github.com/salesdoc/monitoring-api/internal/config"
	"github.com/salesdoc/monitoring-api/internal/email"
	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
	"github.com/salesdoc/monitoring-api/internal/store"
	syncw "github.com/salesdoc/monitoring-api/internal/sync"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("DB ulandi va migratsiya bajarildi")

	// Email sender (SMTP yoki dev rejim)
	mailer := email.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	if mailer.Configured() {
		log.Printf("email: SMTP %s:%s", cfg.SMTPHost, cfg.SMTPPort)
	} else {
		log.Println("email: SMTP sozlanmagan — DEV rejim (kodlar logga chiqadi)")
	}

	// Boshlang'ich admin (agar admin yo'q bo'lsa)
	if n, err := st.CountAdmins(ctx); err == nil && n == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(cfg.AdminPass), bcrypt.DefaultCost)
		if _, err := st.CreateUser(ctx, cfg.AdminEmail, string(hash), "Administrator", "admin", "", ""); err != nil {
			log.Printf("bootstrap admin xatosi: %v", err)
		} else {
			log.Printf("boshlang'ich admin yaratildi: %s (parol: ADMIN_PASSWORD)", cfg.AdminEmail)
		}
	}

	pbx := onlinepbx.New(cfg.OnpbxBase, cfg.OnpbxDomain, cfg.OnpbxAPIKey, cfg.OnpbxAPIID)
	switch {
	case cfg.OnpbxAPIKey != "":
		// Eng yaxshi: backend tokenni o'zi auth.json orqali yangilab turadi (muddatsiz)
		log.Println("mustaqil auth rejimi: api_key (token avto-yangilanadi)")
	case cfg.OnpbxToken != "":
		pbx.SetStaticToken(cfg.OnpbxToken) // vaqtinchalik: token eskirishi mumkin
		log.Println("static token rejimi: to'g'ridan-to'g'ri OnlinePBX (token eskirishi mumkin)")
	case cfg.OnpbxKeysURL != "":
		pbx.SetKeysURL(cfg.OnpbxKeysURL)
		log.Printf("upstream keys rejimi: %s", cfg.OnpbxKeysURL)
	}
	if cfg.OnpbxWSKey != "" {
		pbx.SetWSKey(cfg.OnpbxWSKey) // websocket kaliti
	}

	// Sinx worker (history -> DB) — xom kalitlar, static token yoki upstream bo'lsa ishlaydi
	canAuth := cfg.OnpbxAPIKey != "" || cfg.OnpbxKeysURL != "" || cfg.OnpbxToken != ""
	if canAuth {
		worker := syncw.New(pbx, st, cfg.SyncInterval, cfg.SyncLookback, cfg.HistoryLimit)
		go worker.Run(ctx)
		log.Printf("sync worker ishga tushdi (interval=%s)", cfg.SyncInterval)
	} else {
		log.Println("OGOHLANTIRISH: ONPBX_API_KEY/ID yoki ONPBX_KEYS_URL yo'q — sinx va /keys ishlamaydi")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(st, pbx, mailer, cfg.CORSOrigins, cfg.OnpbxDomain, cfg.WSPort, cfg.WebDir, cfg.AdminPass).Handler(),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("HTTP server: %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("to'xtatilmoqda...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
