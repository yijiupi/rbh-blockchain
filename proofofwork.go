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

const targetBits = 16

var maxNonce uint64 = math.MaxUint64

// 工作量证明POW
type ProofOfWork struct {
	block  *Block   // 区块
	target *big.Int // 目标预值
}

// 新区块的工作量证明
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)                  // 目标预值
	target.Lsh(target, uint(256-targetBits)) // 左移位运算
	pow := &ProofOfWork{b, target}           // 组装工作量证明结构体

	return pow
}

// 获取随机数和区块hash
func (pow *ProofOfWork) Run() (uint64, []byte) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint64 = 0

	fmt.Printf("Mining a new block")
	// 循环挖矿获取正确的随机数
	for nonce < maxNonce {
		data := pow.prepareData(nonce) // 准备数据

		hash = sha256.Sum256(data) // 得到hash
		if math.Remainder(float64(nonce), 100000) == 0 {
			fmt.Printf("\r%x", hash)
		}
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break // 挖矿成功（终止）
		} else {
			nonce++
		}
	}
	fmt.Print("\n\n")

	return nonce, hash[:]
}

// 准备数据（工作量证明成功后需要存储的数据）
func (pow *ProofOfWork) prepareData(nonce uint64) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,      // 上一个交易的hash
			pow.block.HashTransactions(), // 当前区块的所有交易（在默克尔树上）
			IntToHex(pow.block.Timestamp),
			IntToHex(int64(targetBits)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)

	return data
}

// 将整数转换为字节切片（二进制表示）
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
