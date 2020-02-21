package main

import (
	"encoding/json"
	"io/ioutil"
)

func main() {
	map1, err := UpdateFileMap(".")
	if err != nil {
		panic(err)
	}
	data, err := json.Marshal(map1)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("_desc.json", data, 0644)
}
