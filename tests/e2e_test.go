package tests

// 文件路径: tests/e2e_test.go
// 运行命令: go test -v ./tests

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_EndToEnd(t *testing.T) {
	// 1. 编译可执行文件
	binaryName := "test_kvnode"
	build := exec.Command("go", "build", "-o", binaryName, "../cmd/kvnode/main.go")
	if err := build.Run(); err != nil {
		t.Fatalf("编译失败: %v", err)
	}
	defer os.Remove(binaryName) // 测试完删掉二进制文件
	defer os.RemoveAll("data")  // 测试完清理数据目录

	// 2. 准备模拟的用户输入
	inputCommands := `put E2E_KEY E2E_VALUE
get E2E_KEY
exit
`
	// 3. 启动子进程
	cmd := exec.Command("./" + binaryName)
	cmd.Stdin = strings.NewReader(inputCommands) // 将输入重定向给子进程

	var out bytes.Buffer
	cmd.Stdout = &out // 捕获子进程的输出

	// 4. 运行并等待结束
	if err := cmd.Run(); err != nil {
		t.Fatalf("程序运行异常: %v", err)
	}

	// 5. 验证输出结果
	output := out.String()
	if !strings.Contains(output, "写入成功") {
		t.Errorf("期望看到写入成功，实际输出:\n%s", output)
	}
	if !strings.Contains(output, "E2E_VALUE") {
		t.Errorf("期望 Get 到 E2E_VALUE，实际输出:\n%s", output)
	}
}
