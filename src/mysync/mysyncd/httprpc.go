package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mysync/mysyncd/conf"
	"mysync/mysyncd/files"
	"mysync/mysyncd/mycrypto"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
)

type Args struct {
	Valid   []byte
	Msg     []byte
	FileMap map[string]files.FileDesc
}
type Reply struct {
	Valid   []byte
	UpList  []string
	DelList []string
}

type Ctlrpc int

//global vars
var pub_key_dir = "mysyncd/"
var conf_file_dir = "mysyncd/"
var filemap map[string]map[string]files.FileDesc = make(map[string]map[string]files.FileDesc)
var aeskey map[string][]byte = make(map[string][]byte)
var root map[string]string = make(map[string]string)

func set_win_dir() {
	if len(os.Getenv("windir")) == 0 {
		home := os.Getenv("HOME")
		pub_key_dir = path.Join(home, "mysyncd/")
		conf_file_dir = path.Join(home, "mysyncd/")
		return
	}
	log.Println("OS: Windows")
	exe1, _ := os.Executable()
	dir1 := filepath.Dir(exe1)
	conf1 := filepath.Join(dir1, "conf/")
	pub_key_dir = conf1
	conf_file_dir = conf1
}

type OperatorMutex struct {
	listlock, aeslock, rootlock sync.RWMutex
}

var cfg = new(OperatorMutex)

func (self *OperatorMutex) SetList(name1 string, list1 map[string]files.FileDesc) {
	self.listlock.Lock()
	filemap[name1] = list1
	self.listlock.Unlock()
}
func (self *OperatorMutex) GetList(name1 string) map[string]files.FileDesc {
	self.listlock.RLock()
	list1, ok := filemap[name1]
	self.listlock.RUnlock()
	if ok {
		return list1
	} else {
		return nil
	}
}
func (self *OperatorMutex) SetKey(name1 string, key1 []byte) {
	self.aeslock.Lock()
	aeskey[name1] = key1
	self.aeslock.Unlock()
}
func (self *OperatorMutex) GetKey(name1 string) []byte {
	self.aeslock.RLock()
	key1, ok := aeskey[name1]
	self.aeslock.RUnlock()
	if ok {
		return key1
	} else {
		return nil
	}
}
func (self *OperatorMutex) SetRoot(name1 string, path1 string) {
	self.rootlock.Lock()
	root[name1] = path1
	self.rootlock.Unlock()
}
func (self *OperatorMutex) GetRoot(name1 string) string {
	self.rootlock.RLock()
	path1, ok := root[name1]
	self.rootlock.RUnlock()
	if ok {
		return path1
	} else {
		return ""
	}
}
func (self *OperatorMutex) Release(name1 string) {
	self.listlock.Lock()
	delete(filemap, name1)
	self.listlock.Unlock()

	self.aeslock.Lock()
	delete(aeskey, name1)
	self.aeslock.Unlock()

	self.rootlock.Lock()
	delete(root, name1)
	self.rootlock.Unlock()
}

func (t *Ctlrpc) Login(arg *Args, reply *[]byte) error {
	//rsa valid
	name1 := string(arg.Msg[32:])
	pub_keyfile := path.Join(pub_key_dir, fmt.Sprintf("%v.pub", name1))
	pubk := mycrypto.ReadPublicKey(pub_keyfile)
	if pubk == nil {
		return errors.New("fail read public key file")
	}
	if mycrypto.VerifyWithKey(pubk, arg.Msg, arg.Valid) == false {
		return errors.New("RSA Verify fail")
	}
	//new aes256 key
	k1 := make([]byte, 32)
	io.ReadFull(rand.Reader, k1)
	valid, _ := mycrypto.EncodeWithKey(pubk, k1)
	if valid == nil {
		return errors.New("fail RSA encode")
	}
	// rsa encoded ase256 key
	*reply = valid
	conf_file := path.Join(conf_file_dir, fmt.Sprintf("%v.json", name1))
	cfg1 := conf.ReadJSON(conf_file)
	if cfg1 == nil {
		return errors.New("Server Not Configed")
	}
	path1, ok := cfg1["root"]
	if ok == false {
		return errors.New("Path error on Server")
	}
	flist, err := files.GetFileMap(path1)
	if err != nil {
		return err
	}
	//filemap[name1] = flist
	cfg.SetList(name1, flist)
	//root[name1] = path1
	cfg.SetRoot(name1, path1)
	//asekey[name1] = k1
	cfg.SetKey(name1, k1)
	return nil
}

func (t *Ctlrpc) SyncDel(arg *Args, res *Reply) error {
	//compare and return upload list
	name1 := string(arg.Msg[32:])
	k1 := cfg.GetKey(name1)
	vmsg := mycrypto.AES256Decode(k1, arg.Valid)
	if bytes.Compare(vmsg, arg.Msg) != 0 {
		return errors.New("SyncDel: security verify fail")
	}
	flist := cfg.GetList(name1)
	up1, del1 := files.DiffList(arg.FileMap, flist)
	path1 := cfg.GetRoot(name1)
	for _, v := range del1 {
		p1 := path.Join(path1, v)
		os.RemoveAll(p1)
		log.Printf("DEL: %v\n", p1)
	}
	//reply uplosd list
	res.UpList = up1
	res.DelList = del1
	//return new key crypto with old key
	var k = make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	ck := mycrypto.AES256Encode(k1, k)
	res.Valid = ck
	//asekey[name1] = k
	cfg.SetKey(name1, k)
	return nil
}
func (t *Ctlrpc) Logout(arg *Args, res *string) error {
	name1 := string(arg.Msg[32:])
	vmsg := mycrypto.AES256Decode(cfg.GetKey(name1), arg.Valid)
	if bytes.Compare(vmsg, arg.Msg) != 0 {
		return errors.New("Logout: security verify fail")
	}
	*res = "Logout"
	//delete(filemap, name1)
	//delete(asekey, name1)
	//delete(root, name1)
	cfg.Release(name1)
	return nil
}

