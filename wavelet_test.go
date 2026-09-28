package waveletmatrix

import (
	"sort"
	"testing"
	"time"
)

// sample 是 "banana" 的字节序列。
func sample() *Matrix {
	return Build([]uint32{98, 97, 110, 97, 110, 97})
}

// lcg 生成确定性的伪随机序列，便于复现。
func lcg(n int, mod uint32) []uint32 {
	values := make([]uint32, n)
	state := uint64(12345)
	for index := range values {
		state = state*6364136223846793005 + 1442695040888963407
		values[index] = uint32(state>>33) % mod
	}
	return values
}

func TestLengthAndValue(t *testing.T) {
	matrix := sample()
	if matrix.Length() != 6 {
		t.Fatalf("长度不对：%d", matrix.Length())
	}
	value, err := matrix.Value(1)
	if err != nil || value != 97 {
		t.Fatalf("Value(1) 不对：%d %v", value, err)
	}
	if _, err := matrix.Value(9); err == nil {
		t.Fatal("下标越界应该报错")
	}
}

func TestRankPrefix(t *testing.T) {
	matrix := sample()
	cases := []struct {
		value    uint32
		position int
		want     int
	}{
		{97, 3, 1}, // b a n：前 3 个字节只有 1 个 a
		{97, 6, 3},
		{97, 0, 0},
		{98, 1, 1},
		{98, 6, 1},
		{110, 5, 2},
		{99, 6, 0}, // 值域内但没出现过
	}
	for _, c := range cases {
		got, err := matrix.Rank(c.value, c.position)
		if err != nil || got != c.want {
			t.Fatalf("Rank(%d, %d) = %d, %v；想要 %d", c.value, c.position, got, err, c.want)
		}
	}
}

func TestRankRejectsBadPosition(t *testing.T) {
	matrix := sample()
	if _, err := matrix.Rank(97, -1); err == nil {
		t.Fatal("position 为负应该报错")
	}
	if _, err := matrix.Rank(97, 7); err == nil {
		t.Fatal("position 超过长度应该报错")
	}
}

func TestRankRejectsOutOfDomain(t *testing.T) {
	matrix := sample()
	if _, err := matrix.Rank(200, 6); err == nil {
		t.Fatal("超出构建时值域应该报错")
	}
}

func TestRankInvariantTotalCount(t *testing.T) {
	values := lcg(1000, 17)
	matrix := Build(values)
	counts := map[uint32]int{}
	for _, value := range values {
		counts[value]++
	}
	for value, want := range counts {
		got, err := matrix.Rank(value, matrix.Length())
		if err != nil || got != want {
			t.Fatalf("Rank(%d, n) = %d, %v；想要 %d", value, got, err, want)
		}
	}
}

func TestRankMatchesScan(t *testing.T) {
	values := lcg(500, 9)
	matrix := Build(values)
	for position := 0; position <= len(values); position += 7 {
		for symbol := uint32(0); symbol < 9; symbol++ {
			want := 0
			for index := 0; index < position; index++ {
				if values[index] == symbol {
					want++
				}
			}
			got, err := matrix.Rank(symbol, position)
			if err != nil || got != want {
				t.Fatalf("Rank(%d, %d) = %d, %v；想要 %d", symbol, position, got, err, want)
			}
		}
	}
}

func TestSelect(t *testing.T) {
	matrix := sample()
	cases := []struct {
		value uint32
		rank  int
		want  int
	}{
		{97, 1, 1},
		{97, 2, 3},
		{97, 3, 5},
		{98, 1, 0},
		{110, 1, 2},
		{110, 2, 4},
	}
	for _, c := range cases {
		got, err := matrix.Select(c.value, c.rank)
		if err != nil || got != c.want {
			t.Fatalf("Select(%d, %d) = %d, %v；想要 %d", c.value, c.rank, got, err, c.want)
		}
	}
}

func TestSelectRejectsBadRank(t *testing.T) {
	matrix := sample()
	if _, err := matrix.Select(97, 0); err == nil {
		t.Fatal("rank 从 1 开始，0 应该报错")
	}
	if _, err := matrix.Select(97, 4); err == nil {
		t.Fatal("rank 超过出现次数应该报错")
	}
	if _, err := matrix.Select(122, 1); err == nil {
		t.Fatal("没出现过的符号应该报错")
	}
	if _, err := matrix.Select(200, 1); err == nil {
		t.Fatal("超出值域应该报错")
	}
}

