package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

// 交易
type Transaction struct {
	ID   []byte     // 当前交易id
	Vin  []TxInput  // 上一个交易的输入
	Vout []TxOutput // 当前交易的输出
}

// 把交易内容整体做hash
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	//txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// 把交易内容整体序列化
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// 判断一个交易是否是创币交易
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == 0
}
