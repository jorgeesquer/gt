package lib

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/gtlang/gt/core"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

func init() {
	core.RegisterLib(Crypt, `

declare namespace crypto {
    export function signSHA1_RSA_PCKS1(privateKey: string, value: string): byte[]
    // export function verifySHA1_RSA(publicKey: string, message: string, signature: byte[]): void

    export function signTempSHA1(value: string): string
    export function checkTempSignSHA1(value: string, hash: string): boolean

    export function signSHA1(value: string): string
    export function checkSignSHA1(value: string, hash: string): boolean

    export function setGlobalPassword(pwd: string): void
    export function encrypt(value: byte[], pwd?: byte[]): byte[]
    export function decrypt(value: byte[], pwd?: byte[]): byte[]
    export function encryptTripleDES(value: byte[] | string, pwd?: byte[] | string): byte[]
    export function decryptTripleDES(value: byte[] | string, pwd?: byte[] | string): byte[]
    export function encryptString(value: string, pwd?: string): string
    export function decryptString(value: string, pwd?: string): string
    export function hashSHA(value: string): string
    export function hashSHA256(value: string): string
    export function hashSHA512(value: string): string
    export function hmacSHA256(value: byte[] | string, pwd?: byte[] | string): byte[]
    export function hashPassword(pwd: string): string
    export function compareHashAndPassword(hash: string, pwd: string): boolean
    export function random(len: number): byte[]
    export function randomAlphanumeric(len: number): string
}


`)
}

var globalPassword string
var tempSignKey = RandomAlphanumeric(30)