func TestSelectRankInverse(t *testing.T) {
	values := lcg(2000, 13)
	matrix := Build(values)
	for symbol := uint32(0); symbol < 13; symbol++ {
		total, err := matrix.Rank(symbol, matrix.Length())
		if err != nil {
			t.Fatalf("Rank(%d, n) 报错：%v", symbol, err)
		}
		for rank := 1; rank <= total; rank++ {
			position, err := matrix.Select(symbol, rank)
			if err != nil {
				t.Fatalf("Select(%d, %d) 报错：%v", symbol, rank, err)
			}
			if values[position] != symbol {
				t.Fatalf("Select(%d, %d) = %d，但 values[%d] = %d", symbol, rank, position, position, values[position])
			}
			// rank 与 select 互为逆运算。
			back, err := matrix.Rank(symbol, position+1)
			if err != nil || back != rank {
				t.Fatalf("Rank(%d, %d) = %d, %v；想要 %d", symbol, position+1, back, err, rank)
			}
		}
		if total > 0 {
			last, err := matrix.Select(symbol, total)
			if err != nil {
				t.Fatalf("Select(%d, %d) 报错：%v", symbol, total, err)
			}
			for index := last + 1; index < len(values); index++ {
				if values[index] == symbol {
					t.Fatalf("Select(%d, %d) = %d 不是最后一次出现", symbol, total, last)
				}
			}
		}
	}
}

func TestKthOverWholeSequence(t *testing.T) {
	matrix := sample()
	value, err := matrix.KthSmallest(0, matrix.Length(), 1)
	if err != nil || value != 97 {
		t.Fatalf("整串第 1 小应该是 a：%d %v", value, err)
	}
	value, err = matrix.KthSmallest(0, matrix.Length(), 6)
	if err != nil || value != 110 {
		t.Fatalf("整串第 6 小应该是 n：%d %v", value, err)
	}
}

func TestKthWithinRange(t *testing.T) {
	matrix := sample()
	// banana 前两个字节是 b、a，第 2 小是 b（98）。
	value, err := matrix.KthSmallest(0, 2, 2)
	if err != nil || value != 98 {
		t.Fatalf("KthSmallest(0, 2, 2) = %d, %v；想要 98", value, err)
	}
	value, err = matrix.KthSmallest(1, 4, 1)
	if err != nil || value != 97 {
		t.Fatalf("KthSmallest(1, 4, 1) = %d, %v；想要 97", value, err)
	}
}

func TestKthMatchesSortedCopy(t *testing.T) {
	values := lcg(3000, 100)
	matrix := Build(values)
	for trial := 0; trial < 200; trial++ {
		left := trial * 13 % len(values)
		right := left + 1 + trial*7%(len(values)-left)
		window := append([]uint32(nil), values[left:right]...)
		sort.Slice(window, func(a, b int) bool { return window[a] < window[b] })
		for k := 1; k <= len(window); k += 1 + len(window)/5 {
			got, err := matrix.KthSmallest(left, right, k)
			if err != nil || got != window[k-1] {
				t.Fatalf("KthSmallest(%d, %d, %d) = %d, %v；想要 %d", left, right, k, got, err, window[k-1])
			}
		}
	}
}

func TestKthRejectsBadRange(t *testing.T) {
	matrix := sample()
	if _, err := matrix.KthSmallest(3, 3, 1); err == nil {
		t.Fatal("空区间应该报错")
	}
	if _, err := matrix.KthSmallest(4, 2, 1); err == nil {
		t.Fatal("left > right 应该报错")
	}
	if _, err := matrix.KthSmallest(-1, 2, 1); err == nil {
		t.Fatal("left 为负应该报错")
	}
	if _, err := matrix.KthSmallest(0, 7, 1); err == nil {
		t.Fatal("right 越界应该报错")
	}
	if _, err := matrix.KthSmallest(0, 6, 0); err == nil {
		t.Fatal("k 从 1 开始，0 应该报错")
	}
	if _, err := matrix.KthSmallest(0, 6, -3); err == nil {
		t.Fatal("k 为负应该报错")
	}
	if _, err := matrix.KthSmallest(2, 4, 3); err == nil {
		t.Fatal("k 超过区间长度应该报错")
	}
}

