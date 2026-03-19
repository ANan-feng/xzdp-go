package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// Kafka全局配置
var (
	KafkaWriter *kafka.Writer
	KafkaReader *kafka.Reader
)

// InitKafka 初始化Kafka生产者/消费者（添加默认值兜底）
func InitKafka() {
	// 从.env读取配置，添加默认值
	kafkaBrokers := getEnv("KAFKA_BROKERS", "127.0.0.1:9092") // 默认本地Kafka
	topic := getEnv("KAFKA_SECKILL_TOPIC", "seckill_topic")   // 默认秒杀主题
	groupId := getEnv("KAFKA_GROUP_ID", "seckill_group")      // 默认消费者组

	// 1. 初始化生产者（Writer）
	KafkaWriter = kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{kafkaBrokers},
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{}, // 分区负载均衡
		WriteTimeout: 5 * time.Second,     // 修正后的超时字段
	})

	// 2. 初始化消费者（Reader）
	KafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBrokers},
		Topic:    topic,   // 兜底后不为空
		GroupID:  groupId, // 消费者组ID
		MinBytes: 10e3,    // 10KB
		MaxBytes: 10e6,    // 10MB

		SessionTimeout:    60 * time.Second, // 增加会话超时（默认30秒）
		HeartbeatInterval: 20 * time.Second, // 心跳间隔（通常为 SessionTimeout 的1/3）
		MaxAttempts:       3,
		CommitInterval:    time.Second, // 每秒自动提交位移
	})

	fmt.Println("Kafka 初始化成功，topic=", topic, "brokers=", kafkaBrokers)
}

// 新增：获取环境变量，带默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// CloseKafka 关闭Kafka连接（程序退出时调用）
func CloseKafka() {
	if KafkaWriter != nil {
		KafkaWriter.Close()
	}
	if KafkaReader != nil {
		KafkaReader.Close()
	}
}

// SendSeckillMsg 发送秒杀请求到Kafka
// msg: 秒杀请求消息体（JSON字符串）
func SendSeckillMsg(ctx context.Context, msg []byte) error {
	err := KafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(time.Now().UnixNano(), 10)), // 消息Key（防重复）
		Value: msg,
	})
	if err != nil {
		return fmt.Errorf("发送秒杀消息失败: %v", err)
	}
	return nil
}
