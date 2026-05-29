package _map

import (
	"cmp"
	"iter"
	"reflect"
	"slices"
)

// TreeMap 基于排序键的有序映射实现
type TreeMap[K comparable, V any] struct {
	data   map[K]V
	keys   []K
	lessFn func(a, b K) bool
}

// TreeMapOption TreeMap 的配置选项
type TreeMapOption[K comparable, V any] func(*TreeMap[K, V])

// WithComparator 设置自定义比较器
func WithComparator[K comparable, V any](less func(a, b K) bool) TreeMapOption[K, V] {
	return func(m *TreeMap[K, V]) {
		m.lessFn = less
	}
}

// NewTreeMap 创建一个新的 TreeMap，使用默认比较器
func NewTreeMap[K cmp.Ordered, V any](entries []Entry[K, V], opts ...TreeMapOption[K, V]) *TreeMap[K, V] {
	m := &TreeMap[K, V]{
		data: make(map[K]V),
		lessFn: func(a, b K) bool {
			return a < b
		},
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(m)
	}

	for _, entry := range entries {
		m.Put(entry.Key, entry.Val)
	}

	return m
}

// Put 插入键值对，返回旧值
func (m *TreeMap[K, V]) Put(k K, v V) V {
	oldVal, ok := m.data[k]
	if !ok {
		// 新键，需要插入到排序位置
		m.data[k] = v
		idx, _ := slices.BinarySearchFunc(m.keys, k, func(a, b K) int {
			if m.lessFn(a, b) {
				return -1
			}
			if m.lessFn(b, a) {
				return 1
			}
			return 0
		})
		m.keys = slices.Insert(m.keys, idx, k)
		return v
	}

	m.data[k] = v
	return oldVal
}

// Get 获取值
func (m *TreeMap[K, V]) Get(k K) (V, bool) {
	v, ok := m.data[k]
	return v, ok
}

// Remove 删除键并返回被删除的值
func (m *TreeMap[K, V]) Remove(k K) V {
	v, ok := m.data[k]
	if !ok {
		var zero V
		return zero
	}

	delete(m.data, k)

	// 从排序键列表中移除
	idx, found := slices.BinarySearchFunc(m.keys, k, func(a, b K) int {
		if m.lessFn(a, b) {
			return -1
		}
		if m.lessFn(b, a) {
			return 1
		}
		return 0
	})
	if found {
		m.keys = slices.Delete(m.keys, idx, idx+1)
	}

	return v
}

// RemoveMatch 仅在值匹配时删除
func (m *TreeMap[K, V]) RemoveMatch(k K, oldVal V) bool {
	currentVal, ok := m.data[k]
	if !ok {
		return false
	}

	if reflect.DeepEqual(currentVal, oldVal) {
		m.Remove(k)
		return true
	}
	return false
}

// Size 返回映射大小
func (m *TreeMap[K, V]) Size() int {
	return len(m.data)
}

// IsEmpty 判断是否为空
func (m *TreeMap[K, V]) IsEmpty() bool {
	return len(m.data) == 0
}

// Clear 清空映射
func (m *TreeMap[K, V]) Clear() {
	m.data = make(map[K]V)
	m.keys = nil
}

// PutAll 将另一个 Map 中的所有键值对放入本映射
func (m *TreeMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq2() {
		m.Put(k, v)
	}
}

// GetOrDefault 如果键不存在则返回默认值
func (m *TreeMap[K, V]) GetOrDefault(k K, def V) V {
	v, ok := m.Get(k)
	if !ok {
		return def
	}
	return v
}

// PutIfAbsent 仅在键不存在时放入新值
func (m *TreeMap[K, V]) PutIfAbsent(k K, v V) V {
	if existingVal, ok := m.data[k]; ok {
		return existingVal
	}
	m.Put(k, v)
	var zero V
	return zero
}

// Replace 仅在键存在时替换
func (m *TreeMap[K, V]) Replace(k K, newVal V) (V, bool) {
	oldVal, ok := m.data[k]
	if !ok {
		var zero V
		return zero, false
	}
	m.data[k] = newVal
	return oldVal, true
}

// ReplaceMatch 仅在键存在且值匹配时替换
func (m *TreeMap[K, V]) ReplaceMatch(k K, old, new V) bool {
	currentVal, ok := m.data[k]
	if !ok {
		return false
	}

	if reflect.DeepEqual(currentVal, old) {
		m.data[k] = new
		return true
	}
	return false
}

// FirstKey 返回最小的键
func (m *TreeMap[K, V]) FirstKey() (K, bool) {
	if len(m.keys) == 0 {
		var zero K
		return zero, false
	}
	return m.keys[0], true
}

// LastKey 返回最大的键
func (m *TreeMap[K, V]) LastKey() (K, bool) {
	if len(m.keys) == 0 {
		var zero K
		return zero, false
	}
	return m.keys[len(m.keys)-1], true
}

// Seq2 返回键值对的迭代器（按排序顺序）
func (m *TreeMap[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range m.keys {
			if !yield(k, m.data[k]) {
				return
			}
		}
	}
}

// Seq 返回值的迭代器（按排序顺序）
func (m *TreeMap[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, k := range m.keys {
			if !yield(m.data[k]) {
				return
			}
		}
	}
}
func (m *TreeMap[K, V]) EntrySet() iter.Seq2[K, V] {
	return m.Seq2()
}

// KeySet 返回键的迭代器（按排序顺序）
func (m *TreeMap[K, V]) KeySet() iter.Seq[K] {
	return func(yield func(K) bool) {
		for _, k := range m.keys {
			if !yield(k) {
				return
			}
		}
	}
}

// Values 返回值的迭代器（按排序顺序）
func (m *TreeMap[K, V]) Values() iter.Seq[V] {
	return m.Seq()
}
