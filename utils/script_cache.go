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
	// 由于秒杀操作现在使用内联Lua脚本，不需要从文件加载
	// 可在此处加载其他脚本（如果有的话）
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
