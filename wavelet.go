// Package waveletmatrix 支持 rank / select / 区间第 k 小，只用标准库。
package waveletmatrix

import (
	"errors"
	"fmt"
)

var (
	// ErrRange 表示下标或区间不合法。
	ErrRange = errors.New("waveletmatrix: 下标或区间不合法")
	// ErrValue 表示符号超出构建时的值域。
	ErrValue = errors.New("waveletmatrix: 符号超出值域")
)

// bitVector 是分层的位向量，外加每个 64 位字开头处 1 的前缀计数，
// rank1 为 O(1)，select0/select1 为 O(log n)，不回扫原序列。
type bitVector struct {
	bits   []uint64
	prefix []int // prefix[i] = bits[0:i] 个字里 1 的个数，长度 len(bits)+1
	zeros  int   // 该层 0 的个数
	length int
}

func newBitVector(flags []bool) *bitVector {
	n := len(flags)
	bv := &bitVector{
		bits:   make([]uint64, (n+63)/64),
		prefix: make([]int, (n+63)/64+1),
		length: n,
	}
	for index, set := range flags {
		if set {
			bv.bits[index/64] |= 1 << uint(index%64)
		}
	}
	ones := 0
	for index, word := range bv.bits {
		bv.prefix[index] = ones
		ones += popcount(word)
	}
	bv.prefix[len(bv.bits)] = ones
	bv.zeros = n - ones
	return bv
}

func popcount(word uint64) int {
	count := 0
	for word != 0 {
		word &= word - 1
		count++
	}
	return count
}

// rank1 返回 [0, position) 里 1 的个数，position ∈ [0, length]。
func (bv *bitVector) rank1(position int) int {
	if position == 0 {
		return 0
	}
	word := position / 64
	offset := uint(position % 64)
	count := bv.prefix[word]
	if offset != 0 {
		count += popcount(bv.bits[word] << (64 - offset))
	}
	return count
}

func (bv *bitVector) bitAt(position int) bool {
	return bv.bits[position/64]&(1<<uint(position%64)) != 0
}

// select1 返回第 rank 个（1 起）1 的下标。
func (bv *bitVector) select1(rank int) int {
	return bv.selectBit(rank, true)
}

// select0 返回第 rank 个（1 起）0 的下标。
func (bv *bitVector) select0(rank int) int {
	return bv.selectBit(rank, false)
}

