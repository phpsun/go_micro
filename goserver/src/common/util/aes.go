package util

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// pad uses the PKCS#7 padding scheme to align the a payload to a specific block size
func pad(plaintext []byte, bsize int) ([]byte, error) {
	if bsize >= 256 {
		return nil, errors.New("bsize must be < 256")
	}
	pad := bsize - (len(plaintext) % bsize)
	if pad == 0 {
		pad = bsize
	}
	for i := 0; i < pad; i++ {
		plaintext = append(plaintext, byte(pad))
	}
	return plaintext, nil
}

// unpad strips the padding previously added using the PKCS#7 padding scheme
func unpad(paddedtext []byte) ([]byte, error) {
	length := len(paddedtext)
	paddedtext, lbyte := paddedtext[:length-1], paddedtext[length-1]
	pad := int(lbyte)
	if pad >= 256 || pad > length {
		return nil, errors.New("padding malformed")
	}
	return paddedtext[:length-(pad)], nil
}

// AESEncrypt encrypts a payload with an AES cipher.
// The returned ciphertext has three notable properties:
// 1. ciphertext is aligned to the standard AES block size
// 2. ciphertext is padded using PKCS#7
func AESEncrypt(plaintext, key []byte, iv []byte) ([]byte, error) {
	if len(key) != len(iv) {
		return nil, errors.New("iv mismatch")
	}

	blockSize := aes.BlockSize
	plaintext, err := pad(plaintext, blockSize)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)
	return ciphertext, nil
}

// AESDecrypt decrypts an encrypted payload with an AES cipher.
// The decryption algorithm makes three assumptions:
// 1. ciphertext is aligned to the standard AES block size
// 2. ciphertext is padded using PKCS#7
func AESDecrypt(ciphertext, key []byte, iv []byte) ([]byte, error) {
	if len(key) != len(iv) {
		return nil, errors.New("iv mismatch")
	}

	blockSize := aes.BlockSize
	if len(ciphertext) < blockSize {
		return nil, errors.New("ciphertext too short")
	}

	sz := len(ciphertext)
	if sz%blockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, sz)
	//for s := 0; s < sz; {
	//	e := s + blockSize
	//	block.Decrypt(plaintext[s:e], ciphertext[s:e])
	//	s = e
	//}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)
	return unpad(plaintext)
}
