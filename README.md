# wavelet-matrix

小波矩阵：rank / select / 区间第 k 小，只用 Go 标准库。

```
go test ./...
go run ./cmd/wavelet-matrix --text banana --rank 'a 3' --select 'a 2' --kth '0 2 2'
```

## 口径（README 为准）

- **rank 只看前缀**：`--text banana --rank 'a 3'` 按口径是 `rank=1`（前 3 个字节 b、a、n 里只有 1 个 a），
  现在直接给整串的出现次数 `rank=3`。
- **select 是 rank 的逆运算**：`--select 'a 2'`（第 2 个 a，1 起）按口径是 `select=3`，
  现在报「select 还没有实现」。
- **区间第 k 小只看区间**：`--kth '0 2 2'`（banana 前两个字节是 b、a，第 2 小是 b）按口径是 `kth=98`，
  现在忽略区间、在整串排序结果上取，给 `kth=97`。输出的是字符码的十进制值。
- 结构：按符号的每一位分层存位向量和前缀 rank 表，rank/select/kth 都不许回扫原序列。
- 不变量：`Rank(v, n) == v 的出现次数`；`Select(v, r)` 是第 r 次（1 起）出现的位置；
  `KthSmallest` 与「把区间拷出来排序」的结果逐项一致。
- 规模：10^6 个符号、rank/select/kth 各 10^5 次查询要在 2 秒内完成，内存 O(n·log σ)；不许每次查询重排区间。

## 现在的行为

- `Build` 只存原序列和一份排好序的副本，没有位向量，也没有前缀 rank 表。
- `Rank` 在排序副本上二分，返回整串的出现次数，完全忽略 `position`。
- `Select` 返回 `ErrNoSelect`。
- `KthSmallest` 直接在整串排序结果上取第 k 个，区间参数只用来做合法性检查。

## 计划新增的能力

```
--text 文本           输入序列（按字节）
--pseudo N            生成 N 个伪随机符号（0..3）
--rank 'v pos'        前缀 rank
--select 'v rank'     第 rank 次出现的位置
--kth 'l r k'         区间第 k 小
```

## 输出

```
rank=1
select=3
kth=98
```

## 目录

```
wavelet.go               建表与三种查询
cmd/wavelet-matrix       命令行入口
wavelet_test.go          go test 用例
```
