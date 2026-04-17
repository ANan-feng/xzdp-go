package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"os"
	"time"
)

const emailCodePrefix = "xzdp:email:code:"
const emailLimitPrefix = "xzdp:email:limit:"

// SendEmailCode 发送邮箱验证码
// 入参：context.Context + 显式email参数（移除gin.Context依赖）
func SendEmailCode(ctx context.Context, email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	codeKey := emailCodePrefix + email   // 存验证码
	limitKey := emailLimitPrefix + email // 存频率限制

	// 检查60秒内是否发送过
	//频率限制
	exists, err := RedisClient.Exists(ctx, limitKey).Result()
	if err != nil {
		return fmt.Errorf("redis 异常: %v", err)
	}
	if exists == 1 {
		ttl, _ := RedisClient.TTL(ctx, limitKey).Result()
		return fmt.Errorf("操作过于频繁，请在 %d 秒后重试", int(ttl.Seconds()))
	}

	// 检查验证码是否仍在有效期
	//时间限制
	exists, err = RedisClient.Exists(ctx, codeKey).Result()
	if err != nil {
		return fmt.Errorf("redis 异常: %v", err)
	}
	if exists == 1 {
		return fmt.Errorf("验证码仍有效，请勿重复获取")
	}

	// 生成验证码
	code := GenerateEmailCode()

	// 发送邮件
	body := fmt.Sprintf(`<div style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
		<h2 style="color: #333;">登录验证码</h2>
		<p style="font-size: 16px; color: #666;">您的验证码是：<strong style="color: #007bff; font-size: 20px;">%s</strong></p>
		<p style="font-size: 12px; color: #999;">验证码有效期5分钟，请尽快使用</p>
	</div>`, code)
	if err := sendEmail(email, "登录验证码", body); err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}

	// 存入 Redis (Pipeline 保证原子性)
	pipe := RedisClient.Pipeline()
	pipe.Set(ctx, codeKey, code, 5*time.Minute)
	pipe.Set(ctx, limitKey, 1, 60*time.Second) // 60秒防刷
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis 存储失败: %v", err)
	}

	return nil
}

// VerifyAndConsumeCode 校验并消费验证码（原子性操作）
func VerifyAndConsumeCode(ctx context.Context, email, inputCode string) bool {
	redisKey := emailCodePrefix + email

	// 使用Lua脚本保证原子性（Get + Del 原子操作）
	script := `
		local storedCode = redis.call('get', KEYS[1])
		if storedCode == ARGV[1] then
			redis.call('del', KEYS[1])
			return 1
		end
		return 0
	`
	result, err := RedisClient.Eval(ctx, script, []string{redisKey}, inputCode).Result()
	if err != nil {
		return false
	}
	return result.(int64) == 1
}

// GenerateEmailCode 生成6位随机验证码
func GenerateEmailCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2]))
}
func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("EMAIL_SMTP_HOST")
	smtpPort := os.Getenv("EMAIL_SMTP_PORT")
	emailFrom := os.Getenv("EMAIL_FROM")
	emailPassword := os.Getenv("EMAIL_PASSWORD")

	auth := smtp.PlainAuth("", emailFrom, emailPassword, smtpHost)

	// 构造邮件内容
	msg := []byte("From: " + emailFrom + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// 根据端口选择 TLS 配置
	if smtpPort == "465" {
		// SSL 直连
		tlsConfig := &tls.Config{ServerName: smtpHost}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(emailFrom); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(msg)
		if err != nil {
			return err
		}
		return w.Close()
	} else {
		// 587 STARTTLS
		return smtp.SendMail(addr, auth, emailFrom, []string{to}, msg)
	}
}
