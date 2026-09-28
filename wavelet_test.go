package waveletmatrix

import (
	"errors"
	"math/rand"
	"sort"
	"testing"
)

// sample 是 "banana" 的字节序列。
func sample() *Matrix {
	return Build([]uint32{98, 97, 110, 97, 110, 97})
}

func bruteRank(values []uint32, value uint32, position int) int {
	count := 0
	for index := 0; index < position; index++ {
		if values[index] == value {
			count++
		}
	}
	return count
}

func bruteSelect(values []uint32, value uint32, r int) int {
	count := 0
	for index, v := range values {
		if v == value {
			count++
			if count == r {
				return index
			}
		}
	}
	return -1
}

func bruteKth(values []uint32, left, right, k int) uint32 {
	window := make([]uint32, right-left)
	copy(window, values[left:right])
	sort.Slice(window, func(a, b int) bool { return window[a] < window[b] })
	return window[k-1]
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
		{97, 3, 1},  // 前三个字节 b a n 里只有 1 个 a
		{97, 6, 3},  // 整串 3 个 a
		{97, 0, 0},  // 空前缀
		{98, 1, 1},  // 第一个字节就是 b
		{110, 5, 2}, // 前五个字节里 2 个 n
		{122, 6, 0}, // 值域内没出现过的符号
	}
	for _, c := range cases {
		got, err := matrix.Rank(c.value, c.position)
		if err != nil || got != c.want {
			t.Fatalf("Rank(%d, %d) = %d, %v；想要 %d", c.value, c.position, got, err, c.want)
		}
	}
}

func TestRankRejectsBadArgs(t *testing.T) {
	matrix := sample()
	for _, position := range []int{-1, 7} {
		if _, err := matrix.Rank(97, position); !errors.Is(err, ErrRange) {
			t.Fatalf("position=%d 应该报 ErrRange：%v", position, err)
		}
	}
	if _, err := matrix.Rank(128, 3); !errors.Is(err, ErrValue) {
		t.Fatalf("value 超出值域应该报 ErrValue：%v", err)
	}
}

func TestSelect(t *testing.T) {
	matrix := sample()
	cases := []struct {
		value uint32
		r     int
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
		got, err := matrix.Select(c.value, c.r)
		if err != nil || got != c.want {
			t.Fatalf("Select(%d, %d) = %d, %v；想要 %d", c.value, c.r, got, err, c.want)
		}
	}
}

func TestSelectRejectsBadArgs(t *testing.T) {
	matrix := sample()
	if _, err := matrix.Select(97, 0); !errors.Is(err, ErrRange) {
		t.Fatalf("r=0 应该报 ErrRange：%v", err)
	}
	if _, err := matrix.Select(97, 4); !errors.Is(err, ErrRange) {
		t.Fatalf("r 超过出现次数应该报 ErrRange：%v", err)
	}
	if _, err := matrix.Select(122, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("没出现过的符号应该报 ErrRange：%v", err)
	}
	if _, err := matrix.Select(128, 1); !errors.Is(err, ErrValue) {
		t.Fatalf("value 超出值域应该报 ErrValue：%v", err)
	}
}

