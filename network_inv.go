package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"log"
)

func handleInv(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Printf("handleInv decode error: %v", err)
		return
	}

	switch payload.Type {
	case "block":
		// 对方告知有新区块，检查本地是否已存在
		blocksInTransitMutex.Lock() // 并非安全锁
		blocksInTransit = payload.Items
		if len(blocksInTransit) == 0 {
			blocksInTransitMutex.Unlock()
			return
		}
		// 请求第一个区块
		sendGetData(payload.AddrFrom, "block", blocksInTransit[0])
		blocksInTransit = blocksInTransit[1:]
		blocksInTransitMutex.Unlock()

	case "tx":
		// 对方告知有新交易，检查内存池是否已存在，若不存在则请求
		memPoolMutex.RLock()
		for _, txID := range payload.Items {
			txHash := hex.EncodeToString(txID)
			if _, ok := memPool[txHash]; !ok {
				sendGetData(payload.AddrFrom, "tx", txID)
			}
		}
		memPoolMutex.RUnlock()
	}
}
