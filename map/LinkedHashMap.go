package _map

import (
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"

	jsonv2 "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type Entry[K comparable, V any] struct {
	Key K
	Val V
}

type Option[K comparable, V any] func(*LinkedHashMap[K, V])

// WithAccessOrder 设置访问顺序模式（LRU）
func WithAccessOrder[K comparable, V any]() Option[K, V] {
	return func(m *LinkedHashMap[K, V]) {
		m.accessOrder = true
	}
}

// Node 双向链表节点（对应 LinkedHashMap 内部 Entry）
type Node[K comparable, V any] struct {
	key  K
	val  V
	prev *Node[K, V]
	next *Node[K, V]
}

// LinkedHashMap 复刻 Java LinkedHashMap
type LinkedHashMap[K comparable, V any] struct {
	data        map[K]*Node[K, V]
	head, tail  *Node[K, V]
	size        int
	accessOrder bool
}

// NewLinkedHashMap 默认插入顺序（等价 new LinkedHashMap<>()）
//
// 用例:
// _map.NewLinkedHashMap(
//
//		[]_map.Entry[string, int]{
//			{"apple", 1},
//			{"banana", 2},
//			{"cherry", 3},
//		},
//	)
func NewLinkedHashMap[K comparable, V any](entries []Entry[K, V], opts ...Option[K, V]) *LinkedHashMap[K, V] {
	m := &LinkedHashMap[K, V]{
		data:        make(map[K]*Node[K, V]),
		accessOrder: false,
	}

	for _, entry := range entries {
		m.Put(entry.Key, entry.Val)
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Put 对应 Java put()，JS set()
func (m *LinkedHashMap[K, V]) Put(key K, val V) V {
	if node, ok := m.data[key]; ok {
		oldVal := node.val
		node.val = val
		if m.accessOrder {
			m.moveToTail(node)
		}
		return oldVal
	}

	newNode := &Node[K, V]{key: key, val: val}
	m.data[key] = newNode
	m.addToTail(newNode)
	m.size++
	return val
}

// Get 对应 Java get()，JS get()
func (m *LinkedHashMap[K, V]) Get(key K) (V, bool) {
	node, ok := m.data[key]
	if !ok {
		var zero V
		return zero, false
	}

	if m.accessOrder {
		m.moveToTail(node)
	}
	return node.val, true
}

// Remove 对应 Java remove()，JS delete()
func (m *LinkedHashMap[K, V]) Remove(key K) V {
	node, ok := m.data[key]
	if !ok {
		var zero V
		return zero
	}
	delete(m.data, key)
	m.removeNode(node)
	m.size--
	return node.val
}

// RemoveMatch 仅在键存在且当前值等于oldVal时删除
func (m *LinkedHashMap[K, V]) RemoveMatch(key K, oldVal V) bool {
	node, ok := m.data[key]
	if !ok {
		return false
	}

	// 使用反射比较值是否相等
	if reflect.DeepEqual(node.val, oldVal) {
		delete(m.data, key)
		m.removeNode(node)
		m.size--
		return true
	}
	return false
}

// Size 对应 size()，JS size
func (m *LinkedHashMap[K, V]) Size() int {
	return m.size
}

// IsEmpty 判断是否为空
func (m *LinkedHashMap[K, V]) IsEmpty() bool {
	return m.size == 0
}

// Clear 清空
func (m *LinkedHashMap[K, V]) Clear() {
	m.data = make(map[K]*Node[K, V])
	m.head, m.tail = nil, nil
	m.size = 0
}

// PutAll 将另一个 Map 中的所有键值对放入本映射
func (m *LinkedHashMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq2() {
		m.Put(k, v)
	}
}

// GetOrDefault 如果键不存在则返回默认值
func (m *LinkedHashMap[K, V]) GetOrDefault(key K, def V) V {
	v, ok := m.Get(key)
	if !ok {
		return def
	}
	return v
}

// PutIfAbsent 仅在键不存在时放入新值，返回旧值（若存在）
func (m *LinkedHashMap[K, V]) PutIfAbsent(key K, val V) V {
	if node, ok := m.data[key]; ok {
		return node.val
	}
	m.Put(key, val)
	var zero V
	return zero
}

// Replace 仅在键存在时替换为新值，返回旧值和是否成功
func (m *LinkedHashMap[K, V]) Replace(key K, newVal V) (V, bool) {
	node, ok := m.data[key]
	if !ok {
		var zero V
		return zero, false
	}
	oldVal := node.val
	node.val = newVal
	if m.accessOrder {
		m.moveToTail(node)
	}
	return oldVal, true
}

// ReplaceMatch 仅在键存在且当前值等于old时替换为new
func (m *LinkedHashMap[K, V]) ReplaceMatch(key K, old, new V) bool {
	node, ok := m.data[key]
	if !ok {
		return false
	}

	// 使用反射比较值是否相等
	if reflect.DeepEqual(node.val, old) {
		node.val = new
		if m.accessOrder {
			m.moveToTail(node)
		}
		return true
	}
	return false
}

// ContainsKey 对应 Java containsKey()，JS has()
func (m *LinkedHashMap[K, V]) ContainsKey(key K) bool {
	_, ok := m.data[key]
	return ok
}

// Seq2 返回一个迭代器，支持 range 遍历（Go 1.23+ Seq2）
func (m *LinkedHashMap[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		cur := m.head
		for cur != nil {
			if !yield(cur.key, cur.val) {
				return
			}
			cur = cur.next
		}
	}
}

// Seq 返回键的迭代器（Go 1.23+ Seq）
func (m *LinkedHashMap[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		cur := m.head
		for cur != nil {
			if !yield(cur.val) {
				return
			}
			cur = cur.next
		}
	}
}
func (m *LinkedHashMap[K, V]) EntrySet() iter.Seq2[K, V] {
	return m.Seq2()
}

// KeySet 返回键的迭代器（Go 1.23+ Seq）
func (m *LinkedHashMap[K, V]) KeySet() iter.Seq[K] {
	return func(yield func(K) bool) {
		cur := m.head
		for cur != nil {
			if !yield(cur.key) {
				return
			}
			cur = cur.next
		}
	}
}

// Values 返回值的迭代器（Go 1.23+ Seq）
func (m *LinkedHashMap[K, V]) Values() iter.Seq[V] {
	return m.Seq()
}

// 内部：尾部添加节点
func (m *LinkedHashMap[K, V]) addToTail(node *Node[K, V]) {
	if m.tail == nil {
		m.head = node
		m.tail = node
		return
	}
	node.prev = m.tail
	m.tail.next = node
	m.tail = node
}

// 内部：移除节点
func (m *LinkedHashMap[K, V]) removeNode(node *Node[K, V]) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		m.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		m.tail = node.prev
	}
	node.prev, node.next = nil, nil
}

