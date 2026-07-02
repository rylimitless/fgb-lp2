package mailer

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v3"
)

// Sender is the interface for sending emails.
// Swap implementations by changing the concrete type used in main.go —
// no other code changes needed.
type Sender interface {
	SendWelcome(ctx context.Context, toEmail, toName, tempPassword string) error
	SendEnrollmentConfirmation(ctx context.Context, toEmail, toName, courseTitle string, courseID int64) error
	SendCourseComplete(ctx context.Context, toEmail, toName, courseTitle string, scorePct float64, certificateCode string, badgeNames []string) error
	SendReviewDecision(ctx context.Context, toEmail, toName, itemType, itemTitle, decision, notes string) error
	SendPasswordReset(ctx context.Context, toEmail, toName, resetURL string) error
}

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	client *resend.Client
	from   string
	appURL string
}

// NewResend creates a Resend-backed Sender.
// Reads RESEND_API_KEY, RESEND_FROM, and APP_URL from the environment.
func NewResend() Sender {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("[mailer] RESEND_API_KEY not set — emails will be logged but not sent")
		return &noopSender{tag: "resend"}
	}

	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}

	return &ResendSender{
		client: resend.NewClient(apiKey),
		from:   from,
		appURL: appURL,
	}
}

// SendWelcome sends a welcome email with temporary credentials.
func (s *ResendSender) SendWelcome(ctx context.Context, toEmail, toName, tempPassword string) error {
	loginURL := s.appURL + "/login"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
  <h2 style="color: #1a1a2e; margin-bottom: 8px;">Welcome to FGB Academy</h2>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Your FGB Academy account has been created. Use the credentials below to sign in:</p>

  <div style="background: #f4f4f8; border-radius: 8px; padding: 16px; margin: 20px 0;">
    <table cellpadding="4" style="font-size: 14px;">
      <tr><td style="color: #666; padding-right: 16px;">Email</td><td style="color: #1a1a2e; font-weight: 600;">%s</td></tr>
      <tr><td style="color: #666; padding-right: 16px;">Password</td><td style="color: #1a1a2e; font-weight: 600; font-family: monospace; letter-spacing: 1px;">%s</td></tr>
    </table>
  </div>

  <a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px;">Sign in to FGB Academy</a>

  <p style="color: #888; font-size: 12px; margin-top: 24px;">For security, please change your password after logging in via Account Settings.</p>
</body>
</html>`, toName, toEmail, tempPassword, loginURL)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: "Welcome to FGB Academy",
		Html:    html,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	log.Printf("[mailer] welcome email sent to %s (id: %s)", toEmail, sent.Id)
	return nil
}

// SendEnrollmentConfirmation sends an email when a user is enrolled in a course.
func (s *ResendSender) SendEnrollmentConfirmation(ctx context.Context, toEmail, toName, courseTitle string, courseID int64) error {
	courseURL := fmt.Sprintf("%s/lesson-player?id=%d", s.appURL, courseID)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
  <h2 style="color: #1a1a2e; margin-bottom: 8px;">You've been enrolled!</h2>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">You have been enrolled in <strong>%s</strong> on FGB Academy. You can start learning right away.</p>

  <a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px; margin: 16px 0;">Start Learning</a>

  <p style="color: #888; font-size: 12px; margin-top: 24px;">If you have any questions, reach out to your learning administrator.</p>
</body>
</html>`, toName, courseTitle, courseURL)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Enrolled: %s", courseTitle),
		Html:    html,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	log.Printf("[mailer] enrollment email sent to %s (id: %s)", toEmail, sent.Id)
	return nil
}

