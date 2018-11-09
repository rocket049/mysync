package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mysync/mysyncd/conf"
	"mysync/mysyncd/files"
	"mysync/mysyncd/mycrypto"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"time"
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
type AppendData struct {
	Key  int
	Name string
	Gz   []byte
}

type Ctlrpc int

var host, root string
var pri_key_dir = "conf/"
var conf_file_dir = "conf/"

func main() {
	var conf1 = flag.String("conf", "local", "[-conf name] select special config file 'name.json'")
	var mode1 = flag.String("mode", "rpc", "[-mode rpc/http] 纯rpc模式/rpc(安全连接tls)、http混合模式")
	flag.Parse()

	set_win_dir()

	var cfg = conf.ReadJSON(path.Join(conf_file_dir, *conf1+".json"))
	if cfg == nil {
		panic("No config " + *conf1 + ".json")
	}
	root = cfg["root"]
	host = cfg["host"]
	var name1 = cfg["key"]
	var client *rpc.Client
	var err error
	if *mode1 == "http" {
		fmt.Println("http mode")
		client, err = rpc.DialHTTPPath("tcp", host, "/mysync/ctlrpc")
	} else {
		fmt.Println("rpc/tls mode")
		var cfg tls.Config
		roots := x509.NewCertPool()
		pem, err := ioutil.ReadFile(path.Join(pri_key_dir, "rootcas/root-cert.pem"))
		if err != nil {
			log.Fatalf("Read PEM error:%v\n", err)
		}
		roots.AppendCertsFromPEM(pem)
		cfg.RootCAs = roots
		conn1, err := tls.Dial("tcp", host, &cfg)
		if err != nil {
			log.Fatal(err)
		}
		defer conn1.Close()
		client = rpc.NewClient(conn1)
	}

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
		for i, v := range uplist {
			log.Printf("UPLOAD %v: %v\n", i+1, v)
		}
		if *mode1 == "http" {
			key1 = uploadList(uplist, name1, key1)
			if key1 == nil {
				panic("upload list fail")
			}
		} else {
			err = rpcUploadList(client, uplist, name1, key1)
			if err != nil {
				panic(err)
			}
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
	filename1 := zipList(uplist)

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
	var url = fmt.Sprintf("http://%v/mysync/upload", host)
	ret := files.PostFile(filename1, url, &files.MyValid{Sig: valid, Msg: msg})
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
		pri_key_dir = path.Join(home, "config/mysync/")
		conf_file_dir = path.Join(home, "config/mysync/")
		return
	}
	log.Println("OS: Windows")
	exe1, _ := os.Executable()
	dir1 := filepath.Dir(exe1)
	conf1 := filepath.Join(dir1, "config/mysync/")
	pri_key_dir = conf1
	conf_file_dir = conf1
}

func rpcUploadList(rpc1 *rpc.Client, uplist []string, name1 string, k []byte) error {
	upfile := zipList(uplist)
	//Rpc CreateTempFile
	//verify message
	var b1 = make([]byte, 32)
	io.ReadFull(rand.Reader, b1)
	buf1 := bytes.NewBuffer(b1)
	buf1.WriteString(name1)
	//crypto message with received key
	valid := mycrypto.AES256Encode(k, buf1.Bytes())
	if valid == nil {
		return errors.New("rpcUploadList: aes crypto error")
	}
	var arg1 = Args{valid, buf1.Bytes(), nil}
	var fid int
	err := rpc1.Call("Ctlrpc.CreateTempFile", &arg1, &fid)
	if err != nil {
		return err
	}
	//Rpc AppendFile
	var size1, sent int
	info1, _ := os.Stat(upfile)
	size1 = int(info1.Size())
	var arg2 = AppendData{fid, name1, nil}
	var reply int
	fp1, err := os.Open(upfile)
	if err != nil {
		return errors.New(fmt.Sprintf("open %s:%v", upfile, err))
	}
	defer fp1.Close()
	fp := bufio.NewReaderSize(fp1, 1024*1024*4)
	block1 := make([]byte, 1024*1024*2)
	buf2 := bytes.NewBufferString("")
	zw1 := gzip.NewWriter(buf2)
	//var idx int
	fmt.Printf("Size : %.1fM\n", float64(size1)/float64(1024*1024))
	for n, _ := fp.Read(block1); n > 0; n, _ = fp.Read(block1) {
		zw1.Reset(buf2)
		zw1.Write(block1[:n])
		zw1.Flush()
		arg2.Gz = buf2.Bytes()
		err = rpc1.Call("Ctlrpc.AppendFile", &arg2, &reply)
		if err != nil {
			return err
		}
		buf2.Reset()
		if reply != n {
			return errors.New("AppendFile: send data not correct")
		}
		//idx += 1
		sent += n
		fmt.Printf("\rSend: %d%%", (sent*100)/size1)
	}
	fmt.Printf("\n")
	arg2.Gz = nil
	err = rpc1.Call("Ctlrpc.FinishFile", &arg2, &reply)
	if err != nil {
		return err
	}
	return nil
}
func zipList(uplist []string) string {
	//create zip file
	dir1 := filepath.Join(root, "_backup")
	os.MkdirAll(dir1, os.ModePerm)

	now1 := time.Now()
	var filename1 string
	for i := 1; true; i++ {
		backupName := fmt.Sprintf("up%s-%d%s", now1.Format("20060102"), i, ".zip")
		_, err := os.Stat(filepath.Join(dir1, backupName))
		if err != nil {
			filename1 = filepath.Join(dir1, backupName)
			break
		}
	}
	fp, err := os.Create(filename1)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer fp.Close()
	zf := zip.NewWriter(fp)
	defer zf.Close()
	os.Chdir(root)
	for _, p := range uplist {
		p1 := p
		fp1, err := os.Open(p1)
		if err != nil {
			log.Println(err)
			return ""
		}
		st1, err := fp1.Stat()
		if err != nil {
			log.Println(err)
			return ""
		}
		fh, err := zip.FileInfoHeader(st1)
		fh.Name = p1
		//log.Println(fh.Name, st1.Name())
		if err != nil {
			log.Println(err)
			return ""
		}
		fh.Method = zip.Deflate
		fh.Modified = st1.ModTime()
		f, err := zf.CreateHeader(fh)
		if err != nil {
			log.Println(err)
			return ""
		}
		if st1.IsDir() {
			fp1.Close()
		} else {
			io.Copy(f, fp1)
			fp1.Close()
		}
	}
	return filename1
}