// 内部：移到尾部（LRU访问顺序）
func (m *LinkedHashMap[K, V]) moveToTail(node *Node[K, V]) {
	if node == m.tail {
		return
	}
	m.removeNode(node)
	m.addToTail(node)
}

// MarshalJSON 将 LinkedHashMap 序列化为 JSON 字节数组，保持插入顺序
func (m *LinkedHashMap[K, V]) MarshalJSON() ([]byte, error) {
	if m.data == nil || m.size == 0 {
		return []byte("{}"), nil
	}

	var sb strings.Builder
	sb.WriteByte('{')

	first := true
	for key, val := range m.Seq2() {
		if !first {
			sb.WriteByte(',')
		}
		first = false

		keyBytes, err := jsonv2.Marshal(key)
		if err != nil {
			return nil, err
		}
		keyStr := string(keyBytes)
		if len(keyStr) == 0 || keyStr[0] != '"' {
			sb.WriteByte('"')
			sb.Write(keyBytes)
			sb.WriteByte('"')
		} else {
			sb.Write(keyBytes)
		}

		sb.WriteByte(':')

		valBytes, err := jsonv2.Marshal(val)
		if err != nil {
			return nil, err
		}
		sb.Write(valBytes)
	}

	sb.WriteByte('}')
	return []byte(sb.String()), nil
}

