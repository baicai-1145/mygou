package kugou

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// The concept edition's public key, from KuGouMusicApi's crypto.js.
const publicKeyLite = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDECi0Np2UR87scwrvTr72L6oO01rBbbBPriSDFPxr3Z5syug0O24QyQO8bg27+0+4kBzTBTBOZ/WWU0WryL1JSXRTXLgFVxtzIY41Pe7lPOgsfTCn5kZcvKhYKJesKnnJDNr5/abvTGf+rHG3YRwsCHcQ08/q6ifSioBszvb3QiwIDAQAB
-----END PUBLIC KEY-----`

const randomAlphabet = "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func md5Bytes(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

// randomString is KuGouMusicApi's randomString: uppercase letters and
// digits, the same alphabet the device pretend uses.
func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = randomAlphabet[randIndex(len(randomAlphabet))]
	}
	return string(b)
}

func randIndex(n int) int {
	var b [1]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			return 0
		}
		if int(b[0]) < 256-(256%n) {
			return int(b[0]) % n
		}
	}
}

// randomUUID is a v4 UUID, as KuGouMusicApi's getGuid makes.
func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func publicKey() *rsa.PublicKey {
	block, _ := pem.Decode([]byte(publicKeyLite))
	if block == nil {
		panic("kugou: bad public key")
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	return k.(*rsa.PublicKey)
}

// rsaPKCS1v15Hex is KuGouMusicApi's rsaEncrypt2.
func rsaPKCS1v15Hex(plain []byte) (string, error) {
	enc, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey(), plain)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(enc), nil
}

// rsaRawHex is KuGouMusicApi's cryptoRSAEncrypt: the message is left
// padded with zeros to the key length and raised to e, without padding.
func rsaRawHex(plain []byte) (string, error) {
	key := publicKey()
	size := key.Size()
	if len(plain) > size {
		return "", errors.New("kugou: data longer than the RSA key")
	}
	padded := make([]byte, size)
	copy(padded[size-len(plain):], plain)
	m := new(big.Int).SetBytes(padded)
	e := new(big.Int).SetInt64(int64(key.E))
	c := new(big.Int).Exp(m, e, key.N)
	out := c.Text(16)
	if len(out) < size*2 {
		out = strings.Repeat("0", size*2-len(out)) + out
	}
	return out, nil
}

// aesEncryptCBC encrypts with AES-CBC and PKCS#7 padding.
func aesEncryptCBC(key, iv, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	buf := make([]byte, len(plain)+pad)
	copy(buf, plain)
	for i := len(plain); i < len(buf); i++ {
		buf[i] = byte(pad)
	}
	out := make([]byte, len(buf))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, buf)
	return out, nil
}

func aesDecryptCBC(key, iv, data []byte) ([]byte, error) {
	if len(data)%aes.BlockSize != 0 || len(data) == 0 {
		return nil, errors.New("kugou: bad ciphertext length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	pad := int(out[len(out)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(out) {
		return nil, errors.New("kugou: bad padding")
	}
	for _, b := range out[len(out)-pad:] {
		if int(b) != pad {
			return nil, errors.New("kugou: bad padding")
		}
	}
	return out[:len(out)-pad], nil
}

// playlistAESEncrypt is KuGouMusicApi's playlistAesEncrypt: a random six
// character key, whose MD5 halves are the AES key and IV.
func playlistAESEncrypt(plain string) (key, ciphertext string, err error) {
	key = strings.ToLower(randomString(6))
	sum := md5Hex(key)
	enc, err := aesEncryptCBC([]byte(sum[:16]), []byte(sum[16:32]), []byte(plain))
	if err != nil {
		return "", "", err
	}
	return key, base64.StdEncoding.EncodeToString(enc), nil
}

// playlistAESDecrypt decrypts data encrypted with the key playlistAESEncrypt
// returned.
func playlistAESDecrypt(key string, data []byte) ([]byte, error) {
	sum := md5Hex(key)
	return aesDecryptCBC([]byte(sum[:16]), []byte(sum[16:32]), data)
}

// krcKey is the XOR key KRC lyrics are obfuscated with.
var krcKey = [16]byte{64, 71, 97, 119, 94, 50, 116, 71, 81, 54, 49, 45, 206, 210, 110, 105}

// DecodeKRC turns the content /lyric returns into text: four bytes of
// header, XOR, then zlib.
func DecodeKRC(data []byte) (string, error) {
	if len(data) <= 4 {
		return "", errors.New("kugou: krc too short")
	}
	b := append([]byte(nil), data[4:]...)
	for i := range b {
		b[i] ^= krcKey[i%len(krcKey)]
	}
	zr, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		fr := flate.NewReader(bytes.NewReader(b))
		out, err2 := io.ReadAll(fr)
		if err2 != nil {
			return "", err
		}
		return string(out), nil
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
