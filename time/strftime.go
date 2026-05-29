package _time

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

var mapping = map[string]string{
	// 【时间类】
	"%tH": "15",
	"%tI": "03",
	"%tk": "15",
	"%tl": "3",
	"%tM": "04",
	"%tS": "05",
	"%tL": ".000",
	"%tN": ".000000000",
	"%tp": "pm",
	"%Tp": "PM",
	"%tz": "-0700",
	"%tZ": "MST",
	// %ts、%tQ 需特殊代码处理，无layout映射

	// 【日期类】
	"%tB": "January",
	"%tb": "Jan",
	"%th": "Jan",
	"%tA": "Monday",
	"%ta": "Mon",
	"%tC": "20",
	"%tY": "2006",
	"%ty": "06",
	"%tj": "002",
	"%tm": "01",
	"%td": "02",
	"%te": "2",

	// 【组合类（你重点要的6个）】
	"%tR": "15:04",
	"%tT": "15:04:05",
	"%tr": "03:04:05 PM",
	"%tD": "01/02/06",
	"%tF": "2006-01-02",
	"%tc": "Mon Jan 02 15:04:05 MST 2006",

	// ========== 二、基础 strftime 通用占位符（兼容混用场景） ==========
	"%Y": "2006", "%y": "06", "%C": "20", "%G": "2006", "%g": "06",
	"%m": "01", "%-m": "1", "%b": "Jan", "%B": "January",
	"%d": "02", "%-d": "2", "%e": "2", "%j": "002",
	"%a": "Mon", "%A": "Monday", "%w": "0", "%u": "1",
	"%U": "00", "%W": "00", "%V": "01",
	"%H": "15", "%k": "15", "%I": "03", "%l": "3",
	"%M": "04", "%S": "05", "%L": "000", "%f": "000000",
	"%p": "PM", "%P": "pm",
	"%z": "-0700", "%:z": "-07:00", "%::z": "-07:00:00", "%Z": "MST",
	"%F": "2006-01-02", "%T": "15:04:05", "%R": "15:04", "%D": "01/02/06",

	// ========== 三、特殊转义/分隔符 ==========
	"%%": "%",  // 输出%本身
	"%n": "\n", // 平台独立换行符（原文5）
	"%t": "\t", // 制表符
}

var (
	StrftimePattern *regexp.Regexp
	sortedKeys      []string     // 缓存排序后的 keys
	mu              sync.RWMutex // 保护并发访问
)

func init() {
	rebuildPattern()
}

func rebuildPattern() {
	// 从 mapping 的 key 生成正则表达式
	patterns := make([]string, 0, len(mapping))
	for k := range mapping {
		// 转义正则特殊字符（如 %:z 中的 :）
		escaped := regexp.QuoteMeta(k)
		patterns = append(patterns, escaped)
	}

	// 按长度降序排序，确保长匹配优先（如 %-m 在 %m 之前）
	sort.Slice(patterns, func(i, j int) bool {
		return len(patterns[i]) > len(patterns[j])
	})

	// 组合成正则表达式
	pattern := strings.Join(patterns, "|")
	StrftimePattern = regexp.MustCompile(pattern)

	// 初始化并缓存排序后的 keys
	sortedKeys = make([]string, 0, len(mapping))
	for k := range mapping {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return len(sortedKeys[i]) > len(sortedKeys[j])
	})
}

// AddPattern 动态追加mapping规则，并更新正则表达式
func AddPattern(patterns map[string]string) {
	mu.Lock()
	defer mu.Unlock()

	for k, v := range patterns {
		mapping[k] = v
	}

	rebuildPattern()
}

// IsStrftimeFormat 检查给定的格式字符串是否包含 strftime 风格的占位符
// 参数 layout: 待检查的格式字符串
// 返回值: 如果包含 strftime 占位符返回 true，否则返回 false
func IsStrftimeFormat(layout string) bool {
	return StrftimePattern.MatchString(layout)
}

// Strftime 将 strftime 风格的占位符转换为 Go 时间格式
// 参数 layout: 包含 strftime 占位符的格式字符串（如 "%Y-%m-%d %H:%M:%S"）
// 返回值: 转换后的 Go 时间格式字符串（如 "2006-01-02 15:04:05"）
//
// 支持的占位符包括：
//   - 时间类：%tH, %tM, %tS, %tL, %tN 等
//   - 日期类：%tY, %tm, %td, %tB, %tA 等
//   - 组合类：%tR, %tT, %tD, %tF, %tc 等
//   - 基础 strftime：%Y, %m, %d, %H, %M, %S 等
//   - 特殊字符：%% (百分号), %n (换行), %t (制表符)

func Strftime(layout string) string {
	// 快速检查：如果没有 %，直接返回
	if !strings.ContainsRune(layout, '%') {
		return layout
	}

	goLayout := layout

	// 使用缓存的 sortedKeys，避免重复创建和排序
	for _, k := range sortedKeys {
		// 先检查是否存在，避免不必要的 ReplaceAll
		if !strings.Contains(goLayout, k) {
			continue
		}

		// 执行替换
		goLayout = strings.ReplaceAll(goLayout, k, mapping[k])

		// 早期退出：如果已经没有 % 了，说明没有更多占位符需要替换
		if !strings.ContainsRune(goLayout, '%') {
			break
		}
	}

	return goLayout
}
