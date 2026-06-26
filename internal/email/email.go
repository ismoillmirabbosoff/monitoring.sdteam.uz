// Package email tasdiqlash kodlarini email orqali yuboradi.
// SMTP sozlanmagan bo'lsa, dev rejimда kod server logiga chiqadi.
package email

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Sender struct {
	host string
	port string
	user string
	pass string
	from string
}

func New(host, port, user, pass, from string) *Sender {
	if from == "" {
		from = user
	}
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

// Configured SMTP sozlangan-sozlanmaganini bildiradi.
func (s *Sender) Configured() bool { return s.host != "" }

// SendCode tasdiqlash kodini yuboradi (yoki dev rejimda logga yozadi).
func (s *Sender) SendCode(to, code string) error {
	subject := "Monitoring — kirish kodi"
	body := fmt.Sprintf("Sizning kirish kodingiz: %s\n\nKod 10 daqiqa amal qiladi.\nAgar bu siz bo'lmasangiz, e'tiborsiz qoldiring.", code)

	if !s.Configured() {
		// DEV rejim: haqiqiy email yo'q — kodni logga chiqaramiz
		log.Printf("📧 [DEV] %s uchun kirish kodi: %s  (SMTP sozlanmagan)", to, code)
		return nil
	}

	msg := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := s.host + ":" + s.port
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	if err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("email yuborish: %w", err)
	}
	log.Printf("📧 %s uchun kirish kodi yuborildi", to)
	return nil
}
