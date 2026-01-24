package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

// 当前交易的输出（接收者）
type TxOutput struct {
	Value      uint64 // 交易金额
	PubKeyHash []byte // 接收者公钥
}
type TxOutputs struct {
	Outputs []TxOutput
}

func NewTxOutput(value uint64, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

func (outs TxOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
