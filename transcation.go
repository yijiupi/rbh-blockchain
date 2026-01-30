package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
)

// 交易
type Transaction struct {
	ID   []byte     // 当前交易id
	Vin  []TxInput  // 上一个交易的输入
	Vout []TxOutput // 当前交易的输出
}

// 把交易内容整体做hash
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	//txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// 把交易内容整体序列化
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// 判断一个交易是否是创币交易
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == 0
}

// 新交易（UTXO转账）
func NewUTXOTransaction(wallet *Wallet, to string, amount uint64, UTXOSet *UTXOSet) *Transaction {
	/*
			inputs = []TxInput{
		    {
		        Txid:      []byte{0x12, 0x34, 0x56, ...},  // "tx001"的32字节哈希
		        Vout:      1,                              // 引用输出索引1
		        Signature: nil,
		        PubKey:    []byte{0xAA, 0xBB, 0xCC, ...},  // 发送者公钥
		    },
		    {
		        Txid:      []byte{0x12, 0x34, 0x56, ...},  // 同一个"tx001"交易
		        Vout:      2,                              // 引用输出索引2
		        Signature: nil,
		        PubKey:    []byte{0xAA, 0xBB, 0xCC, ...},  // 同一个发送者公钥
		    },
		}
		outputs = []TxOutput{
		    {
		        Value:      amount,            // 发送给接收者的金额
		        PubKeyHash: toPubKeyHash,      // 接收者的公钥哈希
		    },
		    {
		        Value:      balance - amount,  // 找零金额（退回给发送者）
		        PubKeyHash: fromPubKeyHash,    // 发送者的公钥哈希
		    },
		}
	*/
	var inputs []TxInput // 组装TxInput列表，这里包含所有准备消费Vout
	var outputs []TxOutput

	pubKeyHash := HashPubKey(wallet.PublicKey)                                 // 发送者公钥
	balance, validOutputs := UTXOSet.GetUnSpendableOutputs(pubKeyHash, amount) // 收集发送者UTXO的balance直至满足amount

	if balance < amount {
		log.Panic("ERROR: Not enough funds") // 余额不足
	}

	// balance余额来源的output列表（txid:outIndex）
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TxInput{txID, uint64(out), nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	from := fmt.Sprintf("%s", wallet.GetAddress())      // 发送者
	outputs = append(outputs, *NewTxOutput(amount, to)) // 为了将to的string类型转为公钥的byte类型
	if balance > amount {                               // balance-amount剩余的金额记录到utxo里面
		outputs = append(outputs, *NewTxOutput(balance-amount, from)) // 为了将from的string类型转为公钥的byte类型
	}

	tx := Transaction{nil, inputs, outputs}                    // 完成交易结构体组装
	tx.ID = tx.Hash()                                          // 生成交易的id
	UTXOSet.BlockChain.SignTransaction(&tx, wallet.PrivateKey) // 交易和私钥去签名

	return &tx //交易数据已经完全组装完成内容无空
}

// 签名（当前交易里，我的私钥，我上一次的交易）
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() { // 验证时否时coinbase交易
		return
	}
	// 当前交易的vin.Txid是上一个交易prevTXs的key
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil { // 得到上一个交易是否存在
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy() // 当前交易复制一份（为了签名，input里签名和公钥必须是nil才好签名）
	// 循环当前交易副本的input
	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)] // 当前交易成为上一个交易的
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		dataToSign := fmt.Sprintf("%x\n", txCopy)
		// 椭圆曲线签名的核心逻辑
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign)) // 私钥和交易进行签名
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature // 得到签名给tx赋值，tx最终所有参数都组装完成
		txCopy.Vin[inID].PubKey = nil
	}
}

// 复制当前的交易
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, vin := range tx.Vin { // tx.vin复制一份去掉公钥和签名
		inputs = append(inputs, TxInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout { // 当交易的输出复制一份
		outputs = append(outputs, TxOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

// Verify verifies signatures of Transaction inputs
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}
