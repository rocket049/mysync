package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
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
	Mode    int8 //0:upload and delete, 1:upload, not delete
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

var host string

const root = "."

const pri_key_dir = "_mysync/"
const conf_file_dir = "_mysync/"

var mode *int

func main() {
	mode = flag.Int("m", 1, "mode: 0-upload and delete, 1-upload not delete")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var cfg1 = conf.ReadJSON(path.Join(conf_file_dir, "config.json"))
	if cfg1 == nil {
		panic("No config.json ")
	}

	host = cfg1["host"]
	var name1 = cfg1["key"]
	var client *rpc.Client
	var err error

	var cfg tls.Config
	roots := x509.NewCertPool()
	pem, err := ioutil.ReadFile(path.Join(pri_key_dir, "cert.pem"))
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

		err = rpcUploadList(client, uplist, name1, key1)
		if err != nil {
			panic(err)
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
	local_list, err := files.UpdateFileMap(root)
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
	//当 Mode==1, 服务端不删除多余的文件
	var arg = Args{Valid: valid, Msg: buf1.Bytes(), FileMap: local_list, Mode: int8(*mode)}
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

func logout(rpc1 *rpc.Client, name1 string, k []byte) error {
	var b1 = make([]byte, 32)
	io.ReadFull(rand.Reader, b1)
	buf1 := bytes.NewBuffer(b1)
	buf1.WriteString(name1)
	valid := mycrypto.AES256Encode(k, buf1.Bytes())
	if valid == nil {
		return errors.New("logout error AES256Encode")
	}
	var arg = Args{valid, buf1.Bytes(), nil, 0}
	var reply string = "logout: No reply"
	err := rpc1.Call("Ctlrpc.Logout", &arg, &reply)
	log.Println(reply)
	return err
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
	var arg1 = Args{valid, buf1.Bytes(), nil, 0}
	var fid int
	err := rpc1.Call("Ctlrpc.CreateTempFile", &arg1, &fid)
	if err != nil {
		return err
	}
	//Rpc AppendFile
	var size1, sent int
	info1, err := os.Stat(upfile)
	if err != nil {
		return err
	}
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
	if size1 < 1024*1024 {
		fmt.Printf("Size : %.1fK\n", float64(size1)/float64(1024))
	} else {
		fmt.Printf("Size : %.1fM\n", float64(size1)/float64(1024*1024))
	}

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
	if len(uplist) == 0 {
		return ""
	}
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
