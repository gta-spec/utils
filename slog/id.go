package _slog

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	traceSequence uint32 = 1000
	localIP       string
)

// GenerateTraceId 生成TraceId: IP(8位) + 时间戳(13位) + 自增序列(4位) + 进程ID(5位)
// 示例: 0ad1348f1403169275002100356696
func GenerateTraceId() string {
	ipPart := localIP

	timestamp := time.Now().UnixMilli()
	timePart := fmt.Sprintf("%013d", timestamp)

	seq := atomic.AddUint32(&traceSequence, 1)
	if seq > 9000 {
		atomic.StoreUint32(&traceSequence, 1000)
		seq = 1000
	}
	seqPart := fmt.Sprintf("%04d", seq)

	pid := os.Getpid()
	pidPart := fmt.Sprintf("%05d", pid)

	return ipPart + timePart + seqPart + pidPart
}

// GenerateSpanId 生成SpanId，表示调用链路树中的位置
// 根节点: 0
// 第一层调用: 0.1, 0.2, 0.3...
// 第二层调用: 0.1.1, 0.1.2, 0.2.1...
func GenerateSpanId(parentSpanId string) string {
	if parentSpanId == "" {
		return "0"
	}

	if parentSpanId == "0" {
		return "0.1"
	}

	parts := strings.Split(parentSpanId, ".")

	lastPart := parts[len(parts)-1]
	num, err := strconv.Atoi(lastPart)
	if err != nil {
		num = 0
	}
	num++

	parts[len(parts)-1] = strconv.Itoa(num)
	return strings.Join(parts, ".")
}
