package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetProjectRoot 获取项目根目录（兼容不同运行环境）
func GetProjectRoot() string {
	// 获取当前文件的绝对路径
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get project root")
	}
	// 向上回退到项目根目录（utils/ → 根目录）
	root := filepath.Dir(filepath.Dir(file))
	return root
}

// LoadLuaScript 加载Lua脚本文件
// scriptPath: 相对于项目根目录的路径（如 "scripts/seckill/seckill_pre_check.lua"）
func LoadLuaScript(scriptPath string) (string, error) {
	// 拼接绝对路径
	fullPath := filepath.Join(GetProjectRoot(), scriptPath)
	// 读取文件内容
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
