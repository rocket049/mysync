package mycrypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"os"
)

const bits = 2048

func GenKeyPair(name string) error {
	var pubk rsa.PublicKey
	rng := rand.Reader
	prik, err := rsa.GenerateKey(rng, bits)
	if err != nil {
		panic(err)
	}
	pubk.N = prik.N
	pubk.E = prik.E
	// save private key
	var blk pem.Block
	blk.Type = "RSA PRIVATE KEY"
	file1 := name + ".key"
	fp, err := os.Create(file1)
	if err != nil {
		panic(err)
	}
	blk.Bytes = x509.MarshalPKCS1PrivateKey(prik)
	err = pem.Encode(fp, &blk)
	if err != nil {
		panic(err)
	}
	fp.Close()
	//save public key
	blk.Type = "RSA PUBLIC KEY"
	file1 = name + ".pub"
	fp, err = os.Create(file1)
	if err != nil {
		panic(err)
	}
	blk.Bytes = x509.MarshalPKCS1PublicKey(&pubk)
	err = pem.Encode(fp, &blk)
	if err != nil {
		panic(err)
	}
	fp.Close()
	return nil
}

func ReadPrivateKey(filename string) *rsa.PrivateKey {
	ke1, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil
	}
	block, _ := pem.Decode(ke1)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		log.Println("failed to decode PEM block containing private key")
		return nil
	}
	k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Println(err)
		return nil
	} else {
		return k
	}
}

func ReadPublicKey(filename string) *rsa.PublicKey {
	ke1, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil
	}
	block, _ := pem.Decode(ke1)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		log.Println("failed to decode PEM block containing public key")
		return nil
	}
	k, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		log.Println(err)
		return nil
	} else {
		return k
	}
}

func Encode(pubkeyfile, msg string) ([]byte, error) {
	pubk := ReadPublicKey(pubkeyfile)
	if pubk == nil {
		log.Println("error read publickey " + pubkeyfile)
		return nil, errors.New("error read publickey " + pubkeyfile)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, pubk, []byte(msg))
}

func Decode(prikeyfile string, ciphertext []byte) ([]byte, error) {
	prik := ReadPrivateKey(prikeyfile)
	if prik == nil {
		log.Println("err read privatekey " + prikeyfile)
		return nil, errors.New("err read privatekey " + prikeyfile)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, prik, ciphertext)
}

func EncodeWithKey(pubk *rsa.PublicKey, msg []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pubk, msg)
}

func DecodeWithKey(prik *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, prik, ciphertext)
}

func SignWithKey(prik *rsa.PrivateKey, msg []byte) ([]byte, error) {
	hashed := sha256.Sum256(msg)
	return rsa.SignPKCS1v15(rand.Reader, prik, crypto.SHA256, hashed[:])
}

func VerifyWithKey(pubk *rsa.PublicKey, msg, signature []byte) bool {
	hashed := sha256.Sum256(msg)
	err := rsa.VerifyPKCS1v15(pubk, crypto.SHA256, hashed[:], signature)
	if err != nil {
		log.Println("Error verify rsa:", err)
		return false
	} else {
		return true
	}
}
