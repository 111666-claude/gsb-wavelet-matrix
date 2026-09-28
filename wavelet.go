// Package waveletmatrix 支持 rank / select / 区间第 k 小，只用标准库。
package waveletmatrix

import (
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrRange 表示下标、区间或名次不合法。
	ErrRange = errors.New("waveletmatrix: 下标或区间不合法")
	// ErrValue 表示符号超出构建时的值域。
	ErrValue = errors.New("waveletmatrix: 符号超出值域")
)

// bitVector 是一层位向量：打包的位 + 每个 64 位块的前缀 1 计数。
type bitVector struct {
	bits  []uint64
	ones  []uint32 // ones[i] = bits[:i] 里 1 的个数
	count int      // 1 的总数
	size  int      // 位数
}

func newBitVector(size int) *bitVector {
	return &bitVector{bits: make([]uint64, (size+63)/64), ones: make([]uint32, (size+63)/64+1), size: size}
}

func (bv *bitVector) set(index int) {
	bv.bits[index/64] |= 1 << uint(index%64)
}

func (bv *bitVector) bit(index int) int {
	return int(bv.bits[index/64] >> uint(index%64) & 1)
}

func (bv *bitVector) finish() {
	for index, word := range bv.bits {
		bv.ones[index+1] = bv.ones[index] + uint32(popcount(word))
	}
	bv.count = int(bv.ones[len(bv.ones)-1])
}

func popcount(word uint64) int {
	word -= word >> 1 & 0x5555555555555555
	word = (word & 0x3333333333333333) + (word>>2)&0x3333333333333333
	word = (word + word>>4) & 0x0f0f0f0f0f0f0f0f
	return int(word * 0x0101010101010101 >> 56)
}

// rank1 返回 [0, position) 里 1 的个数，position 取值 [0, size]。
func (bv *bitVector) rank1(position int) int {
	block := position / 64
	if position%64 == 0 {
		return int(bv.ones[block])
	}
	return int(bv.ones[block]) + popcount(bv.bits[block]&((uint64(1)<<uint(position%64))-1))
}

func (bv *bitVector) rank0(position int) int {
	return position - bv.rank1(position)
}

// select1 返回第 r 个 1（r 从 1 起）的下标，r 取值 [1, count]。
func (bv *bitVector) select1(r int) int {
	block := sort.Search(len(bv.ones), func(index int) bool { return int(bv.ones[index]) >= r }) - 1
	rest := r - int(bv.ones[block])
	word := bv.bits[block]
	for offset := 0; offset < 64; offset++ {
		if word&1 == 1 {
			rest--
			if rest == 0 {
				return block*64 + offset
			}
		}
		word >>= 1
	}
	return -1
}

// select0 返回第 r 个 0（r 从 1 起）的下标，r 取值 [1, size-count]。
func (bv *bitVector) select0(r int) int {
	block := sort.Search(len(bv.ones), func(index int) bool { return index*64-int(bv.ones[index]) >= r }) - 1
	rest := r - (block*64 - int(bv.ones[block]))
	word := ^bv.bits[block]
	for offset := 0; offset < 64; offset++ {
		if word&1 == 1 {
			rest--
			if rest == 0 {
				return block*64 + offset
			}
		}
		word >>= 1
	}
	return -1
}

// Matrix 是小波矩阵：按符号的每一位分层存位向量和前缀 rank 表。
type Matrix struct {
	values []uint32     // 原序列副本，仅供 Length / Value
	levels []*bitVector // 从最高位到最低位，每层一个位向量
	zeros  []int        // zeros[d] 是第 d 层 0 的个数（1 区间的起点）
	depth  int          // 层数；空序列为 0
}

// Build 建立索引。序列不会被保留，内部自行拷贝。
func Build(values []uint32) *Matrix {
	copied := make([]uint32, len(values))
	copy(copied, values)
	matrix := &Matrix{values: copied}
	if len(values) == 0 {
		return matrix
	}
	var max uint32
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	depth := 1
	for max>>uint(depth) != 0 {
		depth++
	}
	matrix.depth = depth
	matrix.levels = make([]*bitVector, depth)
	matrix.zeros = make([]int, depth)
	current := copied
	for d := 0; d < depth; d++ {
		shift := uint(depth - 1 - d)
		bv := newBitVector(len(current))
		for index, value := range current {
			if value>>shift&1 == 1 {
				bv.set(index)
			}
		}
		bv.finish()
		matrix.levels[d] = bv
		matrix.zeros[d] = len(current) - bv.count
		// 稳定划分：bit=0 的保持相对顺序放前面，bit=1 的放后面。
		next := make([]uint32, len(current))
		zeroPos, onePos := 0, matrix.zeros[d]
		for _, value := range current {
			if value>>shift&1 == 0 {
				next[zeroPos] = value
				zeroPos++
			} else {
				next[onePos] = value
				onePos++
			}
		}
		current = next
	}
	return matrix
}

