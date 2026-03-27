package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

func handleGetBlocks(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getBlocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Printf("handleGetBlocks decode error: %v", err)
		return
	}

	// 获取本地所有区块哈希（或从最新区块开始返回一批）
	// 简化：返回本地所有区块哈希（实际应返回一定数量）
	var blocksHashes [][]byte
	bci := bc.Iterator()
	for {
		block := bci.Next()
		blocksHashes = append(blocksHashes, block.Hash)
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	// 构造 inv 消息返回
	invPayload := inv{
		AddrFrom: nodeAddress,
		Type:     "block",
		Items:    blocksHashes,
	}
	data := gobEncode(invPayload)
	requestMsg := append(commandToBytes("inv"), data...)
	sendData(payload.AddrFrom, requestMsg)
}
