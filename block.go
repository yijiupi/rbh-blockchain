package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

// / 区块
type Block struct {
	Height        uint64          // 区块高度
	Nonce         uint64          // 随机数
	Trainscations []*Trainscation // 交易列表
	PrevBlockHash []byte          // 上一个区块的hash
	Hash          []byte          // 当前区块的hash
	Timestamp     int64           // 时间戳
}

// 反序列化一个区块
func DeserializeBlock(d []byte) *Block {
	var block Block
	// 使用gob解码器
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
