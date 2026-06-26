package emailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"notes-of-ashen/internal/config"
	apperrors "notes-of-ashen/internal/errors"
)

const defaultSendTimeout = 10 * time.Second

type Sender struct {
	conf config.EmailConf
}

func NewSender(conf config.EmailConf) *Sender {
	return &Sender{conf: conf}
}

func (s *Sender) SendVerifyCode(ctx context.Context, to, purpose, code string) error {
	if !s.conf.Enabled {
		return apperrors.Forbidden("email service is disabled")
	}
	if err := s.validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	host := strings.TrimSpace(s.conf.SMTPHost)
	addr := fmt.Sprintf("%s:%d", host, s.conf.SMTPPort)
	dialer := net.Dialer{Timeout: defaultSendTimeout}

	mode := strings.ToLower(strings.TrimSpace(s.conf.TLSMode))
	if mode == "" {
		mode = "implicit"
	}

	var client *smtp.Client
	switch mode {
	case "implicit":
		conn, err := tls.DialWithDialer(&dialer, "tcp", addr, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return fmt.Errorf("connect smtp server: %w", err)
		}
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		} else {
			_ = conn.SetDeadline(time.Now().Add(defaultSendTimeout))
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("create smtp client: %w", err)
		}
	case "starttls":
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("connect smtp server: %w", err)
		}
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		} else {
			_ = conn.SetDeadline(time.Now().Add(defaultSendTimeout))
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("create smtp client: %w", err)
		}
		// STARTTLS 必须在 Auth 之前完成，先升级为 TLS 连接。
		if err := client.StartTLS(&tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	case "none":
		// 明文传输，仅用于内网测试环境，生产环境请勿使用。
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("connect smtp server: %w", err)
		}
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		} else {
			_ = conn.SetDeadline(time.Now().Add(defaultSendTimeout))
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("create smtp client: %w", err)
		}
	default:
		return fmt.Errorf("unsupported email tls mode: %s", s.conf.TLSMode)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", strings.TrimSpace(s.conf.SMTPUsername), s.conf.SMTPPassword, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	from := s.fromAddress()
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	to = strings.TrimSpace(to)
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set smtp recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open smtp data writer: %w", err)
	}
	message := buildVerifyCodeMessage(from, to, purpose, code)
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp data writer: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit smtp client: %w", err)
	}
	return nil
}

func (s *Sender) validate() error {
	if strings.TrimSpace(s.conf.SMTPHost) == "" ||
		s.conf.SMTPPort <= 0 ||
		strings.TrimSpace(s.conf.SMTPUsername) == "" ||
		strings.TrimSpace(s.conf.SMTPPassword) == "" {
		return apperrors.ServiceUnavailable("email service is not configured")
	}
	return nil
}

func (s *Sender) fromAddress() mail.Address {
	address := strings.TrimSpace(s.conf.From)
	if address == "" {
		address = strings.TrimSpace(s.conf.SMTPUsername)
	}
	return mail.Address{
		Name:    strings.TrimSpace(s.conf.FromName),
		Address: address,
	}
}

func buildVerifyCodeMessage(from mail.Address, to, purpose, code string) string {
	subject := "Notes of Ashen 验证码"
	body := fmt.Sprintf(`你的 %s 验证码是：%s

验证码 5 分钟内有效，请勿泄露给他人。
如果不是你本人操作，请忽略这封邮件。
`, purposeName(purpose), code)

	headers := []string{
		"From: " + from.String(),
		"To: " + strings.TrimSpace(to),
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func purposeName(purpose string) string {
	switch purpose {
	case "register":
		return "注册"
	case "reset_password":
		return "找回密码"
	case "change_password":
		return "修改密码"
	case "update_email":
		return "修改邮箱"
	default:
		return "身份验证"
	}
}
