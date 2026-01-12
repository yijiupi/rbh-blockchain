package main

// 当前交易的输入（拥有者）
type TxInput struct {
	Txid      []byte // 上一个交易的id
	Vout      uint64 // 上一个交易的输出
	Signature []byte // 上一个交易的签名
	PubKey    []byte // 公钥（用于验证签名）
}
