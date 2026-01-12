package main

// 当前交易的输出（接收者）
type TxOutput struct {
	Value      uint   // 交易金额
	PubKeyHash []byte // 接收者公钥
}
