package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"mysyncd/conf"
	"mysyncd/files"
	"mysyncd/mycrypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
)

type Args struct {
	Valid   []byte //crypto []byte with rsa(first time), aes
	Msg     []byte
	FileMap map[string]files.FileDesc
}
type Reply struct {
	Valid   []byte //crypto []byte with rsa(first time),aes
	UpList  []string
	DelList []string
}

type Ctlrpc int

var host, root string
var pri_key_dir = "/home/ufhz/conf/"
var conf_file_dir = "/home/ufhz/conf/"

func main() {
	set_win_dir()
	var cfg = conf.ReadJSON(path.Join(conf_file_dir, "local.json"))
	if cfg == nil {
		panic("No config local.json")
	}
	root = cfg["root"]
	host = cfg["host"]
	var name1 = cfg["key"]
	client, err := rpc.DialHTTPPath("tcp", host, "/ctlrpc")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	key1 := login(client, name1)
	if key1 == nil {
		panic("login fail")
	}

	uplist, key1 := syncDel(client, name1, key1)
	if key1 == nil {
		panic("sync and del fail")
	}
	if len(uplist) > 0 {
		for _, v := range uplist {
			log.Println(v)
		}
		key1 = uploadList(uplist, name1, key1)
		if key1 == nil {
			panic("upload list fail")
		}
	}

	err = logout(client, name1, key1)
	if err != nil {
		panic(err)
	}
}

func login(rpc1 *rpc.Client, name1 string) []byte {
	var pri_key = path.Join(pri_key_dir, fmt.Sprintf("%v.key", name1))
	prik := mycrypto.ReadPrivateKey(pri_key)
	if prik == nil {
		log.Println("error ReadPrivateKey :", pri_key)
		return nil
	}
	var r1 = make([]byte, 32)
	io.ReadFull(rand.Reader, r1)
	buf1 := bytes.NewBuffer(r1)
	buf1.WriteString(name1)
	valid, err := mycrypto.SignWithKey(prik, buf1.Bytes())
	if err != nil {
		log.Println(err)
		return nil
	}
	var arg Args
	arg.Valid = valid
	arg.Msg = buf1.Bytes()
	var reply []byte
	err = rpc1.Call("Ctlrpc.Login", &arg, &reply)
	if err != nil {
		log.Println(err)
		return nil
	}
	key1, err := mycrypto.DecodeWithKey(prik, reply)
	if err == nil {
		return key1
	} else {
		log.Println(err)
		return nil
	}
}

func syncDel(rpc1 *rpc.Client, name1 string, k []byte) (uplist []string, retk []byte) {
	local_list, err := files.GetFileMap(root)
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	//verify message
	var b1 = make([]byte, 32)
	io.ReadFull(rand.Reader, b1)
	buf1 := bytes.NewBuffer(b1)
	buf1.WriteString(name1)
	//crypto message with received key
	valid := mycrypto.AES256Encode(k, buf1.Bytes())
	if valid == nil {
		return nil, nil
	}
	var arg = Args{valid, buf1.Bytes(), local_list}
	var reply Reply
	err = rpc1.Call("Ctlrpc.SyncDel", &arg, &reply)
	k1 := mycrypto.AES256Decode(k, reply.Valid)
	if err == nil {
		for i, v := range reply.DelList {
			log.Printf("DEL %v: %v\n", i+1, v)
		}
		return reply.UpList, k1
	} else {
		log.Println(err)
		return nil, nil
	}
}

//upload list and return new key
func uploadList(uplist []string, name1 string, k []byte) []byte {
	//create zip file
	home := os.Getenv("HOME")
    if len(home)==0{
        //windows
        home = "/"
    }
	dir1 := path.Join( home, ".tmp")
	os.MkdirAll(dir1, os.ModePerm)
	filename1 := path.Join(dir1, "up.zip")
	fp, err := os.Create(filename1)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer os.Remove(filename1)
	zf := zip.NewWriter(fp)
	os.Chdir(root)
	for _, p := range uplist {
		p1 := p
		fp1, err := os.Open(p1)
		if err != nil {
			zf.Close()
			fp.Close()
			log.Println(err)
			return nil
		}
		st1, err := fp1.Stat()
		if err != nil {
			zf.Close()
			fp.Close()
			log.Println(err)
			return nil
		}
		fh, err := zip.FileInfoHeader(st1)
		fh.Name = p1
		//log.Println(fh.Name, st1.Name())
		if err != nil {
			zf.Close()
			fp.Close()
			log.Println(err)
			return nil
		}
		fh.Method = zip.Deflate
		fh.Modified = st1.ModTime()
		f, err := zf.CreateHeader(fh)
		if err != nil {
			zf.Close()
			fp.Close()
			log.Println(err)
			return nil
		}
		if st1.IsDir() {
			fp1.Close()
		} else {
			io.Copy(f, fp1)
			fp1.Close()
		}
	}
	zf.Close()
	fp.Close()
	//valid and upload
	var b1 = make([]byte, 32)
	io.ReadFull(rand.Reader, b1)
	buf1 := bytes.NewBuffer(b1)
	buf1.WriteString(name1)
	msg := buf1.Bytes()
	valid := mycrypto.AES256Encode(k, msg)
	if valid == nil {
		log.Println("error AES256Encode")
		return nil
	}
	var url = fmt.Sprintf("http://%v/upload", host)
	ret := files.PostFile(filename1, url, &files.MyValid{valid, msg})
	if ret == nil {
		log.Println("error upload zip")
		return nil
	}
	rets := string(ret)
	res, err := hex.DecodeString(rets)
	if err != nil {
		log.Println(err)
		return nil
	}
	return mycrypto.AES256Decode(k, res)
}

func logout(rpc1 *rpc.Client, name1 string, k []byte) error {
	var b1 = make([]byte, 32)
	io.ReadFull(rand.Reader, b1)
	buf1 := bytes.NewBuffer(b1)
	buf1.WriteString(name1)
	valid := mycrypto.AES256Encode(k, buf1.Bytes())
	if valid == nil {
		return errors.New("logout error AES256Encode")
	}
	var arg = Args{valid, buf1.Bytes(), nil}
	var reply string = "logout: No reply"
	err := rpc1.Call("Ctlrpc.Logout", &arg, &reply)
	log.Println(reply)
	return err
}

func set_win_dir() {
	if len(os.Getenv("windir")) == 0 {
		home := os.Getenv("HOME")
		pri_key_dir = path.Join(home, "mysync/")
		conf_file_dir = path.Join(home, "mysync/")
		return
	}
	log.Println("OS: Windows")
	exe1, _ := os.Executable()
	dir1 := filepath.Dir(exe1)
	conf1 := filepath.Join(dir1, "conf/")
	pri_key_dir = conf1
	conf_file_dir = conf1
}