func TestEmptyMatrix(t *testing.T) {
	matrix := Build(nil)
	if matrix.Length() != 0 {
		t.Fatalf("空序列长度应该是 0：%d", matrix.Length())
	}
	got, err := matrix.Rank(1, 0)
	if err != nil || got != 0 {
		t.Fatalf("空序列的 rank 应该是 0：%d %v", got, err)
	}
	if _, err := matrix.Rank(1, 1); err == nil {
		t.Fatal("空序列 position 越界应该报错")
	}
	if _, err := matrix.Select(1, 1); err == nil {
		t.Fatal("空序列的 select 应该报错")
	}
	if _, err := matrix.KthSmallest(0, 1, 1); err == nil {
		t.Fatal("空序列的 kth 应该报错")
	}
}

func TestSingleElement(t *testing.T) {
	matrix := Build([]uint32{42})
	if got, _ := matrix.Rank(42, 1); got != 1 {
		t.Fatalf("Rank(42, 1) = %d；想要 1", got)
	}
	position, err := matrix.Select(42, 1)
	if err != nil || position != 0 {
		t.Fatalf("Select(42, 1) = %d, %v；想要 0", position, err)
	}
	value, err := matrix.KthSmallest(0, 1, 1)
	if err != nil || value != 42 {
		t.Fatalf("KthSmallest(0, 1, 1) = %d, %v；想要 42", value, err)
	}
}

func TestDeterministic(t *testing.T) {
	values := lcg(1000, 50)
	matrix := Build(values)
	for repeat := 0; repeat < 3; repeat++ {
		rank, err1 := matrix.Rank(7, 500)
		sel, err2 := matrix.Select(7, 3)
		kth, err3 := matrix.KthSmallest(100, 900, 123)
		if err1 != nil || err2 != nil || err3 != nil {
			t.Fatalf("查询报错：%v %v %v", err1, err2, err3)
		}
		if repeat > 0 {
			rank2, _ := matrix.Rank(7, 500)
			sel2, _ := matrix.Select(7, 3)
			kth2, _ := matrix.KthSmallest(100, 900, 123)
			if rank != rank2 || sel != sel2 || kth != kth2 {
				t.Fatal("同一查询重复执行结果不同")
			}
		}
	}
}

// TestScale 对应规模要求：10^6 个符号、三种查询各 10^5 次在 2 秒内。
func TestScale(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过规模测试")
	}
	const n = 1_000_000
	const queries = 100_000
	values := lcg(n, 1<<20)
	matrix := Build(values)

	start := time.Now()
	var sink uint64
	state := uint64(99991)
	next := func() uint64 {
		state = state*6364136223846793005 + 1442695040888963407
		return state >> 33
	}
	for query := 0; query < queries; query++ {
		count, err := matrix.Rank(uint32(next()%(1<<20)), int(next()%uint64(n+1)))
		if err != nil {
			t.Fatalf("rank 报错：%v", err)
		}
		sink += uint64(count)
	}
	for query := 0; query < queries; query++ {
		symbol := uint32(next() % (1 << 20))
		total, _ := matrix.Rank(symbol, n)
		if total == 0 {
			continue
		}
		position, err := matrix.Select(symbol, int(next()%uint64(total))+1)
		if err != nil {
			t.Fatalf("select 报错：%v", err)
		}
		sink += uint64(position)
	}
	for query := 0; query < queries; query++ {
		left := int(next() % uint64(n))
		right := left + 1 + int(next()%uint64(n-left))
		value, err := matrix.KthSmallest(left, right, int(next()%uint64(right-left))+1)
		if err != nil {
			t.Fatalf("kth 报错：%v", err)
		}
		sink += uint64(value)
	}
	elapsed := time.Since(start)
	t.Logf("3×10^5 次查询（n=10^6）耗时 %v，sink=%d", elapsed, sink)
	if elapsed > 2*time.Second {
		t.Fatalf("超过 2 秒：%v", elapsed)
	}
}
