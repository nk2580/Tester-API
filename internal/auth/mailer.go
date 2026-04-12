package auth

import "log"

type Mailer interface {
	SendVerificationEmail(toEmail, token string) error
	SendPasswordResetEmail(toEmail, token string) error
	SendPasswordResetConfirmationEmail(toEmail string) error
}

type LogMailer struct{}

func (LogMailer) SendVerificationEmail(toEmail, token string) error {
	_ = token
	log.Printf("verification email queued for %s", toEmail)
	return nil
}

func (LogMailer) SendPasswordResetEmail(toEmail, token string) error {
	_ = token
	log.Printf("password reset email queued for %s", toEmail)
	return nil
}

func (LogMailer) SendPasswordResetConfirmationEmail(toEmail string) error {
	log.Printf("password reset confirmation email queued for %s", toEmail)
	return nil
}
