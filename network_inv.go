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
		downloadMutex.Lock()
		for _, hash := range payload.Items {
			hashStr := hex.EncodeToString(hash)
			if _, downloading := downloadingBlocks[hashStr]; !downloading {
				blocksToDownload[hashStr] = true
			}
		}
		downloadMutex.Unlock()
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
