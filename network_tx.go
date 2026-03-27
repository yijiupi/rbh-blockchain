package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"log"
)

func handleTx(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Printf("handleTx decode error: %v", err)
		return
	}

	// 反序列化交易
	var transaction Transaction
	decTx := gob.NewDecoder(bytes.NewReader(payload.Transaction))
	err = decTx.Decode(&transaction)
	if err != nil {
		log.Printf("Transaction deserialize error: %v", err)
		return
	}

	// 验证交易（签名、UTXO等）
	ok, err := bc.VerifyTransaction(&transaction)
	if err != nil {
		log.Printf("VerifyTransaction error: %v", err)
		return
	}
	if !ok {
		log.Printf("Invalid transaction received")
		return
	}

	// 加入内存池
	txID := hex.EncodeToString(transaction.ID)
	memPool[txID] = transaction

	// 广播给其他节点（可选，避免循环广播）
	for _, node := range knownNodes {
		if node != payload.AddFrom && node != nodeAddress {
			sendTx(node, &transaction)
		}
	}
}