var Crypt = []core.NativeFunction{
	core.NativeFunction{
		Name:      "crypto.signSHA1_RSA_PCKS1",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			key := args[0].ToBytes()
			text := args[1].ToString()

			h := sha1.New()
			h.Write([]byte(text))
			sum := h.Sum(nil)

			block, _ := pem.Decode(key)
			if block == nil {
				return core.NullValue, fmt.Errorf("error parsing private key")
			}

			privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return core.NullValue, fmt.Errorf("error parsing private key: %v", err)
			}

			sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, sum)
			if err != nil {
				return core.NullValue, fmt.Errorf("error signing: %v", err)
			}

			return core.NewBytes(sig), nil
		},
	},
	// core.NativeFunc{
	// 	Name:      "crypto.verifySHA1_RSA",
	// 	Arguments: 3,
	// 	Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	// 		if err := ValidateArgs(args, core.StringType, core.StringType, core.BytesType); err != nil {
	// 			return core.NullValue, err
	// 		}

	// 		key := args[0].ToBytes()
	// 		message := args[1].ToBytes()
	// 		sign := args[2].ToBytes()

	// 		h := sha256.New()
	// 		h.Write(message)
	// 		d := h.Sum(nil)

	// 		block, _ := pem.Decode(key)
	// 		if block == nil {
	// 			return core.NullValue, fmt.Errorf("error parsing private key")
	// 		}

	// 		pubKey10I, err := x509.ParsePKIXPublicKey(block.Bytes)
	// 		if err != nil {
	// 			return core.NullValue, fmt.Errorf("error parsing private key: %v", err)
	// 		}

	// 		pubKey := pubKey10I.(*rsa.PublicKey)

	// 		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA1, d, sign); err != nil {
	// 			return core.NullValue, err
	// 		}

	// 		return core.NullValue, nil
	// 	},
	// },
	core.NativeFunction{
		Name:      "crypto.signSHA1",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			s := args[0].ToString() + "runtime" + globalPassword + "systems"
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return core.NewString(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.checkSignSHA1",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			s := args[0].ToString() + "runtime" + globalPassword + "systems"
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			ok := hash == args[1].ToString()

			//--------------------------------------------------
			// TEMP: remove in 11/2/2020
			if !ok {
				s = args[0].ToString() + "sclsoftware/scl" + globalPassword + "systems"
				h = sha1.New()
				h.Write([]byte(s))
				hash = hex.EncodeToString(h.Sum(nil))
				ok = hash == args[1].ToString()
			}
			if !ok {
				s = args[0].ToString() + "github.com/gtlang/gt/core" + globalPassword + "systems"
				h = sha1.New()
				h.Write([]byte(s))
				hash = hex.EncodeToString(h.Sum(nil))
				ok = hash == args[1].ToString()
			}
			//--------------------------------------------------

			return core.NewBool(ok), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.signTempSHA1",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			// untrusted users can check but not sign

			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			s := args[0].ToString() + tempSignKey
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return core.NewString(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.checkTempSignSHA1",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			// untrusted users can check but not sign

			s := args[0].ToString() + tempSignKey
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))

			ok := hash == args[1].ToString()
			return core.NewBool(ok), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.setGlobalPassword",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			globalPassword = args[0].ToString()
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.hmacSHA256",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Bytes); err != nil {
				return core.NullValue, err
			}

			msg := args[0].ToBytes()
			key := args[1].ToBytes()

			sig := hmac.New(sha256.New, key)
			sig.Write(msg)
			hash := sig.Sum(nil)

			return core.NewBytes(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.hashSHA",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			h := sha1.New()
			h.Write([]byte(args[0].ToString()))
			hash := hex.EncodeToString(h.Sum(nil))
			return core.NewString(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.hashSHA256",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			h := sha256.New()
			h.Write([]byte(args[0].ToString()))
			hash := hex.EncodeToString(h.Sum(nil))
			return core.NewString(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.hashSHA512",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			h := sha512.New()
			h.Write([]byte(args[0].ToString()))
			hash := hex.EncodeToString(h.Sum(nil))
			return core.NewString(hash), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.encryptString",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			var pwd string
			switch len(args) {
			case 0:
				return core.NullValue, fmt.Errorf("expected 1 argument, got 0")
			case 1:
				pwd = GetContext(vm).getProtectedItem("password").ToString()
				if pwd == "" {
					pwd = globalPassword
					if pwd == "" {
						return core.NullValue, fmt.Errorf("no password configured")
					}
				}
			case 2:
				pwd = args[1].ToString()
			}

			s, err := Encrypts(args[0].ToString(), pwd)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.decryptString",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			var pwd string
			switch len(args) {
			case 0:
				return core.NullValue, fmt.Errorf("expected 1 argument, got 0")
			case 1:
				pwd = GetContext(vm).getProtectedItem("password").ToString()
				if pwd == "" {
					pwd = globalPassword
					if pwd == "" {
						return core.NullValue, fmt.Errorf("no password configured")
					}
				}
			case 2:
				pwd = args[1].ToString()
			}

			s, err := Decrypts(args[0].ToString(), pwd)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.encrypt",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			var pwd string
			switch len(args) {
			case 0:
				return core.NullValue, fmt.Errorf("expected 1 argument, got 0")
			case 1:
				pwd = GetContext(vm).getProtectedItem("password").ToString()
				if pwd == "" {
					pwd = globalPassword
					if pwd == "" {
						return core.NullValue, fmt.Errorf("no password configured")
					}
				}

			case 2:
				pwd = args[1].ToString()
			}

			b, err := Encrypt(args[0].ToBytes(), []byte(pwd))
			if err != nil {
				return core.NullValue, err
			}
			return core.NewBytes(b), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.decrypt",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			var pwd string
			switch len(args) {
			case 0:
				return core.NullValue, fmt.Errorf("expected 1 argument, got 0")
			case 1:
				pwd = GetContext(vm).getProtectedItem("password").ToString()
				if pwd == "" {
					pwd = globalPassword
					if pwd == "" {
						return core.NullValue, fmt.Errorf("no password configured")
					}
				}

			case 2:
				pwd = args[1].ToString()
			}

			b, err := Decrypt(args[0].ToBytes(), []byte(pwd))
			if err != nil {
				return core.NullValue, err
			}
			return core.NewBytes(b), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.encryptTripleDES",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Bytes); err != nil {
				return core.NullValue, err
			}
			b, err := EncryptTripleDESCBC(args[0].ToBytes(), args[1].ToBytes())
			if err != nil {
				return core.NullValue, err
			}
			return core.NewBytes(b), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.decryptTripleDES",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Bytes); err != nil {
				return core.NullValue, err
			}
			b, err := DecryptTripleDESCBC(args[0].ToBytes(), args[1].ToBytes())
			if err != nil {
				return core.NullValue, err
			}
			return core.NewBytes(b), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.hashPassword",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			s := HashPassword(args[0].ToString())
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.compareHashAndPassword",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			ok := CheckHashPasword(args[0].ToString(), args[1].ToString())
			return core.NewBool(ok), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.random",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			b := Random(int(args[0].ToInt()))
			return core.NewBytes(b), nil
		},
	},
	core.NativeFunction{
		Name:      "crypto.randomAlphanumeric",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			ln := int(args[0].ToInt())
			if ln < 1 {
				return core.NullValue, fmt.Errorf("invalid len: %d", ln)
			}
			s := RandomAlphanumeric(ln)
			return core.NewString(s), nil
		},
	},
}

