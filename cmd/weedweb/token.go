package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"io"
)

var hashKey []byte
var hashFunc = sha256.New
var block cipher.Block

// GenerateRandomKey creates a random key with the given strength.
func GenerateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func init() {
	hashKey = GenerateRandomKey(64)
	var err error
	block, err = aes.NewCipher(GenerateRandomKey(32))
	if err != nil {
		panic(err)
	}
}

func generateToken(v interface{}) ([]byte, error) {
	var b []byte
	var err error
	if b, err = json.Marshal(v); err != nil {
		return nil, err
	}
	if b, err = encrypt(block, b); err != nil {
		return nil, err
	}
	b = encode(b)
	mac := createMac(hmac.New(hashFunc, hashKey), b)
	b = append(b, '|')
	b = append(b, mac...)
	b = encode(b)
	return b, nil
}

func validateToken(s []byte, v interface{}) error {
	b, err := decode(s)
	if err != nil {
		return err
	}
	parts := bytes.SplitN(b, []byte("|"), 2)
	if len(parts) != 2 {
		return errors.New("token: invalid value %v")
	}
	h := hmac.New(hashFunc, hashKey)
	if err = verifyMac(h, parts[0], parts[1]); err != nil {
		return err
	}
	b, err = decode(parts[0])
	if err != nil {
		return err
	}
	if b, err = decrypt(block, b); err != nil {
		return err
	}

	if err = json.Unmarshal(b, v); err != nil {
		return err
	}

	return nil
}

// Encryption -----------------------------------------------------------------

// encrypt encrypts a value using the given block in counter mode.
//
// A random initialization vector (http://goo.gl/zF67k) with the length of the
// block size is prepended to the resulting ciphertext.
func encrypt(block cipher.Block, value []byte) ([]byte, error) {
	iv := GenerateRandomKey(block.BlockSize())
	if iv == nil {
		return nil, errors.New("securecookie: failed to generate random iv")
	}
	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	// Return iv + ciphertext.
	return append(iv, value...), nil
}

// decrypt decrypts a value using the given block in counter mode.
//
// The value to be decrypted must be prepended by a initialization vector
// (http://goo.gl/zF67k) with the length of the block size.
func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}
	return nil, errors.New("securecookie: the value could not be decrypted")
}

// Encoding -------------------------------------------------------------------

// encode encodes a value using base64.
func encode(value []byte) []byte {
	encoded := make([]byte,
		base64.URLEncoding.EncodedLen(len(value)))
	base64.URLEncoding.Encode(encoded, value)
	return encoded
}

// decode decodes a cookie using base64.
func decode(value []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(value)))
	b, err := base64.URLEncoding.Decode(decoded, value)
	if err != nil {
		return nil, err
	}
	return decoded[:b], nil
}

// Authentication -------------------------------------------------------------

// createMac creates a message authentication code (MAC).
func createMac(h hash.Hash, value []byte) []byte {
	h.Write(value)
	return h.Sum(nil)
}

// verifyMac verifies that a message authentication code (MAC)
// is valid.
func verifyMac(h hash.Hash, value []byte, mac []byte) error {
	mac2 := createMac(h, value)
	if len(mac) == len(mac2) &&
		subtle.ConstantTimeCompare(mac, mac2) ==
			1 {
		return nil
	}
	return errors.New("Mac invalid")
}