type NullWriter struct {
	fp *os.File
}

func (self *NullWriter) Write(b []byte) (int, error) {
	if self.fp == nil {
		self.fp, _ = os.OpenFile(path.Join(conf_file_dir, "mysyncd.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	self.fp.Write(b)
	return len(b), nil
}
func (self *NullWriter) Close() {
	self.fp.Close()
}

func main() {
	set_win_dir()
	//set log not output
	var null1 = new(NullWriter)
	log.SetOutput(null1)
	defer null1.Close()

	ctl := new(Ctlrpc)
	rpc1 := rpc.NewServer()
	rpc1.Register(ctl)
	http.HandleFunc("/mysync/upload", HandleUpload)
	rpc1.HandleHTTP("/mysync/ctlrpc", "/mysync/dbgrpc")
	l, e := net.Listen("tcp", ":6080")
	defer l.Close()
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	wait_sig()
}

func wait_sig() {
	var c chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("\nSignal:", s)
}
func mkdir_p(p string) error {
	d, _ := path.Split(p)
	if len(d) == 0 {
		return nil
	}
	return os.MkdirAll(d, os.ModePerm)
}

//recive zip file contain all upload list
func HandleUpload(resp http.ResponseWriter, req *http.Request) {
	r, err := req.MultipartReader()
	if err != nil {
		log.Println(err)
		resp.WriteHeader(200)
		resp.Write([]byte("fail get multiform"))
		return
	}
	form1, err := r.ReadForm(5 * 1024 * 1024)
	if err != nil {
		log.Println("ReadForm:", err)
		resp.WriteHeader(200)
		resp.Write([]byte("fail read form"))
		return
	}
	sig, _ := hex.DecodeString(form1.Value["sig"][0])
	msg, _ := hex.DecodeString(form1.Value["msg"][0])
	//log.Println(msg, req.FormValue("msg"))
	name1 := string(msg[32:])
	k1 := cfg.GetKey(name1)
	vmsg := mycrypto.AES256Decode(k1, sig)
	if bytes.Compare(vmsg, msg) != 0 {
		resp.WriteHeader(200)
		resp.Write([]byte("fail verify"))
		return
	}
	fh1, ok := form1.File["upfile"]
	if !ok {
		log.Println(err)
		resp.WriteHeader(200)
		resp.Write([]byte("fail upload"))
		return
	}
	file1, _ := fh1[0].Open()
	root := cfg.GetRoot(name1)
	tmp := path.Join(os.Getenv("HOME"), ".tmp")
	os.MkdirAll(tmp, os.ModePerm)
	tmpzip, _ := ioutil.TempFile(tmp, "up")
	defer os.Remove(tmpzip.Name())
	defer tmpzip.Close()
	size1, _ := io.Copy(tmpzip, file1)
	file1.Close()
	tmpzip.Seek(0, 0)
	zreader, err := zip.NewReader(tmpzip, size1)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(200)
		resp.Write([]byte("fail unzip"))
		return
	}
	for _, v := range zreader.File {
		info := v.FileInfo()
		path1 := path.Join(root, v.Name)
		log.Printf("UPLOAD: %v\n", path1)
		if info.IsDir() {
			_, err1 := os.Stat(path1)
			if err1 != nil {
				err1 = os.MkdirAll(path1, os.ModePerm)
				if err1 != nil {
					log.Println(err1)
					resp.WriteHeader(200)
					resp.Write([]byte("fail mkdir"))
					return
				}
			}
		}
	}
	for _, v := range zreader.File {
		info := v.FileInfo()
		path1 := path.Join(root, v.Name)
		if !info.IsDir() {
			//修复缺失的目录
			err1 := mkdir_p(path1)
			if err1 != nil {
				log.Println(err1)
				resp.WriteHeader(200)
				resp.Write([]byte("fail mkdir"))
				return
			}
			f1, err1 := os.Create(path1)
			if err1 != nil {
				log.Println(err1)
				resp.WriteHeader(200)
				resp.Write([]byte("fail create"))
				return
			}
			rd1, _ := v.Open()
			_, err1 = io.Copy(f1, rd1)
			if err1 != nil {
				log.Println(err1)
				resp.WriteHeader(200)
				resp.Write([]byte("fail unzip file"))
				return
			}
			rd1.Close()
			f1.Close()
			os.Chtimes(path1, v.Modified, v.Modified)
		}
	}
	for _, v := range zreader.File {
		info := v.FileInfo()
		path1 := path.Join(root, v.Name)
		if info.IsDir() {
			os.Chtimes(path1, v.Modified, v.Modified)
		}
	}
	//return new key crypto with old key
	var k = make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	ck := hex.EncodeToString(mycrypto.AES256Encode(k1, k))
	cfg.SetKey(name1, k)
	resp.WriteHeader(200)
	resp.Write([]byte("ok " + ck))
}
