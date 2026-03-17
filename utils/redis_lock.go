package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLock 分布式锁结构体
type RedisLock struct {
	client     *redis.Client
	ctx        context.Context
	key        string        // 锁Key
	value      string        // 锁值（唯一标识，防止误删）
	expire     time.Duration // 锁过期时间
	renewChan  chan struct{} // 续约停止信号
	renewMutex sync.Mutex    // 续约互斥锁
	isLocked   bool          // 是否持有锁
}

// NewRedisLock 创建分布式锁实例
// key: 锁Key（如"seckill:lock:voucher:1:user:1001"）
// expire: 锁过期时间（建议30s，避免死锁）
func NewRedisLock(ctx context.Context, client *redis.Client, key string, expire time.Duration) *RedisLock {
	// 生成唯一value（防止不同进程误删对方的锁）
	b := make([]byte, 16)
	rand.Read(b)
	value := base64.URLEncoding.EncodeToString(b)

	return &RedisLock{
		client:    client,
		ctx:       ctx,
		key:       key,
		value:     value,
		expire:    expire,
		renewChan: make(chan struct{}),
	}
}

// Lock 获取分布式锁（带自动续约）
func (rl *RedisLock) Lock() (bool, error) {
	// SET NX EX：原子操作，不存在则设置+过期时间
	success, err := rl.client.SetNX(rl.ctx, rl.key, rl.value, rl.expire).Result()
	if err != nil {
		return false, fmt.Errorf("set nx failed: %v", err)
	}
	if !success {
		return false, nil // 锁已被占用
	}

	// 标记持有锁，并启动自动续约
	rl.isLocked = true
	go rl.autoRenew()
	return true, nil
}

// autoRenew 自动续约（每隔expire/3时间续期）
func (rl *RedisLock) autoRenew() {
	ticker := time.NewTicker(rl.expire / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.renewMutex.Lock()
			if !rl.isLocked {
				rl.renewMutex.Unlock()
				return
			}
			// 续期脚本：仅当锁存在且value匹配时才续期
			script := `
				if redis.call('get', KEYS[1]) == ARGV[1] then
					return redis.call('expire', KEYS[1], ARGV[2])
				else
					return 0
				end
			`
			_, err := rl.client.Eval(rl.ctx, script, []string{rl.key}, rl.value, int(rl.expire.Seconds())).Result()
			if err != nil {
				fmt.Printf("锁续约失败：key=%s, err=%v\n", rl.key, err)
			}
			rl.renewMutex.Unlock()
		case <-rl.renewChan:
			return // 收到停止信号，退出续约
		case <-rl.ctx.Done():
			return // 上下文取消，退出续约
		}
	}
}

// Unlock 释放分布式锁（仅释放自己持有的锁）
func (rl *RedisLock) Unlock() error {
	rl.renewMutex.Lock()
	defer rl.renewMutex.Unlock()

	if !rl.isLocked {
		return nil
	}

	// 释放脚本：原子操作，防止误删其他进程的锁
	script := `
		if redis.call('get', KEYS[1]) == ARGV[1] then
			return redis.call('del', KEYS[1])
		else
			return 0
		end
	`
	_, err := rl.client.Eval(rl.ctx, script, []string{rl.key}, rl.value).Result()
	if err != nil {
		return fmt.Errorf("unlock failed: %v", err)
	}

	// 停止续约，标记锁已释放
	close(rl.renewChan)
	rl.isLocked = false
	return nil
}

// IsLocked 判断是否持有锁
func (rl *RedisLock) IsLocked() bool {
	rl.renewMutex.Lock()
	defer rl.renewMutex.Unlock()
	return rl.isLocked
}
