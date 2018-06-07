package conf

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

func ReadJSON(filename string) map[string]string {
	var WConf map[string]string = make(map[string]string)
	var r1 = make(map[string]interface{})
	buf1, err1 := ioutil.ReadFile(filename)
	if err1 != nil {
		log.Println(err1)
		return nil
	}
	err1 = json.Unmarshal(buf1, &r1)
	if err1 != nil {
		log.Println(err1)
		return nil
	}
	var s1 string
	var ok bool
	for k, v := range r1 {
		s1, ok = v.(string)
		if ok {
			WConf[k] = s1
		}
	}
	return WConf
}