func (bv *bitVector) selectBit(rank int, one bool) int {
	if rank < 1 {
		return -1
	}
	// 二分落在哪个 64 位字。
	lo, hi := 0, len(bv.bits)
	for lo < hi {
		mid := (lo + hi) / 2
		var count int
		if one {
			count = bv.prefix[mid+1]
		} else {
			count = (mid+1)*64 - bv.prefix[mid+1]
			if tail := (mid + 1) * 64; tail > bv.length {
				count -= tail - bv.length
			}
		}
		if count < rank {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= len(bv.bits) {
		return -1
	}
	before := bv.prefix[lo]
	if !one {
		before = lo*64 - before
		if tail := lo * 64; tail > bv.length {
			before -= tail - bv.length
		}
	}
	remain := rank - before // 该字内第几个目标位（1 起）
	word := bv.bits[lo]
	if !one {
		// 末尾虚拟位按 1 处理，保证不会被选到。
		word |= ^uint64(0) << uint(bv.length-lo*64)
		word = ^word
	}
	for remain > 0 {
		next := word & (^word + 1) // 最低位的 1
		if next == 0 {
			return -1
		}
		remain--
		if remain == 0 {
			position := lo*64 + trailingZeros(next)
			if position >= bv.length {
				return -1
			}
			return position
		}
		word ^= next
	}
	return -1
}

func trailingZeros(word uint64) int {
	count := 0
	for word&1 == 0 {
		word >>= 1
		count++
	}
	return count
}

// Matrix 是小波矩阵：每个符号位一层，存位向量和前缀 rank 表。
type Matrix struct {
	levels []*bitVector // 最高位到最低位
	zeros  []int        // 每层 0 的个数，即 1 分支的起始下标
	bits   int          // 使用的位数
	hasMax bool
	max    uint32 // 构建时的值域上界（值域为 [0, max]）
	values []uint32
}

// Build 建立索引，只依赖分层位向量与前缀计数。
func Build(values []uint32) *Matrix {
	copied := make([]uint32, len(values))
	copy(copied, values)

	m := &Matrix{values: copied}
	if len(copied) > 0 {
		m.hasMax = true
		m.max = copied[0]
		for _, value := range copied[1:] {
			if value > m.max {
				m.max = value
			}
		}
		m.bits = 1
		for boundary := uint32(1); boundary <= m.max; {
			if boundary == 1<<31 {
				m.bits = 32
				break
			}
			boundary <<= 1
			m.bits++
		}
	}

	current := copied
	m.levels = make([]*bitVector, m.bits)
	m.zeros = make([]int, m.bits)
	for level := 0; level < m.bits; level++ {
		shift := uint(m.bits - 1 - level)
		flags := make([]bool, len(current))
		zeros := make([]uint32, 0, len(current))
		ones := make([]uint32, 0, len(current))
		for index, value := range current {
			set := value&(1<<shift) != 0
			flags[index] = set
			if set {
				ones = append(ones, value)
			} else {
				zeros = append(zeros, value)
			}
		}
		bv := newBitVector(flags)
		m.levels[level] = bv
		m.zeros[level] = bv.zeros
		current = append(zeros, ones...)
	}
	return m
}

// Length 返回序列长度。
func (m *Matrix) Length() int { return len(m.values) }

// Value 返回下标处的值。
func (m *Matrix) Value(position int) (uint32, error) {
	if position < 0 || position >= len(m.values) {
		return 0, fmt.Errorf("%w: position=%d", ErrRange, position)
	}
	return m.values[position], nil
}

func (m *Matrix) checkValue(value uint32) error {
	if m.hasMax && value > m.max {
		return fmt.Errorf("%w: value=%d 超过构建时上界 %d", ErrValue, value, m.max)
	}
	return nil
}

// Rank 只数 [0, position) 里 value 出现的次数。
// position 允许等于序列长度；越界（含负数）报错。空序列上对 position=0 恒为 0。
func (m *Matrix) Rank(value uint32, position int) (int, error) {
	if position < 0 || position > len(m.values) {
		return 0, fmt.Errorf("%w: position=%d", ErrRange, position)
	}
	if !m.hasMax {
		return 0, nil
	}
	if err := m.checkValue(value); err != nil {
		return 0, err
	}
	lo, hi := 0, position
	for level := 0; level < m.bits; level++ {
		shift := uint(m.bits - 1 - level)
		zl := m.levels[level].rank1(lo)
		zh := m.levels[level].rank1(hi)
		if value&(1<<shift) != 0 {
			lo = m.zeros[level] + zl
			hi = m.zeros[level] + zh
		} else {
			lo -= zl
			hi -= zh
		}
	}
	return hi - lo, nil
}

// Select 返回 value 第 rank 次（1 起）出现的下标；超过出现次数报错。
func (m *Matrix) Select(value uint32, rank int) (int, error) {
	if rank < 1 {
		return -1, fmt.Errorf("%w: rank=%d 必须从 1 开始", ErrRange, rank)
	}
	if !m.hasMax {
		return -1, fmt.Errorf("%w: rank=%d 超过出现次数 0", ErrRange, rank)
	}
	if err := m.checkValue(value); err != nil {
		return -1, err
	}
	// 正向把 [0, n) 按 value 的各位映射下去，lo 即 value 稳定段的起点。
	lo, hi := 0, len(m.values)
	for level := 0; level < m.bits; level++ {
		shift := uint(m.bits - 1 - level)
		zl := m.levels[level].rank1(lo)
		zh := m.levels[level].rank1(hi)
		if value&(1<<shift) != 0 {
			lo = m.zeros[level] + zl
			hi = m.zeros[level] + zh
		} else {
			lo -= zl
			hi -= zh
		}
	}
	position := lo + rank - 1
	// 从最低位逐层逆向上溯。
	for level := m.bits - 1; level >= 0; level-- {
		shift := uint(m.bits - 1 - level)
		bv := m.levels[level]
		var mapped int
		if value&(1<<shift) != 0 {
			mapped = bv.select1(position - m.zeros[level] + 1)
		} else {
			mapped = bv.select0(position + 1)
		}
		if mapped < 0 {
			return -1, fmt.Errorf("%w: rank=%d 超过 value=%d 的出现次数", ErrRange, rank, value)
		}
		position = mapped
	}
	return position, nil
}

// KthSmallest 返回 [left, right) 里第 k 小（k 从 1 起）的符号。
func (m *Matrix) KthSmallest(left, right, k int) (uint32, error) {
	if left < 0 || right < 0 || left > len(m.values) || right > len(m.values) {
		return 0, fmt.Errorf("%w: 区间 [%d,%d) 越界（长度 %d）", ErrRange, left, right, len(m.values))
	}
	if left >= right {
		return 0, fmt.Errorf("%w: left=%d 必须小于 right=%d", ErrRange, left, right)
	}
	if k <= 0 {
		return 0, fmt.Errorf("%w: k=%d 必须从 1 开始", ErrRange, k)
	}
	if k > right-left {
		return 0, fmt.Errorf("%w: k=%d 超过区间长度 %d", ErrRange, k, right-left)
	}

	lo, hi := left, right
	var value uint32
	for level := 0; level < m.bits; level++ {
		bv := m.levels[level]
		zl := bv.rank1(lo)
		zh := bv.rank1(hi)
		zeroCount := (hi - lo) - (zh - zl)
		if k <= zeroCount {
			lo -= zl
			hi -= zh
		} else {
			k -= zeroCount
			value |= 1 << uint(m.bits-1-level)
			lo = m.zeros[level] + zl
			hi = m.zeros[level] + zh
		}
	}
	return value, nil
}
