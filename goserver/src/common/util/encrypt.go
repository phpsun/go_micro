package util

import (
	"encoding/base64"
	"encoding/hex"
)

var AES_IV = []byte{0x2b, 0x7e, 0x15, 0x16, 0x28, 0xae, 0xd2, 0xa6, 0xab, 0xf7, 0x15, 0x88, 0x09, 0xcf, 0x4f, 0x3c}

func Encrypt(data []byte, key string) (string, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	v, err := AESEncrypt(data, k, AES_IV)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(v), nil
}

func Decrypt(data string, key string) ([]byte, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	val, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return AESDecrypt(val, k, AES_IV)
}

/*
func EncryptTest() {
	// Content-Type: text/plain; charset="ISO-8859-1"
	aesKey := pbkdf2.Key([]byte("18490635119204798369755864430"), []byte("oconnection0917"), 16, 16, sha1.New)
	key := hex.EncodeToString(aesKey)

	b, _ := Decrypt("Z2gtkl8URjr+CkO5du82jA==", key)
	fmt.Println(string(b)) // "123"

	s, _ := Encrypt([]byte("456"), key)
	fmt.Println(s) // "B2AeC94me+oG/76i6YOs0g=="
}
*/
