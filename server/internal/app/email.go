package app

import (
	"context"
	"fmt"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/email"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

type schoolEmailSender struct {
	sender       genericEmailSender
	otpSender    otpEmailSender
	subject      string
	purpose      string
	schoolName   string
	expireMinute int
}

type genericEmailSender interface {
	Send(ctx context.Context, to string, subject string, textBody string) error
}

type otpEmailSender interface {
	SendOTP(ctx context.Context, to string, subject string, code string, purpose string, schoolName string, expireMinutes int) error
}

func newSchoolEmailSender(cfg config.EmailConfig, database *db.DB) (*schoolEmailSender, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	template := schoolEmailSender{
		subject:      strings.TrimSpace(cfg.StudentVerificationSubject),
		purpose:      cfg.TencentTemplatePurpose,
		schoolName:   cfg.TencentTemplateSchoolName,
		expireMinute: cfg.TencentTemplateExpireMinutes,
	}
	if template.subject == "" {
		template.subject = "学生认证验证码"
	}
	switch strings.TrimSpace(cfg.Driver) {
	case "", "smtp":
		sender, err := email.NewSMTPSender(email.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.From,
			FromName: cfg.FromName,
			UseTLS:   cfg.UseTLS,
			StartTLS: cfg.StartTLS,
		})
		if err != nil {
			return nil, err
		}
		template.sender = sender
		return &template, nil
	case "blackhole":
		template.sender = email.NewBlackholeSender()
		return &template, nil
	case "tencent_ses":
		sender, err := email.NewTencentSESSender(email.TencentSESConfig{
			SecretID:       cfg.TencentSecretID,
			SecretKey:      cfg.TencentSecretKey,
			Region:         cfg.TencentRegion,
			Endpoint:       cfg.TencentEndpoint,
			From:           cfg.From,
			FromName:       cfg.FromName,
			ReplyTo:        cfg.TencentReplyTo,
			TemplateID:     cfg.TencentTemplateID,
			DefaultPurpose: cfg.TencentTemplatePurpose,
			DefaultSchool:  cfg.TencentTemplateSchoolName,
			DefaultExpire:  cfg.TencentTemplateExpireMinutes,
		})
		if err != nil {
			return nil, err
		}
		template.otpSender = sender
		return &template, nil
	case "resend":
		sender, err := newResendSender(cfg)
		if err != nil {
			return nil, err
		}
		template.otpSender = sender
		return &template, nil
	case "multi":
		sender, err := newMultiProviderOTPSender(cfg, database)
		if err != nil {
			return nil, err
		}
		template.otpSender = sender
		return &template, nil
	default:
		return nil, fmt.Errorf("unsupported email driver: %s", cfg.Driver)
	}
}

func newTencentSESSender(cfg config.EmailConfig) (email.OTPSender, error) {
	return email.NewTencentSESSender(email.TencentSESConfig{
		SecretID:       cfg.TencentSecretID,
		SecretKey:      cfg.TencentSecretKey,
		Region:         cfg.TencentRegion,
		Endpoint:       cfg.TencentEndpoint,
		From:           cfg.From,
		FromName:       cfg.FromName,
		ReplyTo:        cfg.TencentReplyTo,
		TemplateID:     cfg.TencentTemplateID,
		DefaultPurpose: cfg.TencentTemplatePurpose,
		DefaultSchool:  cfg.TencentTemplateSchoolName,
		DefaultExpire:  cfg.TencentTemplateExpireMinutes,
	})
}

func newResendSender(cfg config.EmailConfig) (email.OTPSender, error) {
	replyTo := strings.TrimSpace(cfg.ResendReplyTo)
	if replyTo == "" {
		replyTo = cfg.TencentReplyTo
	}
	return email.NewResendSender(email.ResendConfig{
		APIKey:   cfg.ResendAPIKey,
		Endpoint: cfg.ResendEndpoint,
		From:     cfg.From,
		FromName: cfg.FromName,
		ReplyTo:  replyTo,
	})
}

func newMultiProviderOTPSender(cfg config.EmailConfig, database *db.DB) (email.OTPSender, error) {
	tencentSender, err := newTencentSESSender(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize Tencent SES provider: %w", err)
	}
	resendSender, err := newResendSender(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize Resend provider: %w", err)
	}
	providers := []email.OTPProvider{
		{Name: email.ProviderTencentSES, Sender: tencentSender},
		{Name: email.ProviderResend, Sender: resendSender},
	}
	available := map[string]email.OTPSender{
		email.ProviderTencentSES: tencentSender,
		email.ProviderResend:     resendSender,
	}
	policy, err := email.ParseDeliveryPolicy(cfg.ProviderPolicy)
	if err != nil {
		return nil, err
	}
	if len(policy.Providers) == 0 {
		policy = email.DeliveryPolicy{
			Mode:        "priority",
			MaxAttempts: 2,
			Providers: []email.DeliveryPolicyEntry{
				{Name: email.ProviderTencentSES, Enabled: true, Priority: 10, Weight: 100},
				{Name: email.ProviderResend, Enabled: true, Priority: 20, Weight: 100},
			},
		}
	}
	policy = email.NormalizeDeliveryPolicy(policy, available)
	policyProvider := email.NewSystemConfigPolicyProvider(systemconfig.NewRepository(database), policy, available)
	return email.NewFailoverOTPSender(providers, policy, policyProvider)
}

func (s *schoolEmailSender) SendAdmissionOTP(ctx context.Context, emailAddress string, code string) error {
	return s.sendOTP(ctx, emailAddress, code)
}

func (s *schoolEmailSender) SendStudentVerificationOTP(ctx context.Context, emailAddress string, code string) error {
	return s.sendOTP(ctx, emailAddress, code)
}

func (s *schoolEmailSender) sendOTP(ctx context.Context, emailAddress string, code string) error {
	if s == nil {
		return fmt.Errorf("school email sender is not configured")
	}
	normalizedCode := strings.TrimSpace(code)
	if s.otpSender != nil {
		return s.otpSender.SendOTP(
			ctx,
			emailAddress,
			s.subject,
			normalizedCode,
			s.purpose,
			s.schoolName,
			s.expireMinute,
		)
	}
	if s.sender == nil {
		return fmt.Errorf("school email sender is not configured")
	}
	body := fmt.Sprintf("你的 StuHelper 学生认证验证码是：%s\n\n验证码 5 分钟内有效。若非本人操作，请忽略本邮件。", normalizedCode)
	return s.sender.Send(ctx, emailAddress, s.subject, body)
}
