package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"log"
)

// handleTx 处理收到的交易消息
func handleTx(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload tx

	// 1. 解析消息，提取交易数据
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	if err := dec.Decode(&payload); err != nil {
		log.Printf("handleTx decode error: %v", err)
		return
	}

	// 2. 反序列化交易对象
	var transaction Transaction
	if err := gob.NewDecoder(bytes.NewReader(payload.Transaction)).Decode(&transaction); err != nil {
		log.Printf("Transaction deserialize error: %v", err)
		return
	}

	txID := hex.EncodeToString(transaction.ID)

	// 3. 原子性检查并标记已广播，避免重复处理（TOCTOU 防护）
	//    - 使用写锁保护“检查+写入”操作，确保不会有两个 goroutine 同时判定未广播并同时进入后续流程。
	broadcastedTxsMutex.Lock()
	if broadcastedTxs[txID] {
		broadcastedTxsMutex.Unlock()
		return // 已处理过，直接返回
	}
	broadcastedTxs[txID] = true // 标记为已广播，后续相同交易不再重复处理
	broadcastedTxsMutex.Unlock()

	// 4. 验证交易（签名、UTXO 等）
	//    注意：验证可能耗时，此时已释放广播锁，不影响其他消息的处理。
	ok, err := bc.VerifyTransaction(&transaction)
	if err != nil {
		log.Printf("VerifyTransaction error: %v", err)
		// 验证出错，清除广播标记，允许未来重新处理（如网络重传）
		broadcastedTxsMutex.Lock()
		delete(broadcastedTxs, txID)
		broadcastedTxsMutex.Unlock()
		return
	}
	if !ok {
		log.Printf("Invalid transaction received")
		// 无效交易，清除标记并丢弃
		broadcastedTxsMutex.Lock()
		delete(broadcastedTxs, txID)
		broadcastedTxsMutex.Unlock()
		return
	}

	// 5. 将交易加入内存池（避免重复存储）
	memPoolMutex.Lock()
	if _, exists := memPool[txID]; !exists {
		memPool[txID] = transaction
	}
	memPoolMutex.Unlock()

	// 6. 广播给其他节点（使用快照避免长时间持锁）
	//    先复制 knownNodes 快照，释放锁后再遍历广播，防止阻塞网络接收。
	knownNodesMutex.RLock()
	nodes := make([]string, len(knownNodes))
	copy(nodes, knownNodes)
	knownNodesMutex.RUnlock()

	for _, node := range nodes {
		// 避免回环：不发给消息来源节点，也不发给自己
		if node != payload.AddFrom && node != nodeAddress {
			sendTx(node, &transaction)
		}
	}
}
