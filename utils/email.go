package utils

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	emailCodePrefix         = "xzdp:email:code:"
	emailSendHistoryPrefix  = "xzdp:email:send:history:"
	EmailCodeTTL            = 5 * time.Minute
	EmailSendWindow         = 24 * time.Hour
	EmailSendInterval       = 60 * time.Second
	EmailSendDailyLimit     = 10
	EmailSendHistoryTimeout = EmailSendWindow + time.Minute
)

func emailCodeKey(email string) string {
	return fmt.Sprintf("%s%s", emailCodePrefix, email)
}

func emailSendHistoryKey(email string) string {
	return fmt.Sprintf("%s%s", emailSendHistoryPrefix, email)
}

func ValidateEmailFormat(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("邮箱格式不正确")
	}
	return nil
}

func CheckEmailSendLimit(ctx context.Context, email string) error {
	key := emailSendHistoryKey(email)
	now := time.Now()
	windowStart := fmt.Sprintf("%d", now.Add(-EmailSendWindow).Unix())
	last60sStart := fmt.Sprintf("%d", now.Add(-EmailSendInterval).Unix())
	current := fmt.Sprintf("%d", now.Unix())

	if err := RedisClient.ZRemRangeByScore(ctx, key, "-inf", windowStart).Err(); err != nil {
		return fmt.Errorf("redis 异常: %v", err)
	}

	dailyCount, err := RedisClient.ZCount(ctx, key, windowStart, current).Result()
	if err != nil {
		return fmt.Errorf("redis 异常: %v", err)
	}
	if dailyCount >= EmailSendDailyLimit {
		return fmt.Errorf("今日发送次数超限")
	}

	recentCount, err := RedisClient.ZCount(ctx, key, last60sStart, current).Result()
	if err != nil {
		return fmt.Errorf("redis 异常: %v", err)
	}
	if recentCount > 0 {
		return fmt.Errorf("操作频繁，请60秒后重试")
	}

	return nil
}

func SaveEmailCode(ctx context.Context, email, code string) error {
	return RedisClient.Set(ctx, emailCodeKey(email), code, EmailCodeTTL).Err()
}

func DeleteEmailCode(ctx context.Context, email string) error {
	return RedisClient.Del(ctx, emailCodeKey(email)).Err()
}

func AddEmailSendRecord(ctx context.Context, email, member string) error {
	key := emailSendHistoryKey(email)
	if err := RedisClient.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().Unix()), Member: member}).Err(); err != nil {
		return err
	}
	return RedisClient.Expire(ctx, key, EmailSendHistoryTimeout).Err()
}

func RemoveEmailSendRecord(ctx context.Context, email, member string) error {
	return RedisClient.ZRem(ctx, emailSendHistoryKey(email), member).Err()
}

func VerifyAndConsumeCode(ctx context.Context, email, inputCode string) bool {
	redisKey := emailCodeKey(email)

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

func GenerateEmailCode() string {
	max := big.NewInt(1000000)
	n, err := cryptorand.Int(cryptorand.Reader, max)
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func SendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("EMAIL_SMTP_HOST")
	smtpPort := os.Getenv("EMAIL_SMTP_PORT")
	emailFrom := os.Getenv("EMAIL_FROM")
	emailPassword := os.Getenv("EMAIL_PASSWORD")

	auth := smtp.PlainAuth("", emailFrom, emailPassword, smtpHost)

	msg := []byte("From: " + emailFrom + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	if smtpPort == "465" {
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
	}

	return smtp.SendMail(addr, auth, emailFrom, []string{to}, msg)
}
