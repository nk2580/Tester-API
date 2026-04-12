package services

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/nk2580/Tester-API/internal/config"
)

type EmailSender interface {
	SendPasswordResetEmail(toEmail, resetLink string) error
}

type LogEmailSender struct {
	logger *slog.Logger
	from   string
}

func (s LogEmailSender) SendPasswordResetEmail(toEmail, resetLink string) error {
	s.logger.Info("password reset email", "to", toEmail, "from", s.from, "reset_link", resetLink)
	return nil
}

type SMTPEmailSender struct {
	cfg config.Config
}

func (s SMTPEmailSender) SendPasswordResetEmail(toEmail, resetLink string) error {
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: Password Reset\r\n\r\nUse this link to reset your password: %s\r\n", toEmail, resetLink))
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	tlsConfig := &tls.Config{ServerName: s.cfg.SMTPHost}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(s.cfg.EmailFrom); err != nil {
		return err
	}
	if err := client.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func NewEmailSender(cfg config.Config, logger *slog.Logger) EmailSender {
	if cfg.EmailProvider == "smtp" {
		return SMTPEmailSender{cfg: cfg}
	}
	return LogEmailSender{logger: logger, from: cfg.EmailFrom}
}
