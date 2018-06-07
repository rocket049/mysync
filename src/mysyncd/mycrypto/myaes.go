package mycrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
)

func AES256Encode(key, msg []byte) []byte {
	if len(key) != 32 {
		log.Println("AES256Encode: key length must be 32. Now:", len(key))
		return nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println("AES256Encode:", err)
		return nil
	}
	var iv = make([]byte, block.BlockSize())
	io.ReadFull(rand.Reader, iv)
	aesx := cipher.NewCTR(block, iv)
	var res = make([]byte, len(msg))
	aesx.XORKeyStream(res, msg)
	buf1 := bytes.NewBuffer(iv)
	buf1.Write(res)
	return buf1.Bytes()
}

func AES256Decode(key, msg []byte) []byte {
	if len(key) != 32 {
		log.Println("AES256Decode: key length must be 32. Now:", len(key))
		return nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println("AES256Decode:", err)
		return nil
	}
	bs := block.BlockSize()
	aesx := cipher.NewCTR(block, msg[:bs])
	var res = make([]byte, len(msg)-bs)
	aesx.XORKeyStream(res, msg[bs:])
	return res
}