// SendCourseComplete sends an email when a user completes a course.
func (s *ResendSender) SendCourseComplete(ctx context.Context, toEmail, toName, courseTitle string, scorePct float64, certificateCode string, badgeNames []string) error {
	certURL := ""
	if certificateCode != "" {
		certURL = fmt.Sprintf("%s/certificates/%s", s.appURL, certificateCode)
	}

	badgeHTML := ""
	if len(badgeNames) > 0 {
		badgeHTML = `<p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Badges earned: `
		for i, name := range badgeNames {
			if i > 0 {
				badgeHTML += ", "
			}
			badgeHTML += fmt.Sprintf("<strong>%s</strong>", name)
		}
		badgeHTML += `</p>`
	}

	certButton := ""
	if certURL != "" {
		certButton = fmt.Sprintf(`<a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px; margin: 16px 0;">View Certificate</a>`, certURL)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
  <h2 style="color: #1a1a2e; margin-bottom: 8px;">Course completed!</h2>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Congratulations %s,</p>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">You've successfully completed <strong>%s</strong> with a score of <strong>%.0f%%</strong>.</p>
  %s
  %s
  <p style="color: #888; font-size: 12px; margin-top: 24px;">Keep up the great work — your progress has been saved to your learning record.</p>
</body>
</html>`, toName, courseTitle, scorePct, badgeHTML, certButton)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Course complete: %s", courseTitle),
		Html:    html,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	log.Printf("[mailer] course-complete email sent to %s (id: %s)", toEmail, sent.Id)
	return nil
}

// SendReviewDecision sends an email when a content review decision is made.
func (s *ResendSender) SendReviewDecision(ctx context.Context, toEmail, toName, itemType, itemTitle, decision, notes string) error {
	decisionLabel := decision
	switch decision {
	case "approved":
		decisionLabel = "Approved"
	case "changes_requested":
		decisionLabel = "Changes Requested"
	case "rejected":
		decisionLabel = "Rejected"
	}

	decisionColor := "#2563eb"
	switch decision {
	case "approved":
		decisionColor = "#16a34a"
	case "changes_requested":
		decisionColor = "#d97706"
	case "rejected":
		decisionColor = "#dc2626"
	}

	notesHTML := ""
	if notes != "" {
		notesHTML = fmt.Sprintf(`<div style="background: #f4f4f8; border-radius: 8px; padding: 16px; margin: 20px 0;">
  <p style="color: #666; font-size: 13px; font-weight: 600; margin: 0 0 8px 0;">Reviewer notes:</p>
  <p style="color: #4a4a6a; font-size: 14px; line-height: 1.6; margin: 0;">%s</p>
</div>`, notes)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
  <h2 style="color: #1a1a2e; margin-bottom: 8px;">Content review update</h2>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Your %s <strong>%s</strong> has been reviewed.</p>

  <div style="margin: 20px 0;">
    <span style="display: inline-block; background: %s; color: white; padding: 6px 16px; border-radius: 20px; font-size: 13px; font-weight: 600;">%s</span>
  </div>

  %s

  <a href="%s/content-repository" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px;">View in Content Repository</a>

  <p style="color: #888; font-size: 12px; margin-top: 24px;">For questions, reply to this email or contact your reviewer directly.</p>
</body>
</html>`, toName, itemType, itemTitle, decisionColor, decisionLabel, notesHTML, s.appURL)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Review: %s — %s", itemTitle, decisionLabel),
		Html:    html,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	log.Printf("[mailer] review-decision email sent to %s (id: %s)", toEmail, sent.Id)
	return nil
}

// SendPasswordReset sends a password reset email with a reset link.
func (s *ResendSender) SendPasswordReset(ctx context.Context, toEmail, toName, resetURL string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
  <h2 style="color: #1a1a2e; margin-bottom: 8px;">Reset your password</h2>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #4a4a6a; font-size: 15px; line-height: 1.6;">A password reset was requested for your FGB Academy account. Click the button below to set a new password. This link expires in 1 hour.</p>

  <a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px; margin: 16px 0;">Reset Password</a>

  <p style="color: #888; font-size: 12px; margin-top: 24px;">If you did not request this, you can safely ignore this email. Your password will not change.</p>
</body>
</html>`, toName, resetURL)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: "Reset your FGB Academy password",
		Html:    html,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	log.Printf("[mailer] password-reset email sent to %s (id: %s)", toEmail, sent.Id)
	return nil
}

// noopSender logs emails instead of sending them.
// Used when RESEND_API_KEY is not configured (dev/test mode).
type noopSender struct {
	tag string
}

func (n *noopSender) SendWelcome(ctx context.Context, toEmail, toName, tempPassword string) error {
	log.Printf("[mailer:%s] WOULD SEND welcome to %s <%s> (password: %s)", n.tag, toName, toEmail, tempPassword)
	return nil
}

func (n *noopSender) SendEnrollmentConfirmation(ctx context.Context, toEmail, toName, courseTitle string, courseID int64) error {
	log.Printf("[mailer:%s] WOULD SEND enrollment to %s <%s> for course %q (id=%d)", n.tag, toName, toEmail, courseTitle, courseID)
	return nil
}

func (n *noopSender) SendCourseComplete(ctx context.Context, toEmail, toName, courseTitle string, scorePct float64, certificateCode string, badgeNames []string) error {
	log.Printf("[mailer:%s] WOULD SEND course-complete to %s <%s> for %q (score=%.0f%%, cert=%s, badges=%v)", n.tag, toName, toEmail, courseTitle, scorePct, certificateCode, badgeNames)
	return nil
}

func (n *noopSender) SendReviewDecision(ctx context.Context, toEmail, toName, itemType, itemTitle, decision, notes string) error {
	log.Printf("[mailer:%s] WOULD SEND review-decision to %s <%s> for %s %q: %s", n.tag, toName, toEmail, itemType, itemTitle, decision)
	return nil
}

func (n *noopSender) SendPasswordReset(ctx context.Context, toEmail, toName, resetURL string) error {
	log.Printf("[mailer:%s] WOULD SEND password-reset to %s <%s> (url: %s)", n.tag, toName, toEmail, resetURL)
	return nil
}
