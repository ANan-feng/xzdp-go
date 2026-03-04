package utils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"xzdp-go/model"
)

// SendEmailCode 发送邮箱验证码
func SendEmailCode(email, code string) error {
	// 1. 读取邮箱配置
	smtpHost := os.Getenv("EMAIL_SMTP_HOST")
	smtpPortStr := os.Getenv("EMAIL_SMTP_PORT")
	smtpPort, err := strconv.Atoi(smtpPortStr) // 修复：捕获转换错误
	if err != nil {
		return fmt.Errorf("invalid smtp port: %v", err)
	}
	emailUser := os.Getenv("EMAIL_USER")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	emailFrom := os.Getenv("EMAIL_FROM")
	emailSubject := os.Getenv("EMAIL_SUBJECT")

	// 校验配置非空
	if smtpHost == "" || emailUser == "" || emailPassword == "" || emailFrom == "" {
		return fmt.Errorf("email config is incomplete")
	}

	// 2. 构建邮件内容
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("黑马点评<%s>", emailFrom)
	headers["To"] = email
	headers["Subject"] = emailSubject
	headers["Content-Type"] = "text/html; charset=UTF-8"

	body := fmt.Sprintf(`
		<div style="width: 300px; margin: 0 auto; padding: 20px; border: 1px solid #eee;">
			<h3>黑马点评登录验证码</h3>
			<p>你的登录验证码是：<span style="color: #ff4d4f; font-size: 18px; font-weight: bold;">%s</span></p>
			<p>验证码有效期5分钟，请及时使用，请勿泄露给他人！</p>
		</div>
	`, code)

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 3. 配置TLS
	tlsConfig := &tls.Config{
		ServerName:         smtpHost,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false, // 测试环境可临时改为true
	}

	// 4. 连接SMTP服务器（区分端口处理）
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	var client *smtp.Client
	if smtpPort == 465 {
		// 465端口：直连TLS
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial (465) failed: %v", err)
		}
		client, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("create smtp client (465) failed: %v", err)
		}
	} else if smtpPort == 587 {
		// 587端口：TCP + STARTTLS
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("tcp dial (587) failed: %v", err)
		}
		client, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("create smtp client (587) failed: %v", err)
		}
		// 升级TLS
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start tls failed: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported smtp port: %d (only 465/587 are supported)", smtpPort)
	}
	defer client.Close()

	// 5. 认证
	auth := smtp.PlainAuth("", emailUser, emailPassword, smtpHost)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth failed: %v", err)
	}

	// 6. 发送邮件
	if err = client.Mail(emailFrom); err != nil {
		return fmt.Errorf("smtp mail from failed: %v", err)
	}
	if err = client.Rcpt(email); err != nil {
		return fmt.Errorf("smtp rcpt to failed: %v", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %v", err)
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("smtp write message failed: %v", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close writer failed: %v", err)
	}
	client.Quit()

	// 7. 保存验证码到数据库
	expireTime := time.Now().Add(5 * time.Minute)
	dbResult := DB.Exec(
		"INSERT INTO email_code (email, code, expire_time) VALUES (?, ?, ?)",
		email, code, expireTime,
	)
	if dbResult.Error != nil {
		return fmt.Errorf("save email code to db failed: %v", dbResult.Error)
	}

	return nil
}

// 其他函数（GenerateEmailCode/VerifyEmailCode）保持不变
// GenerateEmailCode 生成6位随机验证码
func GenerateEmailCode() string {
	randNum := time.Now().UnixNano() % 1000000
	return fmt.Sprintf("%06d", randNum)
}

// VerifyEmailCode 验证邮箱验证码是否有效
func VerifyEmailCode(email, code string) bool {
	var emailCode model.EmailCode
	// 查询未过期的验证码
	result := DB.Where("email = ? AND code = ? AND expire_time > ?", email, code, time.Now()).First(&emailCode)
	// 正确判断：Error为nil表示查询成功（验证码有效）
	return result.Error == nil
}
