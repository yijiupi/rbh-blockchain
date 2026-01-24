package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

// / 区块
type Block struct {
	Height        uint64         // 区块高度
	Nonce         uint64         // 随机数
	Transactions  []*Transaction // 交易列表
	PrevBlockHash []byte         // 上一个区块的hash
	Hash          []byte         // 当前区块的hash
	Timestamp     int64          // 时间戳
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

// 创建新区块
func NewBlock(transactions []*Transaction, prevBlockHash []byte, height uint64) *Block {
	block := &Block{height, 0, transactions, prevBlockHash, []byte{}, time.Now().Unix()} // 初始化区块
	pow := NewProofOfWork(block)                                                         // 工作量证明
	nonce, hash := pow.Run()                                                             // 获取随机数和区块hash
	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

// 创建创世区块
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0)
}

// 当前区块的所有交易（在默克尔树上，所有交易都加密合并到了根节点的交易数据集合中）
func (b *Block) HashTransactions() []byte {
	var transactions [][]byte // 当前区块交易列表结合
	// 循环当前区块的所有交易序列化集合
	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}
	mTree := NewMerkleTree(transactions) // 把当前交易的数据加入默克尔树，得到根节点

	return mTree.RootNode.Data
}

// 将区块结构序列化为字节数组
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result) // 创建一个 gob 编码器（Go binary encoder）

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}
