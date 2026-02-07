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

// 加密地址
func Base58Encode(input []byte) []byte {
	var result []byte

	x := big.NewInt(0).SetBytes(input)

	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	// https://en.bitcoin.it/wiki/Base58Check_encoding#Version_bytes
	if input[0] == 0x00 {
		result = append(result, b58Alphabet[0])
	}

	ReverseBytes(result)

	return result
}
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
