package services

import (
	"fmt"
	"log"

	"bethapi/config"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
}

var Email *EmailService

func InitEmail() {
	client := resend.NewClient(config.AppConfig.ResendAPIKey)
	Email = &EmailService{client: client}
	log.Println("Email Service (Resend) Initialized")
}

func (s *EmailService) SendOTP(to string, code string) error {
	params := &resend.SendEmailRequest{
		From:    "Beth AI <no-reply@sipstory.tech>",
		To:      []string{to},
		Subject: "Your Beth AI verification code",
		Html:    fmt.Sprintf("<strong>Your verification code is: %s</strong>", code),
	}

	_, err := s.client.Emails.Send(params)
	return err
}

func (s *EmailService) SendPaymentNotification(to string, amount float64, credits float64) error {
	params := &resend.SendEmailRequest{
		From:    "Beth AI Billing <billing@sipstory.tech>",
		To:      []string{to},
		Subject: "Payment Successful",
		Html:    fmt.Sprintf("Hello, your payment of %.2f was successful. %.2f credits have been added to your account.", amount, credits),
	}

	_, err := s.client.Emails.Send(params)
	return err
}

func (s *EmailService) SendGracePeriodWarning(to string, hoursLeft int, attempt int) error {
	params := &resend.SendEmailRequest{
		From:    "Beth AI Billing <billing@sipstory.tech>",
		To:      []string{to},
		Subject: "Subscription Renewal Failed - Action Required",
		Html:    fmt.Sprintf("Your subscription renewal failed (Attempt %d). You have %d hours left before your account is downgraded.", attempt, hoursLeft),
	}

	_, err := s.client.Emails.Send(params)
	return err
}
