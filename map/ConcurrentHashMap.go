package _map

import (
	"iter"
	"sync"
)

// ConcurrentHashMap 基于 sync.Map 的并发安全映射实现
type ConcurrentHashMap[K comparable, V any] struct {
	data sync.Map
	size int
	mu   sync.RWMutex // 仅用于维护 size
}

// NewConcurrentHashMap 创建一个新的 ConcurrentHashMap
func NewConcurrentHashMap[K comparable, V any](entries []Entry[K, V]) *ConcurrentHashMap[K, V] {
	m := &ConcurrentHashMap[K, V]{
		size: 0,
	}

	for _, entry := range entries {
		m.Put(entry.Key, entry.Val)
	}

	return m
}

// Put 插入键值对，返回旧值
func (m *ConcurrentHashMap[K, V]) Put(k K, v V) V {
	actual, loaded := m.data.Swap(k, v)

	m.mu.Lock()
	if !loaded {
		m.size++
	}
	m.mu.Unlock()

	if loaded {
		return actual.(V)
	}
	return v
}

// Get 获取值
func (m *ConcurrentHashMap[K, V]) Get(k K) (V, bool) {
	val, ok := m.data.Load(k)
	if !ok {
		var zero V
		return zero, false
	}
	return val.(V), true
}

// Remove 删除键并返回被删除的值
func (m *ConcurrentHashMap[K, V]) Remove(k K) V {
	val, loaded := m.data.LoadAndDelete(k)

	if loaded {
		m.mu.Lock()
		m.size--
		m.mu.Unlock()
		return val.(V)
	}

	var zero V
	return zero
}

// RemoveMatch 仅在值匹配时删除
func (m *ConcurrentHashMap[K, V]) RemoveMatch(k K, oldVal V) bool {
	deleted := m.data.CompareAndDelete(k, oldVal)

	if deleted {
		m.mu.Lock()
		m.size--
		m.mu.Unlock()
		return true
	}

	return false
}

// Size 返回映射大小
func (m *ConcurrentHashMap[K, V]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.size
}

// IsEmpty 判断是否为空
func (m *ConcurrentHashMap[K, V]) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 清空映射
func (m *ConcurrentHashMap[K, V]) Clear() {
	m.data.Range(func(key, value interface{}) bool {
		m.data.Delete(key)
		return true
	})

	m.mu.Lock()
	m.size = 0
	m.mu.Unlock()
}

// PutAll 将另一个 Map 中的所有键值对放入本映射
func (m *ConcurrentHashMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq2() {
		m.Put(k, v)
	}
}

// GetOrDefault 如果键不存在则返回默认值
func (m *ConcurrentHashMap[K, V]) GetOrDefault(k K, def V) V {
	v, ok := m.Get(k)
	if !ok {
		return def
	}
	return v
}

// PutIfAbsent 仅在键不存在时放入新值
func (m *ConcurrentHashMap[K, V]) PutIfAbsent(k K, v V) V {
	actual, loaded := m.data.LoadOrStore(k, v)

	if !loaded {
		m.mu.Lock()
		m.size++
		m.mu.Unlock()
		var zero V
		return zero
	}

	return actual.(V)
}

// Replace 仅在键存在时替换
func (m *ConcurrentHashMap[K, V]) Replace(k K, newVal V) (V, bool) {
	actual, loaded := m.data.Swap(k, newVal)

	if loaded {
		return actual.(V), true
	}

	var zero V
	return zero, false
}

// ReplaceMatch 仅在键存在且值匹配时替换
func (m *ConcurrentHashMap[K, V]) ReplaceMatch(k K, old, new V) bool {
	return m.data.CompareAndSwap(k, old, new)
}

// Seq2 返回键值对的迭代器
func (m *ConcurrentHashMap[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.data.Range(func(key, value interface{}) bool {
			return yield(key.(K), value.(V))
		})
	}
}

// Seq 返回值的迭代器
func (m *ConcurrentHashMap[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.data.Range(func(key, value interface{}) bool {
			return yield(value.(V))
		})
	}
}
func (m *ConcurrentHashMap[K, V]) EntrySet() iter.Seq2[K, V] {
	return m.Seq2()
}

// KeySet 返回键的迭代器
func (m *ConcurrentHashMap[K, V]) KeySet() iter.Seq[K] {
	return func(yield func(K) bool) {
		m.data.Range(func(key, value interface{}) bool {
			return yield(key.(K))
		})
	}
}

// Values 返回值的迭代器
func (m *ConcurrentHashMap[K, V]) Values() iter.Seq[V] {
	return m.Seq()
}
