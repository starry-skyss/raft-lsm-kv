package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"raft-lsm-kv/internal/lsm"
)

func main() {
	fmt.Println("=== start ===")

	// 1. 统一使用项目根目录下的 data/ 目录。
	rootDir := "."
	dataDir := filepath.Join(rootDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("无法创建数据目录: %v", err)
	}

	kv := lsm.NewKVStore(rootDir)

	// 4. 通过公开的 Len() 方法安全地获取数据量
	count := kv.Len()
	fmt.Printf("[系统] 数据恢复完成！当前包含 %d 条数据。\n", count)

	// 5. 启动简单的交互式命令行循环 (MVP 客户端)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nkv> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 按空格分割输入的命令
		parts := strings.Split(line, " ")
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "put":
			if len(parts) != 3 {
				fmt.Println("用法: put <key> <value>")
				continue
			}
			if err := kv.Put(parts[1], parts[2]); err != nil {
				fmt.Printf("写入失败: %v\n", err)
			} else {
				fmt.Println("写入成功")
			}

		case "get":
			if len(parts) != 2 {
				fmt.Println("用法: get <key>")
				continue
			}
			val, exists := kv.Get(parts[1])
			if exists {
				fmt.Printf("值: %s\n", val)
			} else {
				fmt.Println("错误: Key 不存在")
			}

		case "del":
			if len(parts) != 2 {
				fmt.Println("用法: del <key>")
				continue
			}
			if err := kv.Delete(parts[1]); err != nil {
				fmt.Printf("删除失败: %v\n", err)
			} else {
				fmt.Println("删除成功")
			}

		case "exit", "quit":
			fmt.Println("关闭系统")
			return

		default:
			fmt.Println("未知命令。支持的命令: put, get, del, exit")
		}
	}
}