const saltLen = 18

func HashPassword(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		// this should only happen if the factor is invalid, but we know it is ok
		panic(err)
	}
	return string(h)
}

func CheckHashPasword(hash, pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)) == nil
}

// Encrypts encrypts the text.
func Encrypts(text, password string) (string, error) {
	e, err := Encrypt([]byte(text), []byte(password))
	if err != nil {
		return "", err
	}

	encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	return encoder.EncodeToString(e), nil
}

// Decrypts decrypts the text.
func Decrypts(text, password string) (string, error) {
	encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	e, err := encoder.DecodeString(text)
	if err != nil {
		return "", err
	}

	d, err := Decrypt(e, []byte(password))
	if err != nil {
		return "", err
	}

	return string(d), err
}

func EncryptTripleDESCBC(decrypted, key []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, err
	}

	iv := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	blockMode := cipher.NewCBCEncrypter(block, iv)

	decrypted = ZeroPadding(decrypted, block.BlockSize())
	encrypted := make([]byte, len(decrypted))
	blockMode.CryptBlocks(encrypted, decrypted)
	return encrypted, nil
}

func DecryptTripleDESCBC(encrypted, key []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, err
	}

	iv := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	blockMode := cipher.NewCBCDecrypter(block, iv)

	decrypted := make([]byte, len(encrypted))
	blockMode.CryptBlocks(decrypted, encrypted)
	decrypted = ZeroUnPadding(decrypted)
	return decrypted, nil
}

func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData,
		func(r rune) bool {
			return r == rune(0)
		})
}

// Encrypts encrypts the text.
func Encrypt(plaintext, password []byte) ([]byte, error) {
	key, salt := generateFromPassword(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return append(salt, gcm.Seal(nonce, nonce, plaintext, nil)...), nil
}

// Decrypts decrypts the text.
func Decrypt(ciphertext, password []byte) ([]byte, error) {
	salt, c, err := decode(ciphertext)
	if err != nil {
		return nil, err
	}

	key := generateFromPasswordAndSalt(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, c[:gcm.NonceSize()], c[gcm.NonceSize():], nil)
}

// decode returns the salt and cipertext
func decode(ciphertext []byte) ([]byte, []byte, error) {
	if len(ciphertext) < saltLen {
		return nil, nil, fmt.Errorf("invalid ciphertext")
	}
	return ciphertext[:saltLen], ciphertext[saltLen:], nil
}

func generateFromPasswordAndSalt(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, 4096, 32, sha1.New)
}

// generateFromPassword returns the key and the salt.
//
// https://github.com/golang/crypto/blob/master/pbkdf2/pbkdf2.go
//
// dk := pbkdf2.Key([]byte("some password"), salt, 4096, 32, sha1.New)
//
func generateFromPassword(password []byte) ([]byte, []byte) {
	salt := Random(saltLen)
	dk := pbkdf2.Key(password, salt, 4096, 32, sha1.New)
	return dk, salt
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func Random(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

func RandomAlphanumeric(size int) string {
	dictionary := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	l := byte(len(dictionary))
	var b = make([]byte, size)
	rand.Read(b)
	for k, v := range b {
		b[k] = dictionary[v%l]
	}
	return string(b)
}