func TestSelectIsInverseOfRank(t *testing.T) {
	matrix := sample()
	// Select(v, Rank(v, n)-1+1) 即最后一次出现的位置；rank 与 select 互逆。
	for _, value := range []uint32{97, 98, 110} {
		total, err := matrix.Rank(value, matrix.Length())
		if err != nil {
			t.Fatal(err)
		}
		last, err := matrix.Select(value, total)
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := matrix.Rank(value, last); got != total-1 {
			t.Fatalf("Rank(%d, Select(%d, %d)) 应为 %d：%d", value, value, total, total-1, got)
		}
		if got, _ := matrix.Rank(value, last+1); got != total {
			t.Fatalf("Rank(%d, Select(%d, %d)+1) 应为 %d：%d", value, value, total, total, got)
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

func TestKthWithinWindow(t *testing.T) {
	matrix := sample()
	value, err := matrix.KthSmallest(0, 2, 2)
	if err != nil || value != 98 {
		t.Fatalf("[0,2) 第 2 小应该是 b(98)：%d %v", value, err)
	}
	value, err = matrix.KthSmallest(1, 4, 1)
	if err != nil || value != 97 {
		t.Fatalf("[1,4) 第 1 小应该是 a(97)：%d %v", value, err)
	}
	value, err = matrix.KthSmallest(2, 6, 3)
	if err != nil || value != 110 {
		t.Fatalf("[2,6) 第 3 小应该是 n(110)：%d %v", value, err)
	}
}

func TestKthRejectsBadRange(t *testing.T) {
	matrix := sample()
	if _, err := matrix.KthSmallest(3, 3, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("空区间应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(4, 2, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("left > right 应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(0, 6, 0); !errors.Is(err, ErrRange) {
		t.Fatalf("k 从 1 开始，0 应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(0, 2, 3); !errors.Is(err, ErrRange) {
		t.Fatalf("k 超过区间长度应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(-1, 2, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("left 为负应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(0, 7, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("right 越界应该报 ErrRange：%v", err)
	}
}

func TestEmptyMatrix(t *testing.T) {
	matrix := Build(nil)
	if matrix.Length() != 0 {
		t.Fatalf("空序列长度应该是 0：%d", matrix.Length())
	}
	if got, err := matrix.Rank(1, 0); err != nil || got != 0 {
		t.Fatalf("空序列的 rank 应该是 0：%d %v", got, err)
	}
	if _, err := matrix.Rank(1, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("空序列 position>0 应该报 ErrRange：%v", err)
	}
	if _, err := matrix.Select(1, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("空序列的 select 应该报 ErrRange：%v", err)
	}
	if _, err := matrix.KthSmallest(0, 0, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("空序列的 kth 应该报 ErrRange：%v", err)
	}
}

func TestAgainstBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	values := make([]uint32, 500)
	for index := range values {
		values[index] = uint32(rng.Intn(20))
	}
	matrix := Build(values)
	for trial := 0; trial < 2000; trial++ {
		value := uint32(rng.Intn(22))
		position := rng.Intn(len(values) + 1)
		got, err := matrix.Rank(value, position)
		if err != nil {
			t.Fatal(err)
		}
		if want := bruteRank(values, value, position); got != want {
			t.Fatalf("Rank(%d, %d) = %d；想要 %d", value, position, got, want)
		}
		total, _ := matrix.Rank(value, len(values))
		if total > 0 {
			r := rng.Intn(total) + 1
			sel, err := matrix.Select(value, r)
			if err != nil {
				t.Fatal(err)
			}
			if want := bruteSelect(values, value, r); sel != want {
				t.Fatalf("Select(%d, %d) = %d；想要 %d", value, r, sel, want)
			}
		}
		left := rng.Intn(len(values))
		right := left + 1 + rng.Intn(len(values)-left)
		k := rng.Intn(right-left) + 1
		kth, err := matrix.KthSmallest(left, right, k)
		if err != nil {
			t.Fatal(err)
		}
		if want := bruteKth(values, left, right, k); kth != want {
			t.Fatalf("KthSmallest(%d, %d, %d) = %d；想要 %d", left, right, k, kth, want)
		}
	}
}

func TestInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	values := make([]uint32, 300)
	for index := range values {
		values[index] = uint32(rng.Intn(5))
	}
	matrix := Build(values)
	for value := uint32(0); value < 5; value++ {
		total, err := matrix.Rank(value, matrix.Length())
		if err != nil {
			t.Fatal(err)
		}
		if want := bruteRank(values, value, len(values)); total != want {
			t.Fatalf("Rank(%d, n) 应等于出现次数 %d：%d", value, want, total)
		}
		for r := 1; r <= total; r++ {
			first, err := matrix.Select(value, r)
			if err != nil {
				t.Fatal(err)
			}
			again, err := matrix.Select(value, r)
			if err != nil || again != first {
				t.Fatalf("重复执行结果应一致：%d vs %d", first, again)
			}
			if got, _ := matrix.Rank(value, first); got != r-1 {
				t.Fatalf("Rank(%d, Select(%d, %d)) 应为 %d：%d", value, value, r, r-1, got)
			}
		}
	}
}

func TestLargeAlphabet(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	values := make([]uint32, 2000)
	for index := range values {
		values[index] = uint32(rng.Intn(1 << 20))
	}
	matrix := Build(values)
	for trial := 0; trial < 500; trial++ {
		left := rng.Intn(len(values))
		right := left + 1 + rng.Intn(len(values)-left)
		k := rng.Intn(right-left) + 1
		got, err := matrix.KthSmallest(left, right, k)
		if err != nil {
			t.Fatal(err)
		}
		if want := bruteKth(values, left, right, k); got != want {
			t.Fatalf("KthSmallest(%d, %d, %d) = %d；想要 %d", left, right, k, got, want)
		}
		value := values[rng.Intn(len(values))]
		position := rng.Intn(len(values) + 1)
		gotRank, err := matrix.Rank(value, position)
		if err != nil {
			t.Fatal(err)
		}
		if want := bruteRank(values, value, position); gotRank != want {
			t.Fatalf("Rank(%d, %d) = %d；想要 %d", value, position, gotRank, want)
		}
	}
	if _, err := matrix.Rank(1<<20, 100); !errors.Is(err, ErrValue) {
		t.Fatalf("value 超出值域应该报 ErrValue：%v", err)
	}
}

func BenchmarkQueries(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	values := make([]uint32, 1_000_000)
	for index := range values {
		values[index] = uint32(rng.Intn(256))
	}
	matrix := Build(values)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		position := rng.Intn(len(values) + 1)
		if _, err := matrix.Rank(97, position); err != nil {
			b.Fatal(err)
		}
		if _, err := matrix.Select(97, rng.Intn(100)+1); err != nil {
			b.Fatal(err)
		}
		left := rng.Intn(len(values))
		right := left + 1 + rng.Intn(len(values)-left)
		if _, err := matrix.KthSmallest(left, right, rng.Intn(right-left)+1); err != nil {
			b.Fatal(err)
		}
	}
}
