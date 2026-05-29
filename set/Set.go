package _set

// Set 对标 java.util.Set
type Set[T comparable] struct {
	data map[T]struct{}
}

// NewSet 构造空集合
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		data: make(map[T]struct{}),
	}
}

// Add 同 Java boolean add(E e)
// 添加元素，已存在返回false，不存在添加并返回true
func (h *Set[T]) Add(e T) bool {
	if h.Contains(e) {
		return false
	}
	h.data[e] = struct{}{}
	return true
}

// Remove 同 Java boolean remove(Object o)
// 删除元素，存在删除返回true，不存在返回false
func (h *Set[T]) Remove(e T) bool {
	if !h.Contains(e) {
		return false
	}
	delete(h.data, e)
	return true
}

// Contains 同 Java boolean contains(Object o)
func (h *Set[T]) Contains(e T) bool {
	_, ok := h.data[e]
	return ok
}

// Size 同 Java int size()
func (h *Set[T]) Size() int {
	return len(h.data)
}

// IsEmpty 同 Java boolean isEmpty()
func (h *Set[T]) IsEmpty() bool {
	return len(h.data) == 0
}

// Clear 同 Java void clear()
func (h *Set[T]) Clear() {
	h.data = make(map[T]struct{})
}

// AddAll 同 Java boolean addAll(Collection<?> c)
func (h *Set[T]) AddAll(c []T) bool {
	modified := false
	for _, v := range c {
		if h.Add(v) {
			modified = true
		}
	}
	return modified
}

// RemoveAll 同 Java boolean removeAll(Collection<?> c)
func (h *Set[T]) RemoveAll(c []T) bool {
	modified := false
	for _, v := range c {
		if h.Remove(v) {
			modified = true
		}
	}
	return modified
}

// RetainAll 同 Java boolean retainAll(Collection<?> c) 保留交集
func (h *Set[T]) RetainAll(c []T) bool {
	temp := make(map[T]struct{})
	for _, v := range c {
		temp[v] = struct{}{}
	}
	originLen := h.Size()
	for k := range h.data {
		if _, ok := temp[k]; !ok {
			delete(h.data, k)
		}
	}
	return originLen != h.Size()
}

// ContainsAll 同 Java boolean containsAll(Collection<?> c)
func (h *Set[T]) ContainsAll(c []T) bool {
	for _, v := range c {
		if !h.Contains(v) {
			return false
		}
	}
	return true
}

// ToSlice 对标 Java toArray() 转为切片
func (h *Set[T]) ToSlice() []T {
	res := make([]T, 0, len(h.data))
	for k := range h.data {
		res = append(res, k)
	}
	return res
}

// Equals 对标 equals 判断集合元素完全一致
func (h *Set[T]) Equals(other *Set[T]) bool {
	if h.Size() != other.Size() {
		return false
	}
	for k := range h.data {
		if !other.Contains(k) {
			return false
		}
	}
	return true
}
