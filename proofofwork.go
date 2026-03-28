package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

/*
*
如果 targetBits = 36，平均尝试次数是 2^36 ≈ 687 亿次，即使使用高端 CPU 每秒 1 亿次，也需要约 1.9 小时。
如果targetBits = 16：平均尝试次数是 2^16 ≈ 65536次，通常在 0.01 秒到 0.1 秒 之间。
意味着哈希值需要小于 2^(256-16)，也就是说哈希值前 16 位必须是 0。概率是 1/2^16 ≈ 1/65,536。
比特币通过动态的修改targetBits来控制出块时间，新目标值 = 旧目标值 × (实际耗时 / 期望耗时)，并限制 4 倍以内，防止剧烈波动
*
*/
const targetBits = 16

var maxNonce uint64 = math.MaxUint64

// ProofOfWork 工作量证明
type ProofOfWork struct {
	block          *Block
	target         *big.Int
	preparedPrefix []byte // 缓存预计算不变部分（不含 nonce）
}

// NewProofOfWork 创建工作量证明，缓存预计算不变部分
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	// 预计算不变部分：PrevBlockHash + HashTransactions + Timestamp + targetBits
	prefix := bytes.Join(
		[][]byte{
			b.PrevBlockHash,
			b.HashTransactions(),
			IntToHex(b.Timestamp),
			IntToHex(int64(targetBits)),
		},
		[]byte{},
	)

	pow := &ProofOfWork{b, target, prefix}
	return pow
}

// Run 开始挖矿，返回 nonce 和区块哈希
func (pow *ProofOfWork) Run() (uint64, []byte) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint64 = 0

	fmt.Printf("Mining a new block")
	for nonce < maxNonce {
		data := pow.prepareData(nonce) // 组合当前 nonce 与区块数据
		hash = sha256.Sum256(data)     // hash 是 [32]byte 类型（数组）
		if nonce%100000 == 0 {
			// 每 10 万次打印一次当前哈希（进度提示，避免程序看起来像卡死了一样）在大多数机器上大约每几秒到几十秒打印一次
			fmt.Printf("\r%x", hash) //
		}
		hashInt.SetBytes(hash[:]) // 传入hash切片类型，转为大数
		// 返回 -1 表示 hashInt < pow.target | 0 表示相等 | 1 表示 hashInt > pow.target
		if hashInt.Cmp(pow.target) == -1 {
			break // 找到有效 nonce，挖矿成功
		} else {
			nonce++
		}
	}
	fmt.Print("\n\n")

	return nonce, hash[:]
}

// 准备数据 预计算不变部分 + nonce
func (pow *ProofOfWork) prepareData(nonce uint64) []byte {
	data := bytes.Join(
		[][]byte{
			pow.preparedPrefix,
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}

// IntToHex 将 int64 转换为大端字节序
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// Validate 验证区块 PoW 是否正确
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1
	return isValid

}
