package main

/// 区块
type Block struct {
	Height        uint64          // 区块高度
	Nonce         uint64          // 随机数
	Trainscations []*Trainscation // 交易列表
	PrevBlockHash []byte          // 上一个区块的hash
	Hash          []byte          // 当前区块的hash
	Timestamp     int64           // 时间戳
}