// Length 返回序列长度。
func (m *Matrix) Length() int {
	return len(m.values)
}

// Value 返回下标处的值。
func (m *Matrix) Value(position int) (uint32, error) {
	if position < 0 || position >= len(m.values) {
		return 0, fmt.Errorf("position=%d 越界（序列长度 %d）: %w", position, len(m.values), ErrRange)
	}
	return m.values[position], nil
}

// checkValue 校验符号在构建时的值域内。
func (m *Matrix) checkValue(value uint32) error {
	if m.depth == 0 {
		return nil
	}
	if value>>uint(m.depth) != 0 {
		return fmt.Errorf("value=%d 超出值域（构建时最大需要 %d 位）: %w", value, m.depth, ErrValue)
	}
	return nil
}

// Rank 返回 [0, position) 里 value 出现的次数。position 取值 [0, Length()]。
func (m *Matrix) Rank(value uint32, position int) (int, error) {
	if position < 0 || position > len(m.values) {
		return 0, fmt.Errorf("position=%d 越界（合法范围 [0, %d]）: %w", position, len(m.values), ErrRange)
	}
	if err := m.checkValue(value); err != nil {
		return 0, err
	}
	left, right := 0, position
	for d := 0; d < m.depth; d++ {
		shift := uint(m.depth - 1 - d)
		bv := m.levels[d]
		if value>>shift&1 == 0 {
			left = bv.rank0(left)
			right = bv.rank0(right)
		} else {
			left = m.zeros[d] + bv.rank1(left)
			right = m.zeros[d] + bv.rank1(right)
		}
	}
	return right - left, nil
}

// Select 返回 value 第 r 次出现（r 从 1 起）的下标。
func (m *Matrix) Select(value uint32, r int) (int, error) {
	if r < 1 {
		return 0, fmt.Errorf("r=%d 必须从 1 起: %w", r, ErrRange)
	}
	if err := m.checkValue(value); err != nil {
		return 0, err
	}
	total, err := m.Rank(value, len(m.values))
	if err != nil {
		return 0, err
	}
	if r > total {
		return 0, fmt.Errorf("r=%d 超过 value=%d 的出现次数 %d: %w", r, value, total, ErrRange)
	}
	// 自顶向下走到 value 所在区间，第 r 次出现是区间内的第 r 个元素。
	left := 0
	for d := 0; d < m.depth; d++ {
		shift := uint(m.depth - 1 - d)
		bv := m.levels[d]
		if value>>shift&1 == 0 {
			left = bv.rank0(left)
		} else {
			left = m.zeros[d] + bv.rank1(left)
		}
	}
	position := left + r - 1
	// 自底向上把最深层下标映射回原序列下标。
	for d := m.depth - 1; d >= 0; d-- {
		shift := uint(m.depth - 1 - d)
		bv := m.levels[d]
		if value>>shift&1 == 0 {
			position = bv.select0(position + 1)
		} else {
			position = bv.select1(position - m.zeros[d] + 1)
		}
	}
	return position, nil
}

// KthSmallest 返回 [left, right) 里第 k 小的值（k 从 1 起）。
func (m *Matrix) KthSmallest(left, right, k int) (uint32, error) {
	if left < 0 || right > len(m.values) || left >= right {
		return 0, fmt.Errorf("区间 [%d, %d) 不合法（序列长度 %d）: %w", left, right, len(m.values), ErrRange)
	}
	if k < 1 || k > right-left {
		return 0, fmt.Errorf("k=%d 越界（区间长度 %d，k 从 1 起）: %w", k, right-left, ErrRange)
	}
	var value uint32
	for d := 0; d < m.depth; d++ {
		shift := uint(m.depth - 1 - d)
		bv := m.levels[d]
		zeros := bv.rank0(right) - bv.rank0(left)
		if k <= zeros {
			left = bv.rank0(left)
			right = bv.rank0(right)
		} else {
			value |= 1 << shift
			k -= zeros
			left = m.zeros[d] + bv.rank1(left)
			right = m.zeros[d] + bv.rank1(right)
		}
	}
	return value, nil
}