// UnmarshalJSON 将 JSON 字节数组反序列化为 Go 对象
func (m *LinkedHashMap[K, V]) UnmarshalJSON(data []byte) error {
	if m.data == nil {
		m.data = make(map[K]*Node[K, V])
	} else {
		m.Clear()
	}

	decoder := jsontext.NewDecoder(strings.NewReader(string(data)))

	token, err := decoder.ReadToken()
	if err != nil {
		return err
	}
	if token.Kind() != jsontext.KindBeginObject {
		return fmt.Errorf("expected object start at offset %d", decoder.InputOffset())
	}

	for {

		token, err := decoder.ReadToken()
		if err != nil {
			return err
		}

		if token.Kind() == jsontext.KindEndObject {
			break
		}

		var key K
		keyStr := token.String()

		keyVal, err := convertKey[K](keyStr)
		if err != nil {
			return fmt.Errorf("cannot convert key '%s' to type %v: %w", keyStr, reflect.TypeOf(key), err)
		}
		key = keyVal

		var val V
		if err := jsonv2.UnmarshalDecode(decoder, &val); err != nil {
			return err
		}

		m.Put(key, val)
	}

	return nil
}

// convertKey 将字符串键转换为目标类型
func convertKey[K comparable](keyStr string) (K, error) {
	var zero K

	var target K
	targetType := reflect.TypeOf(target)

	switch targetType.Kind() {
	case reflect.String:
		if val, ok := interface{}(keyStr).(K); ok {
			return val, nil
		}
		return zero, fmt.Errorf("failed to convert string to string type")

	case reflect.Int:
		val, err := strconv.ParseInt(keyStr, 10, strconv.IntSize)
		if err != nil {
			return zero, err
		}
		return interface{}(int(val)).(K), nil

	case reflect.Int8:
		val, err := strconv.ParseInt(keyStr, 10, 8)
		if err != nil {
			return zero, err
		}
		return interface{}(int8(val)).(K), nil

	case reflect.Int16:
		val, err := strconv.ParseInt(keyStr, 10, 16)
		if err != nil {
			return zero, err
		}
		return interface{}(int16(val)).(K), nil

	case reflect.Int32:
		val, err := strconv.ParseInt(keyStr, 10, 32)
		if err != nil {
			return zero, err
		}
		return interface{}(int32(val)).(K), nil

	case reflect.Int64:
		val, err := strconv.ParseInt(keyStr, 10, 64)
		if err != nil {
			return zero, err
		}
		return interface{}(val).(K), nil

	case reflect.Uint:
		val, err := strconv.ParseUint(keyStr, 10, strconv.IntSize)
		if err != nil {
			return zero, err
		}
		return interface{}(uint(val)).(K), nil

	case reflect.Uint8:
		val, err := strconv.ParseUint(keyStr, 10, 8)
		if err != nil {
			return zero, err
		}
		return interface{}(uint8(val)).(K), nil

	case reflect.Uint16:
		val, err := strconv.ParseUint(keyStr, 10, 16)
		if err != nil {
			return zero, err
		}
		return interface{}(uint16(val)).(K), nil

	case reflect.Uint32:
		val, err := strconv.ParseUint(keyStr, 10, 32)
		if err != nil {
			return zero, err
		}
		return interface{}(uint32(val)).(K), nil

	case reflect.Uint64:
		val, err := strconv.ParseUint(keyStr, 10, 64)
		if err != nil {
			return zero, err
		}
		return interface{}(val).(K), nil

	case reflect.Float32:
		val, err := strconv.ParseFloat(keyStr, 32)
		if err != nil {
			return zero, err
		}
		return interface{}(float32(val)).(K), nil

	case reflect.Float64:
		val, err := strconv.ParseFloat(keyStr, 64)
		if err != nil {
			return zero, err
		}
		return interface{}(val).(K), nil

	case reflect.Bool:
		val, err := strconv.ParseBool(keyStr)
		if err != nil {
			return zero, err
		}
		return interface{}(val).(K), nil

	default:
		return zero, fmt.Errorf("unsupported key type: %v", targetType)
	}
}
