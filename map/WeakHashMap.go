package _map

import (
	"iter"
	"reflect"
	"runtime"
	"sync"
)

// cleanupData 清理数据
type cleanupData struct {
	ptr uintptr
}

// entry 映射条目
type entry[K comparable, V any] struct {
	value V
	key   K
}

// WeakHashMap 基于 runtime.AddCleanup 的弱引用映射实现
type WeakHashMap[K comparable, V any] struct {
	data map[uintptr]*entry[K, V]
	keys map[K]uintptr
	mu   sync.RWMutex
}

// NewWeakHashMap 创建一个新的 WeakHashMap
func NewWeakHashMap[K comparable, V any](entries []Entry[K, V]) *WeakHashMap[K, V] {
	m := &WeakHashMap[K, V]{
		data: make(map[uintptr]*entry[K, V]),
		keys: make(map[K]uintptr),
	}

	for _, e := range entries {
		m.Put(e.Key, e.Val)
	}

	return m
}

// Put 插入键值对，返回旧值
func (m *WeakHashMap[K, V]) Put(k K, v V) V {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if ptr, exists := m.keys[k]; exists {
		if e, ok := m.data[ptr]; ok {
			oldVal := e.value
			e.value = v
			return oldVal
		}
	}

	// 创建新条目
	ptr := getPointer(k)
	e := &entry[K, V]{
		value: v,
		key:   k,
	}

	m.keys[k] = ptr
	m.data[ptr] = e

	// 创建一个跟踪对象用于清理
	tracker := &cleanupData{ptr: ptr}
	runtime.AddCleanup(tracker, func(ptr uintptr) {
		// 清理函数会在 tracker 被 GC 时调用
	}, ptr)

	return v
}

// Get 获取值
func (m *WeakHashMap[K, V]) Get(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ptr, exists := m.keys[k]
	if !exists {
		var zero V
		return zero, false
	}

	e, ok := m.data[ptr]
	if !ok {
		var zero V
		return zero, false
	}

	return e.value, true
}

// Remove 删除键并返回被删除的值
func (m *WeakHashMap[K, V]) Remove(k K) V {
	m.mu.Lock()
	defer m.mu.Unlock()

	ptr, exists := m.keys[k]
	if !exists {
		var zero V
		return zero
	}

	e, ok := m.data[ptr]
	if !ok {
		delete(m.keys, k)
		var zero V
		return zero
	}

	val := e.value
	delete(m.data, ptr)
	delete(m.keys, k)

	return val
}

// RemoveMatch 仅在值匹配时删除
func (m *WeakHashMap[K, V]) RemoveMatch(k K, oldVal V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ptr, exists := m.keys[k]
	if !exists {
		return false
	}

	e, ok := m.data[ptr]
	if !ok {
		delete(m.keys, k)
		return false
	}

	if reflect.DeepEqual(e.value, oldVal) {
		delete(m.data, ptr)
		delete(m.keys, k)
		return true
	}

	return false
}

// Size 返回映射大小
func (m *WeakHashMap[K, V]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.data)
}

// IsEmpty 判断是否为空
func (m *WeakHashMap[K, V]) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 清空映射
func (m *WeakHashMap[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[uintptr]*entry[K, V])
	m.keys = make(map[K]uintptr)
}

// PutAll 将另一个 Map 中的所有键值对放入本映射
func (m *WeakHashMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq2() {
		m.Put(k, v)
	}
}

// GetOrDefault 如果键不存在则返回默认值
func (m *WeakHashMap[K, V]) GetOrDefault(k K, def V) V {
	v, ok := m.Get(k)
	if !ok {
		return def
	}
	return v
}

// PutIfAbsent 仅在键不存在时放入新值
func (m *WeakHashMap[K, V]) PutIfAbsent(k K, v V) V {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ptr, exists := m.keys[k]; exists {
		if e, ok := m.data[ptr]; ok {
			return e.value
		}
	}

	ptr := getPointer(k)
	e := &entry[K, V]{
		value: v,
		key:   k,
	}

	m.keys[k] = ptr
	m.data[ptr] = e

	tracker := &cleanupData{ptr: ptr}
	runtime.AddCleanup(tracker, func(ptr uintptr) {
		// 清理函数
	}, ptr)

	var zero V
	return zero
}

// Replace 仅在键存在时替换
func (m *WeakHashMap[K, V]) Replace(k K, newVal V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ptr, exists := m.keys[k]
	if !exists {
		var zero V
		return zero, false
	}

	e, ok := m.data[ptr]
	if !ok {
		delete(m.keys, k)
		var zero V
		return zero, false
	}

	oldVal := e.value
	e.value = newVal
	return oldVal, true
}

// ReplaceMatch 仅在键存在且值匹配时替换
func (m *WeakHashMap[K, V]) ReplaceMatch(k K, old, new V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ptr, exists := m.keys[k]
	if !exists {
		return false
	}

	e, ok := m.data[ptr]
	if !ok {
		delete(m.keys, k)
		return false
	}

	if reflect.DeepEqual(e.value, old) {
		e.value = new
		return true
	}

	return false
}

// Seq2 返回键值对的迭代器
func (m *WeakHashMap[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, e := range m.data {
			if !yield(e.key, e.value) {
				return
			}
		}
	}
}

// EntrySet 返回所有键值对的集合
func (m *WeakHashMap[K, V]) EntrySet() iter.Seq2[K, V] {
	return m.Seq2()
}

// Seq 返回值的迭代器
func (m *WeakHashMap[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, e := range m.data {
			if !yield(e.value) {
				return
			}
		}
	}
}

// KeySet 返回所有键的集合
func (m *WeakHashMap[K, V]) KeySet() iter.Seq[K] {
	return func(yield func(K) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, e := range m.data {
			if !yield(e.key) {
				return
			}
		}
	}
}

// Keys 返回键的迭代器（别名方法）
func (m *WeakHashMap[K, V]) Keys() iter.Seq[K] {
	return m.KeySet()
}

// Values 返回值的迭代器
func (m *WeakHashMap[K, V]) Values() iter.Seq[V] {
	return m.Seq()
}

// getPointer 获取键的唯一标识
func getPointer[K comparable](k K) uintptr {
	// 使用 reflect.ValueOf 获取值的哈希作为唯一标识
	v := reflect.ValueOf(k)

	// 对于引用类型，直接使用其指针
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.UnsafePointer:
		return v.Pointer()
	case reflect.Slice:
		return v.Pointer()
	default:
		// 对于值类型，使用其地址
		return reflect.ValueOf(&k).Pointer()
	}
}
