package main

import (
	"mysync/mysyncd/mycrypto"
	"flag"
)

func main() {
	var k = flag.String("k", "mykey", "[-k name]:generate rsa key pair ,save in name.pub and name.key")
	flag.Parse()
	//generate key pair
	name1 := *k
	if len(name1) == 0 {
		panic("no name. please use: -h")
	}
	err := mycrypto.GenKeyPair(name1)
	if err != nil {
		panic(err)
	}
	return
}

