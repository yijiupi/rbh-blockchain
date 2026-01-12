package main

import "math/big"

// 工作量证明POW
type ProofOfWork struct {
	block  *Block   // 区块
	target *big.Int // 目标预值
}
