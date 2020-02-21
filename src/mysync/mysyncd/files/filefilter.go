package files

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type FilterWalker struct {
	ResMap   map[string]FileDesc
	OldMap   map[string]FileDesc
	StartPos int
}

func NewFilterWalker(path1 string) (res *FilterWalker) {
	res = new(FilterWalker)
	res.ResMap = make(map[string]FileDesc)
	var err error
	res.OldMap, err = GetCachedFileMap(path1)
	if err != nil {
		res.OldMap = make(map[string]FileDesc)
	}
	if path1 == "." {
		res.StartPos = 0
	} else {
		res.StartPos = len(path1)
	}
	return
}

func (p *FilterWalker) Walker(path1 string, info os.FileInfo, err error) error {
	var fd1 FileDesc
	upath := GetUnixPath(path1[p.StartPos:])
	fd0, ok := p.OldMap[upath]
	if len(upath) == 0 {
		return nil
	}
	if upath[:1] == "." || upath[:1] == "_" {
		return nil
	}
	if info.IsDir() {
		fd1.Pathname = upath
		fd1.Mtime = time.Date(1927, time.November, 10, 23, 0, 0, 0, time.UTC)
		fd1.Size = -1
		fd1.MD5 = nil
	} else {
		fd1.Pathname = upath
		fd1.Mtime = info.ModTime()
		fd1.Size = int(info.Size())
		fd1.MD5 = nil
	}
	if ok == false {
		fd1.MD5 = GetFileMD5(path1)
		p.ResMap[upath] = fd1
		log.Println("add", upath)
		return nil
	}
	if fd1.Mtime.After(fd0.Mtime) {
		fd1.MD5 = GetFileMD5(path1)
		log.Println("update", upath)
	} else {
		fd1.MD5 = fd0.MD5
	}
	p.ResMap[upath] = fd1
	return nil
}

func GetCachedFileMap(rootpath string) (res map[string]FileDesc, err error) {
	fn := filepath.Join(rootpath, "_desc.json")
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	res = make(map[string]FileDesc)
	err = json.Unmarshal(data, &res)
	return
}

func UpdateFileMap(rootpath string) (res map[string]FileDesc, err error) {
	w := NewFilterWalker(rootpath)
	err = filepath.Walk(rootpath, w.Walker)
	if err != nil {
		log.Printf("prevent panic by handling failure accessing a path %q: %v\n", rootpath, err)
		return nil, err
	} else {
		return w.ResMap, nil
	}
}
