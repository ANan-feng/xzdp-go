package utils

import (
	"fmt"
	"sync"
)

// 全局脚本缓存
var (
	scriptCache = make(map[string]string)
	cacheMutex  sync.RWMutex
)

// InitScriptCache 初始化脚本缓存（项目启动时调用）
func InitScriptCache() error {
	// 加载所有秒杀相关脚本
	seckillScripts := []string{
		SeckillPreCheckScriptPath,
		// 可添加其他脚本路径
	}
	for _, path := range seckillScripts {
		content, err := LoadLuaScript(path)
		if err != nil {
			return err
		}
		cacheMutex.Lock()
		scriptCache[path] = content
		cacheMutex.Unlock()
	}
	return nil
}

// GetCachedLuaScript 获取缓存的Lua脚本
func GetCachedLuaScript(path string) (string, error) {
	cacheMutex.RLock()
	content, ok := scriptCache[path]
	cacheMutex.RUnlock()
	if !ok {
		return "", fmt.Errorf("script %s not in cache", path)
	}
	return content, nil
}
