package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
)

// 交易
type Transaction struct {
	ID   []byte     // 当前交易id
	Vin  []TxInput  // 上一个交易的输入
	Vout []TxOutput // 当前交易的输出
}

// Serialize 序列化完整交易（包含 ID）
func (tx Transaction) Serialize() []byte {
	buf := new(bytes.Buffer)
	// ID 固定 32 字节
	if len(tx.ID) != 32 {
		// 如果 ID 长度不对，补零（通常不会发生）
		tmp := make([]byte, 32)
		copy(tmp, tx.ID)
		buf.Write(tmp)
	} else {
		buf.Write(tx.ID)
	}
	// 输入数量
	WriteVarInt(buf, uint64(len(tx.Vin)))
	for _, in := range tx.Vin {
		_ = in.Serialize(buf)
	}
	// 输出数量
	WriteVarInt(buf, uint64(len(tx.Vout)))
	for _, out := range tx.Vout {
		_ = out.Serialize(buf)
	}
	return buf.Bytes()
}

// DeserializeTransaction 从字节流反序列化交易
func DeserializeTransaction(data []byte) (*Transaction, error) {
	r := bytes.NewReader(data)
	id := make([]byte, 32)
	if _, err := io.ReadFull(r, id); err != nil {
		return nil, err
	}
	vinCount, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	vin := make([]TxInput, 0, vinCount)
	for i := uint64(0); i < vinCount; i++ {
		in, err := DeserializeTxInput(r)
		if err != nil {
			return nil, err
		}
		vin = append(vin, *in)
	}
	voutCount, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	vout := make([]TxOutput, 0, voutCount)
	for i := uint64(0); i < voutCount; i++ {
		out, err := DeserializeTxOutput(r)
		if err != nil {
			return nil, err
		}
		vout = append(vout, *out)
	}
	return &Transaction{
		ID:   id,
		Vin:  vin,
		Vout: vout,
	}, nil
}

// Hash 计算交易 ID（用于签名前）—— 必须序列化时不包含 ID 字段
func (tx *Transaction) Hash() []byte {
	// 复制交易，清空 ID
	txCopy := *tx
	txCopy.ID = make([]byte, 32) // 32 字节零值
	// 序列化副本
	data := txCopy.Serialize()
	hash := sha256.Sum256(data)
	return hash[:]
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

	pubKeyHash := HashPubKey(wallet.PublicKey)                                      // 发送者公钥
	balance, validOutputs, err := UTXOSet.GetUnSpendableOutputs(pubKeyHash, amount) // 收集发送者UTXO的balance直至满足amount
	if err != nil {
		log.Panic(err)
	}
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

	tx := Transaction{nil, inputs, outputs}                               // 完成交易结构体组装
	tx.ID = tx.Hash()                                                     // 生成交易的id
	signerr := UTXOSet.BlockChain.SignTransaction(&tx, wallet.PrivateKey) // 交易和私钥去签名
	if signerr != nil {
		log.Printf("Sign transaction error: %v", err)
		// 可根据需要返回 nil 或 panic直接崩溃，这里我们只打印日志
	}

	return &tx //交易数据已经完全组装完成内容无空
}

// 签名（当前交易里，我的私钥，我上一次的交易）
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) error {
	if tx.IsCoinbase() { // 验证时否时coinbase交易
		return nil
	}
	// 验证当前交易的所有输入所引用的前序交易是否都存在
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			return fmt.Errorf("previous transaction not found")
		}
	}
	// 当前交易复制一份（为了签名，input里签名和公钥必须是nil才好签名）
	txCopy := tx.TrimmedCopy()
	for inID, vin := range txCopy.Vin {
		// 获取上一个交易的输出
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		// 边界检查（正常时uint32，我用的uint64所以需要检查）
		if int(vin.Vout) >= len(prevTx.Vout) {
			return fmt.Errorf("vout index out of range")
		}
		// 复制交易设置空签名和公钥
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		// 交易进行hash用于签名
		dataToSign := txCopy.Hash()
		// 椭圆曲线签名的核心逻辑
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, dataToSign)
		if err != nil {
			return fmt.Errorf("ecdsa sign: %w", err)
		}
		// 签名成功
		signature := append(r.Bytes(), s.Bytes()...)
		// 签名成功，赋值给交易，为了安全把副本制空
		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}
	return nil
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

	// 清空 ID，因为签名时不应依赖自身哈希
	txCopy := Transaction{nil, inputs, outputs}

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
		// 边界检查
		if int(vin.Vout) >= len(prevTx.Vout) {
			log.Panic("ERROR: Vout index out of range")
		}
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

		dataToVerify := txCopy.Hash() // 使用交易副本的哈希

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, dataToVerify, &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}
