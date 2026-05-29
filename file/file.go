package _file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// Watcher 监听文件变化
//
// 参数说明:
//   - filename: 要监听的文件或目录路径。如果文件不存在,会自动监听其父目录
//   - done: 用于接收变化通知的 channel,关闭此 channel 可停止监听
//   - events: 要监听的事件类型,使用位运算组合。0 表示监听所有事件
//   - fsnotify.Create - 文件/目录创建
//   - fsnotify.Write  - 文件内容写入
//   - fsnotify.Remove - 文件/目录删除
//   - fsnotify.Rename - 文件/目录重命名
//   - fsnotify.Chmod  - 权限修改
//
// 返回值:
//   - error: 初始化监听器时的错误(如路径无效、权限不足等)
//
// 使用示例:
//
//	// 示例1: 监听文件的所有变化
//	done := make(chan fsnotify.Event)
//	err := Watcher("config.yaml", done, 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	go func() {
//	    for range done {
//	        fmt.Println("文件发生变化")
//	    }
//	}()
//	// 停止监听: close(done)
//
//	// 示例2: 只监听写入和删除事件
//	done := make(chan fsnotify.Event)
//	err := Watcher("data.json", done, fsnotify.Write | fsnotify.Remove)
//
//	// 示例3: 监听整个目录
//	done := make(chan fsnotify.Event)
//	err := Watcher("./logs", done, fsnotify.Create)
//
// 注意事项:
//   - 监听会在后台 goroutine 中运行
//   - 当监听文件时,实际监听的是文件所在目录,会过滤出目标文件的事件
//   - 即使文件不存在也可以监听,文件创建时会收到通知
//   - 记得在不需要时关闭 done channel 以释放资源
func Watcher(filename string, done chan fsnotify.Event, events fsnotify.Op) error {
	if filename == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	var dir string
	var targetFile string

	info, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("获取文件信息失败: %v", err)
		}
		dir = filepath.Dir(filename)
		targetFile = filepath.Base(filename)
	} else {
		if info.IsDir() {
			dir = filename
			targetFile = ""
		} else {
			dir = filepath.Dir(filename)
			targetFile = filepath.Base(filename)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监听器失败: %v", err)
	}

	err = watcher.Add(dir)
	if err != nil {
		watcher.Close()
		return fmt.Errorf("添加监听目录失败: %v", err)
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if events != 0 && event.Op&events == 0 {
					continue
				}

				if targetFile != "" && event.Name != filepath.Join(dir, targetFile) {
					continue
				}

				select {
				case done <- event:
				case <-done:
					return
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("监听器错误: %v", err)
				return

			case <-done:
				return
			}
		}
	}()

	return nil
}
