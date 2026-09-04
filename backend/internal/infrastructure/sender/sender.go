package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"ops-hub/internal/domain/notification"
	"strings"
	"time"
)

// EmailSender 邮件发送器
type EmailSender struct{}

func NewEmailSender() *EmailSender {
	return &EmailSender{}
}

func (s *EmailSender) Send(ctx context.Context, channel *notification.NotificationChannel, title, content string) error {
	var cfg struct {
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
		SMTPUser string `json:"smtp_user"`
		SMTPPass string `json:"smtp_password"`
		From     string `json:"from_address"`
		To       string `json:"to_addresses"` // 逗号分隔
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return fmt.Errorf("解析邮件配置失败: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.From == "" || cfg.To == "" {
		return fmt.Errorf("邮件配置不完整: 需要 smtp_host, from_address, to_addresses")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}

	to := strings.Split(cfg.To, ",")
	for i := range to {
		to[i] = strings.TrimSpace(to[i])
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.From, strings.Join(to, ","), title, content)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.From, to, []byte(msg))
}

// WecomBotSender 企业微信机器人发送器
type WecomBotSender struct {
	client *http.Client
}

func NewWecomBotSender() *WecomBotSender {
	return &WecomBotSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *WecomBotSender) Send(ctx context.Context, channel *notification.NotificationChannel, title, content string) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return fmt.Errorf("解析企微机器人配置失败: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("webhook_url 不能为空")
	}

	markdown := fmt.Sprintf("## %s\n%s", title, content)
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": markdown,
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送企微消息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("企微接口返回非200: %d", resp.StatusCode)
	}
	return nil
}

// SenderFactory 根据渠道类型获取 Sender
func SenderFactory(channelType notification.ChannelType) notification.Sender {
	switch channelType {
	case notification.ChannelTypeEmail:
		return NewEmailSender()
	case notification.ChannelTypeWecomBot:
		return NewWecomBotSender()
	default:
		return nil
	}
}
