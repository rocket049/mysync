package files

import (
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type MyValid struct {
	Sig []byte
	Msg []byte
}

//upload and return new key cryptoed
func PostFile(filename, url string, valid *MyValid) []byte {
	buf1 := bytes.NewBufferString("")
	multi1 := multipart.NewWriter(buf1)
	//multi1.CreateFormField("sig")
	multi1.WriteField("sig", hex.EncodeToString(valid.Sig))
	//multi1.CreateFormField("msg")
	multi1.WriteField("msg", hex.EncodeToString(valid.Msg))
	part, err := multi1.CreateFormFile("upfile", filepath.Base(filename))
	if err != nil {
		log.Println(err)
		return nil
	}
	//boundary1 := multi1.Boundary()
	//close_boundary := fmt.Sprintf("\n\r--%v--\b\r", boundary1)
	//close_buf := bytes.NewBufferString(close_boundary)
	//file
	fp, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer fp.Close()
	io.Copy(part, fp)
	multi1.Close()
	//info1, _ := fp.Stat()
	//fsize := info1.Size()
	//multireader
	//mr1 := io.MultiReader(buf1, fp, close_buf)
	//body_size := fsize + int64(buf1.Len()) + int64(close_buf.Len())
	req, err := http.NewRequest("POST", url, buf1)
	if err != nil {
		log.Println(err)
		return nil
	}
	req.Header.Add("Content-Type", multi1.FormDataContentType())
	//req.ContentLength = body_size
	res, err := http.DefaultClient.Do(req)
	//end upload
	//get response
	buf2 := bytes.NewBufferString("")
	buf2.ReadFrom(res.Body)
	res.Body.Close()
	retb := buf2.Bytes()
	if bytes.Compare(retb[:4], []byte("fail")) == 0 {
		log.Println(string(retb))
		return nil
	} else {
		return retb[3:]
	}
}
