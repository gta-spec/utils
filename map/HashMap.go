package _map

import (
	"iter"
	"reflect"
)

// Map 定义哈希映射的完整行为，Go标准命名
type Map[K comparable, V any] interface {
	Put(k K, v V) V
	Get(k K) (V, bool)
	Remove(k K) V
	RemoveMatch(k K, oldVal V) bool

	Size() int
	IsEmpty() bool
	Clear()
	PutAll(other Map[K, V])

	// GetOrDefault Java8+ 便捷方法
	GetOrDefault(k K, def V) V
	PutIfAbsent(k K, v V) V
	Replace(k K, newVal V) (V, bool)
	ReplaceMatch(k K, old, new V) bool

	// Seq2 Go 1.23+ 标准迭代器（替代 entrySet/keySet/values）返回所有键值对的集合,同 EntrySet()
	Seq2() iter.Seq2[K, V]
	// Seq 返回所有值的集合,同 Values()
	Seq() iter.Seq[V]
	// Values 返回所有值的集合
	Values() iter.Seq[V]
	// KeySet 返回所有键的集合
	KeySet() iter.Seq[K]
	// EntrySet 返回所有键值对的集合
	EntrySet() iter.Seq2[K, V]
}

// HashMap 基于 Go 原生 map 的实现
type HashMap[K comparable, V any] map[K]V

// NewHashMap 创建一个新的 HashMap
func NewHashMap[K comparable, V any](entries []Entry[K, V]) HashMap[K, V] {
	m := make(HashMap[K, V])
	for _, entry := range entries {
		m.Put(entry.Key, entry.Val)
	}
	return m
}

// Put 插入键值对，返回旧值
func (m HashMap[K, V]) Put(k K, v V) V {
	oldVal, ok := m[k]
	m[k] = v
	if ok {
		return oldVal
	}
	return v
}

// Get 获取值
func (m HashMap[K, V]) Get(k K) (V, bool) {
	v, ok := map[K]V(m)[k]
	return v, ok
}

// Remove 删除键并返回被删除的值
func (m HashMap[K, V]) Remove(k K) V {
	v, ok := m[k]
	if ok {
		delete(m, k)
		return v
	}
	var zero V
	return zero
}

// RemoveMatch 仅在值匹配时删除
func (m HashMap[K, V]) RemoveMatch(k K, oldVal V) bool {
	currentVal, ok := m[k]
	if !ok {
		return false
	}

	// 使用反射比较值
	if reflect.DeepEqual(currentVal, oldVal) {
		delete(m, k)
		return true
	}
	return false
}

// Size 返回映射大小
func (m HashMap[K, V]) Size() int {
	return len(m)
}

// IsEmpty 判断是否为空
func (m HashMap[K, V]) IsEmpty() bool {
	return len(m) == 0
}

// Clear 清空映射
func (m HashMap[K, V]) Clear() {
	for k := range m {
		delete(m, k)
	}
}

// PutAll 将另一个 Map 中的所有键值对放入本映射
func (m HashMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq2() {
		m.Put(k, v)
	}
}

// GetOrDefault 如果键不存在则返回默认值
func (m HashMap[K, V]) GetOrDefault(k K, def V) V {
	v, ok := m.Get(k)
	if !ok {
		return def
	}
	return v
}

// PutIfAbsent 仅在键不存在时放入新值
func (m HashMap[K, V]) PutIfAbsent(k K, v V) V {
	if existingVal, ok := m[k]; ok {
		return existingVal
	}
	m[k] = v
	var zero V
	return zero
}

// Replace 仅在键存在时替换
func (m HashMap[K, V]) Replace(k K, newVal V) (V, bool) {
	oldVal, ok := m[k]
	if !ok {
		var zero V
		return zero, false
	}
	m[k] = newVal
	return oldVal, true
}

// ReplaceMatch 仅在键存在且值匹配时替换
func (m HashMap[K, V]) ReplaceMatch(k K, old, new V) bool {
	currentVal, ok := m[k]
	if !ok {
		return false
	}

	if reflect.DeepEqual(currentVal, old) {
		m[k] = new
		return true
	}
	return false
}

// Seq2 返回键值对的迭代器
func (m HashMap[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Seq 返回值的迭代器
func (m HashMap[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m {
			if !yield(v) {
				return
			}
		}
	}
}
func (m HashMap[K, V]) EntrySet() iter.Seq2[K, V] {
	return m.Seq2()
}

// KeySet 返回键的迭代器
func (m HashMap[K, V]) KeySet() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// Values 返回值的迭代器
func (m HashMap[K, V]) Values() iter.Seq[V] {
	return m.Seq()
}
