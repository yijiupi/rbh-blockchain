package main

import (
	"bytes"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// 解密地址
func Base58Decode(input []byte) []byte {
	result := big.NewInt(0) // 定义大数

	for _, b := range input {
		charIndex := bytes.IndexByte(b58Alphabet, b)     // b在b58Alphabet中的位置
		result.Mul(result, big.NewInt(58))               // 乘法58
		result.Add(result, big.NewInt(int64(charIndex))) // 加法，加charIndex
	}

	decoded := result.Bytes() // 大数转字节

	if input[0] == b58Alphabet[0] { // 地址的第一位和
		decoded = append([]byte{0x00}, decoded...) // ...展开切片
	}
	return decoded
}
