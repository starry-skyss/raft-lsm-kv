package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

/*
Type (1 byte): 操作类型。例如 0 代表 Put，1 代表 Delete。
KeyLen (4 bytes): Key 的长度（用 uint32 表示）。
Key (KeyLen bytes): Key 的实际字符串数据。
ValLen (4 bytes): Value 的长度（仅对 Put 有效）。
Value (ValLen bytes): Value 的实际字符串数据。
*/
const (
	OpPut    byte = 0
	OpDelete byte = 1
)

type WAL struct {
	mu   sync.Mutex // 保护文件写入不发生错乱
	file *os.File
}

// OpenWAL 打开或创建 WAL 文件
func OpenWAL(filePath string) (*WAL, error) {
	// os.O_CREATE: 如果文件不存在，操作系统会自动帮你创建它！
	// os.O_APPEND: 无论文件里原来有什么，每次写入都强制追加到文件最末尾！
	// os.O_RDWR:   以读写模式打开（因为我们启动时需要读它来重放，运行时需要写它）
	// 0644:        这是 Linux 的文件权限（拥有者可读写，其他人可读）
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{file: f}, nil
}

func (w *WAL) AppendPut(key, val string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	write := bufio.NewWriter(w.file)
	binary.Write(write, binary.BigEndian, OpPut)
	binary.Write(write, binary.BigEndian, uint32(len(key)))
	write.WriteString(key)
	binary.Write(write, binary.BigEndian, uint32(len(val)))
	write.WriteString(val)
	write.Flush()
	w.file.Sync()

	return nil
}

func (w *WAL) AppendDelete(key string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	write := bufio.NewWriter(w.file)
	binary.Write(write, binary.BigEndian, OpDelete)
	binary.Write(write, binary.BigEndian, uint32(len(key)))
	write.WriteString(key)
	write.Flush()
	w.file.Sync()
	return nil
}

// Close 关闭 WAL 文件
func (w *WAL) Close() error {
	if w.file != nil {
		// 关闭前最好先 Sync 一次
		w.file.Sync()
		return w.file.Close()
	}
	return nil
}

func (w *WAL) Replay(onPut func(key, val string), onDelete func(key string)) error {
	// 1. 将文件指针移动到文件头部
	_, err := w.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// 2. 循环读取并解析（这里省略了具体的二进制解包逻辑）
	for {
		var klen, vlen uint32
		var op byte
		err := binary.Read(w.file, binary.BigEndian, &op)
		if err == io.EOF {
			// 正常结束，跳出循环
			break
		}
		if err != nil {
			// 真正的异常（比如读取中途断网、文件权限、硬件故障等）
			fmt.Println(err)
			break
		}

		binary.Read(w.file, binary.BigEndian, &klen)
		keyBuf := make([]byte, klen)
		if _, err := io.ReadFull(w.file, keyBuf); err != nil {
			return err // 处理读取失败
		}
		if op == OpPut {
			binary.Read(w.file, binary.BigEndian, &vlen)
			valBuf := make([]byte, vlen)
			if _, err := io.ReadFull(w.file, valBuf); err != nil {
				return err
			}
			onPut(string(keyBuf), string(valBuf))
		} else if op == OpDelete {
			onDelete(string(keyBuf))
		}
	}

	// 3. 恢复完成后，把文件指针重新移回末尾，准备接受后续的 Append
	_, err = w.file.Seek(0, io.SeekEnd)
	return err
}
