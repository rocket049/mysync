package files

import (
	"bytes"
	"crypto/md5"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileDesc struct {
	Pathname string
	Mtime    time.Time
	Size     int
	MD5      []byte
}

var file_map map[string]FileDesc
var start_pos int

func GetFileMap(rootpath string) (map[string]FileDesc, error) {
	file_map = make(map[string]FileDesc)
	start_pos = len(rootpath)
	err := filepath.Walk(rootpath, walker)
	if err != nil {
		log.Printf("prevent panic by handling failure accessing a path %q: %v\n", rootpath, err)
		return nil, err
	} else {
		return file_map, nil
	}
}

func GetFileMD5(pathname string) []byte {
	f, err := os.Open(pathname)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Println(err)
		return nil
	}
	res1 := h.Sum(nil)
	return res1[:]
}
func walker(path1 string, info os.FileInfo, err error) error {
	var fd1 FileDesc
	upath := GetUnixPath(path1[start_pos:])
	//name1 := filepath.Base(upath)
	if len(upath) == 0 {
		return nil
	}
	if upath[:1] == "." {
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
		fd1.MD5 = GetFileMD5(path1)
	}
	file_map[upath] = fd1
	return nil
}

//DiffList: return upload_list,del_list
func DiffList(src_list, dist_list map[string]FileDesc) (up, del []string) {
	var up_list, del_list []string
	//get upload list
	for p1, info1 := range src_list {
		info2, ok := dist_list[p1]
		if ok {
			if info2.Size != info1.Size || bytes.Compare(info2.MD5, info1.MD5) != 0 {
				up_list = append(up_list, p1)
			}
		} else {
			up_list = append(up_list, p1)
		}
	}
	//get del_list
	for p1, _ := range dist_list {
		_, ok := src_list[p1]
		if !ok {
			del_list = append(del_list, p1)
		}
	}
	return up_list, del_list
}

func GetUnixPath(path1 string) string {
	var res = path1[:]
	p1 := strings.Index(path1, ":")
	if p1 != -1 {
		res = path1[p1+1:]
	}
	res = strings.Replace(res, "\\", "/", -1)
	return strings.Trim(res, "/")
}

//try
//func main(){
//list1,_ := GetFileMap( os.Args[1] )
//list2,_ := GetFileMap( os.Args[2] )
//log.Println("Root",os.Args[1])
//for k,_ := range list1{
//log.Println(k)
//}
//log.Println("Root",os.Args[2])
//for k,_ := range list2{
//log.Println(k)
//}
//up,del := DiffList(list1,list2)
//log.Println("Upload list:")
//for _,v := range up{
//log.Println(v)
//}
//log.Println("Del list:")
//for _,v := range del{
//log.Println(v)
//}
//}
