package main

import "github.com/boltdb/bolt"

// 区块链
type BlockChain struct {
	tip []byte   // 新区块的hash
	db  *bolt.DB // 数据库
}
