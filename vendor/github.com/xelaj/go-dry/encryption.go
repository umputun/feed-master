// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"sync"

	"golang.org/x/crypto/sha3"
)

var (
	AES = aesCipherPool{NewSyncPoolMap()}
)

// The pool uses sync.Pool internally.
type aesCipherPool struct {
	poolMap *SyncPoolMap
}

func (pool *aesCipherPool) forKey(key []byte) *sync.Pool {
	return pool.poolMap.GetOrAddNew(string(key), func() any {
		block, err := aes.NewCipher(key)
		if err != nil {
			panic(err)
		}
		return block
	})
}

func (pool *aesCipherPool) GetCypher(key []byte) cipher.Block {
	return pool.forKey(key).Get().(cipher.Block)
}

func (pool *aesCipherPool) ReturnCypher(key []byte, block cipher.Block) {
	pool.forKey(key).Put(block)
}

// EncryptAES encrypts plaintext using AES with the given key.
// key should be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256.
// plaintext must not be shorter than key.
func EncryptAES(key []byte, plaintext []byte) []byte {
	block := AES.GetCypher(key)
	defer AES.ReturnCypher(key, block)

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext
}

// DecryptAES decrypts ciphertext using AES with the given key.
// key should be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256.
func DecryptAES(key []byte, ciphertext []byte) []byte {
	block := AES.GetCypher(key)
	defer AES.ReturnCypher(key, block)

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext
}

func Sha3256(text string) []byte {
	h := sha3.New256()
	h.Write([]byte(text))
	return h.Sum(nil)
}

func Sha3512(text string) []byte {
	h := sha3.New512()
	h.Write([]byte(text))
	return h.Sum(nil)
}

func Sha3512Hex(text string) string {
	h := sha3.New512()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func Sha1(input string) []byte {
	r := sha1.Sum([]byte(input))
	return r[:]
}

func Sha1Byte(input []byte) []byte {
	r := sha1.Sum(input)
	return r[:]
}
