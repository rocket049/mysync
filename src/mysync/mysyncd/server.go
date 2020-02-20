package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mr "math/rand"
	"mysync/mysyncd/conf"
	"mysync/mysyncd/files"
	"mysync/mysyncd/mycrypto"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"time"
)

func init() {
	mr.Seed(time.Now().Unix())
}

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
var trans map[int]*os.File = make(map[int]*os.File)

const descName = "_desc.json"

func set_win_dir() {
	if len(os.Getenv("windir")) == 0 {
		//linux
		home := os.Getenv("HOME")
		pub_key_dir = path.Join(home, "config/mysyncd/")
		conf_file_dir = path.Join(home, "config/mysyncd/")
		return
	}
	//windows
	log.Println("OS: Windows")
	exe1, _ := os.Executable()
	dir1 := filepath.Dir(exe1)
	conf1 := filepath.Join(dir1, "config/mysyncd/")
	pub_key_dir = conf1
	conf_file_dir = conf1
}

type OperatorMutex struct {
	listlock, aeslock, rootlock, translock sync.RWMutex
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
func (self *OperatorMutex) SetTrans(key1 int, file1 *os.File) {
	self.translock.Lock()
	trans[key1] = file1
	self.translock.Unlock()
}
func (self *OperatorMutex) GetTrans(key1 int) *os.File {
	self.translock.RLock()
	file1, ok := trans[key1]
	self.translock.RUnlock()
	if ok {
		return file1
	} else {
		return nil
	}
}
func (self *OperatorMutex) DelCloseTrans(key1 int) {
	self.translock.Lock()
	file1, ok := trans[key1]
	if ok {
		file1.Close()
	}
	delete(trans, key1)
	self.translock.Unlock()
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
	if len(arg.Msg) < 33 {
		return errors.New("fail verify")
	}
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
	flist, err := GetFileMap(path1)
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
	if len(arg.Msg) < 33 {
		return errors.New("fail verify")
	}
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
	//temp save filemap
	cfg.SetList(name1, arg.FileMap)
	return nil
}
func (t *Ctlrpc) Logout(arg *Args, res *string) error {
	if len(arg.Msg) < 33 {
		return errors.New("fail verify")
	}
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

func (t *Ctlrpc) CreateTempFile(arg *Args, key1 *int) error {
	//compare and return upload list
	if len(arg.Msg) < 33 {
		return errors.New("fail verify")
	}
	name1 := string(arg.Msg[32:])
	k1 := cfg.GetKey(name1)
	vmsg := mycrypto.AES256Decode(k1, arg.Valid)
	if bytes.Compare(vmsg, arg.Msg) != 0 {
		return errors.New("SyncDel: security verify fail")
	}
	//create file
	*key1 = mr.Int()
	tmp := path.Join(os.Getenv("HOME"), ".tmp")
	os.MkdirAll(tmp, os.ModePerm)
	tmpzip, _ := ioutil.TempFile(tmp, "up")
	cfg.SetTrans(*key1, tmpzip)
	return nil
}

type AppendData struct {
	Key  int
	Name string
	Gz   []byte
}

func (t *Ctlrpc) AppendFile(arg *AppendData, size1 *int) error {
	buf1 := bytes.NewBuffer(arg.Gz)
	zr1, err := gzip.NewReader(buf1)
	if err != nil {
		return err
	}
	fp := cfg.GetTrans(arg.Key)
	if fp == nil {
		return errors.New("File not Created")
	}
	sz1, _ := io.Copy(fp, zr1)
	*size1 = int(sz1)
	zr1.Close()
	return nil
}
func (t *Ctlrpc) FinishFile(arg *AppendData, reply *int) error {
	fp := cfg.GetTrans(arg.Key)
	pathname1 := fp.Name()
	//close fp in DelCloseTrans
	cfg.DelCloseTrans(arg.Key)
	defer os.Remove(pathname1)
	//unzip
	*reply = 1
	err := UnZipFile(pathname1, arg.Name)
	if err != nil {
		return err
	}
	//save filemap
	flist := cfg.GetList(arg.Name)
	data, err := json.Marshal(flist)
	if err != nil {
		return err
	}
	fn := filepath.Join(cfg.GetRoot(arg.Name), descName)
	err = ioutil.WriteFile(fn, data, 0644)
	return err
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
	var host = flag.String("host", ":6080", "[-host ip:port]: bind special address and port")

	flag.Parse()
	set_win_dir()
	//set log not output
	//var null1 = new(NullWriter)
	//log.SetOutput(null1)
	//defer null1.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//set tls config
	cert, err := tls.LoadX509KeyPair(path.Join(pub_key_dir, "rootcas/cert.pem"),
		path.Join(pub_key_dir, "rootcas/key.pem"))
	if err != nil {
		log.Fatal(err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	ctl := new(Ctlrpc)

	err = rpc.Register(ctl)
	if err != nil {
		panic(err)
	}
	l, e := tls.Listen("tcp", *host, cfg)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	defer l.Close()
	go rpc.Accept(l)

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

func UnZipFile(filename, name1 string) error {
	root := cfg.GetRoot(name1)
	zreader, err := zip.OpenReader(filename)
	if err != nil {
		log.Println(err)
		return err
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
					return err1
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
				return err1
			}
			f1, err1 := os.Create(path1)
			if err1 != nil {
				log.Println(err1)
				return err1
			}
			rd1, _ := v.Open()
			_, err1 = io.Copy(f1, rd1)
			if err1 != nil {
				log.Println(err1)
				return err1
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
	return nil
}

func GetCachedFileMap(rootpath string) (res map[string]files.FileDesc, err error) {
	fn := filepath.Join(rootpath, descName)
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	res = make(map[string]files.FileDesc)
	err = json.Unmarshal(data, &res)
	return
}

func GetFileMap(rootpath string) (res map[string]files.FileDesc, err error) {
	res, err = GetCachedFileMap(rootpath)
	if err == nil {
		return
	}
	res, err = files.GetFileMap(rootpath)
	return
}
