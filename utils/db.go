package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接（实习必备：封装数据库连接）
func InitDB() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// 拼接DSN（Data Source Name）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// 连接数据库
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 打印SQL日志（调试用）
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %v", err))
	}

	fmt.Println("Database connected successfully!")
}
