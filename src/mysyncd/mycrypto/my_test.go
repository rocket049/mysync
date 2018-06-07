package mycrypto

import (
	"bytes"
	"testing"
)

func TestAES256Encode(t *testing.T) {
	k := "88888888999999993333333322222222"
	msg := "good idea!"
	ct1 := AES256Encode([]byte(k), []byte(msg))
	if ct1 == nil {
		t.Error("error encoding")
		return
	}
	msg1 := AES256Decode([]byte(k), ct1)
	if msg1 == nil {
		t.Error("error decoding")
		return
	}
	if bytes.Compare([]byte(msg), msg1) != 0 {
		t.Error("not equal", msg, string(msg1))
	}
}
