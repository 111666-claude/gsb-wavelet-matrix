// Package waveletmatrix 支持 rank / select / 区间第 k 小，只用标准库。
package waveletmatrix

import (
	"errors"
	"sort"
)

var (
	// ErrRange 表示下标或区间不合法。
	ErrRange = errors.New("waveletmatrix: 下标或区间不合法")
	// ErrNoSelect 表示 select 还没有实现。
	ErrNoSelect = errors.New("waveletmatrix: select 还没有实现")
)

// Matrix 是小波矩阵。
// 缺陷：只把原序列和一份排好序的副本存下来，没有位向量和前缀 rank 表。
type Matrix struct {
	values []uint32
	sorted []uint32
}

// Build 建立索引。
func Build(values []uint32) *Matrix {
	sorted := make([]uint32, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	copied := make([]uint32, len(values))
	copy(copied, values)
	return &Matrix{values: copied, sorted: sorted}
}

// Length 返回序列长度。
func (m *Matrix) Length() int {
	return len(m.values)
}

// Value 返回下标处的值。
func (m *Matrix) Value(position int) (uint32, error) {
	if position < 0 || position >= len(m.values) {
		return 0, ErrRange
	}
	return m.values[position], nil
}

// Rank 缺陷：忽略 position，直接给整个序列里 value 的出现次数。
func (m *Matrix) Rank(value uint32, position int) int {
	if position < 0 || position > len(m.values) {
		return -1
	}
	first := sort.Search(len(m.sorted), func(index int) bool { return m.sorted[index] >= value })
	last := sort.Search(len(m.sorted), func(index int) bool { return m.sorted[index] > value })
	return last - first
}

// Select 缺陷：还没实现。
func (m *Matrix) Select(value uint32, rank int) (int, error) {
	return -1, ErrNoSelect
}

// KthSmallest 缺陷：忽略区间，直接在整串排序结果上取第 k 小（k 从 1 开始）。
func (m *Matrix) KthSmallest(left, right, k int) (uint32, error) {
	if left < 0 || right > len(m.values) || left >= right {
		return 0, ErrRange
	}
	if k < 1 || k > len(m.sorted) {
		return 0, ErrRange
	}
	return m.sorted[k-1], nil
}
