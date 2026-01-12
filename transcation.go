package main

// 交易
type Trainscation struct {
	ID   []byte     // 当前交易id
	Vin  []TxInput  // 上一个交易的输入
	Vout []TxOutput // 当前交易的输出
}
