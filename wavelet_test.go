package waveletmatrix

import "testing"

// sample 是 "banana" 的字节序列。
func sample() *Matrix {
	return Build([]uint32{98, 97, 110, 97, 110, 97})
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

func TestRankOverWholeSequence(t *testing.T) {
	matrix := sample()
	if got := matrix.Rank(97, matrix.Length()); got != 3 {
		t.Fatalf("整串里 a 应该有 3 个：%d", got)
	}
	if got := matrix.Rank(122, matrix.Length()); got != 0 {
		t.Fatalf("没出现过的符号应该是 0：%d", got)
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

func TestKthRejectsBadRange(t *testing.T) {
	matrix := sample()
	if _, err := matrix.KthSmallest(3, 3, 1); err == nil {
		t.Fatal("空区间应该报错")
	}
	if _, err := matrix.KthSmallest(0, 6, 0); err == nil {
		t.Fatal("k 从 1 开始，0 应该报错")
	}
}

func TestEmptyMatrix(t *testing.T) {
	matrix := Build(nil)
	if matrix.Length() != 0 {
		t.Fatalf("空序列长度应该是 0：%d", matrix.Length())
	}
	if got := matrix.Rank(1, 0); got != 0 {
		t.Fatalf("空序列的 rank 应该是 0：%d", got)
	}
}
