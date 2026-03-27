package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"log"
)

func handleGetData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getData

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Printf("handleGetData decode error: %v", err)
		return
	}

	switch payload.Type {
	case "block":
		// 根据 ID 查找区块
		foundBlock, err := bc.FindBlock(payload.ID) // 改名 foundBlock
		if err != nil {
			log.Printf("Block not found: %x", payload.ID)
			return
		}
		// 发送 block 消息
		blockPayload := block{
			AddrFrom: nodeAddress,
			Block:    foundBlock.Serialize(),
		}
		data := gobEncode(blockPayload)
		requestMsg := append(commandToBytes("block"), data...)
		sendData(payload.AddrFrom, requestMsg)

	case "tx":
		// 从内存池查找交易
		foundTx, ok := memPool[hex.EncodeToString(payload.ID)] // 改名 foundTx
		if !ok {
			log.Printf("Transaction not found in mempool: %x", payload.ID)
			return
		}
		// 发送 tx 消息
		txPayload := tx{
			AddFrom:     nodeAddress,
			Transaction: foundTx.Serialize(),
		}
		data := gobEncode(txPayload)
		requestMsg := append(commandToBytes("tx"), data...)
		sendData(payload.AddrFrom, requestMsg)
	}
}
