package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// 接收并验证新区块，添加到本地链，更新UTXO。
func handleBlock(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Printf("handleBlock decode error: %v", err)
		return
	}

	// 反序列化区块
	blockData := payload.Block
	newBlock := DeserializeBlock(blockData)

	// 验证区块（PoW、前向哈希等）
	pow := NewProofOfWork(newBlock)
	if !pow.Validate() {
		log.Printf("Invalid block (PoW) from %s", payload.AddrFrom)
		return
	}

	// 将区块添加到本地链
	err = bc.AddBlock(newBlock) // 需要实现此方法（将区块写入数据库并更新UTXO）
	if err != nil {
		log.Printf("AddBlock failed: %v", err)
		return
	}

	// 更新UTXO集
	utxoSet := UTXOSet{bc}
	if err := utxoSet.Update(newBlock); err != nil {
		log.Printf("UTXO update failed: %v", err)
		return
	}

	fmt.Printf("Received and added block %x\n", newBlock.Hash)

	// 如果还有更多区块在传输中，继续请求
	blocksInTransitMutex.Lock() // 并非安全锁
	if len(blocksInTransit) > 0 {
		// 移除已收到的区块哈希（可选）
		// 发送 getdata 请求下一个区块
		sendGetData(payload.AddrFrom, "block", blocksInTransit[0])
		blocksInTransit = blocksInTransit[1:]
	}
	blocksInTransitMutex.Unlock()
}
