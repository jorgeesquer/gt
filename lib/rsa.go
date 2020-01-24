package lib

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(RSA, `

declare namespace rsa {
    export function generateKey(size?: number): PrivateKey
    export function decodePEMKey(key: string | byte[]): PrivateKey
    export function decodePublicPEMKey(key: string | byte[]): PublicKey
    export function signPKCS1v15(key: PrivateKey, mesage: string | byte[]): byte[]
    export function verifyPKCS1v15(key: PublicKey, mesage: string | byte[], signature: string | byte[]): boolean

    interface PrivateKey {
        publicKey: PublicKey
        encodePEMKey(): byte[]
        encodePublicPEMKey(): byte[]
    }

    interface PublicKey {

    }
}

`)
}

var RSA = []core.NativeFunction{
	core.NativeFunction{
		Name:      "rsa.generateKey",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			reader := rand.Reader

			var bitSize int
			if len(args) == 0 {
				bitSize = 2048
			} else {
				bitSize = int(args[0].ToInt())
			}

			key, err := rsa.GenerateKey(reader, bitSize)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&rsaPrivateKey{key}), nil
		},
	},
	core.NativeFunction{
		Name:      "rsa.decodePEMKey",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].ToBytes()

			block, _ := pem.Decode(v)

			if block == nil {
				return core.NullValue, fmt.Errorf("error decoding private key")
			}

			enc := x509.IsEncryptedPEMBlock(block)

			b := block.Bytes

			var err error
			if enc {
				b, err = x509.DecryptPEMBlock(block, nil)
				if err != nil {
					return core.NullValue, fmt.Errorf("error decrypting private key")
				}
			}

			key, err := x509.ParsePKCS1PrivateKey(b)
			if err != nil {
				return core.NullValue, fmt.Errorf("error parsing private key: %v", err)
			}

			return core.NewObject(&rsaPrivateKey{key}), nil
		},
	},
	core.NativeFunction{
		Name:      "rsa.decodePublicPEMKey",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].ToBytes()

			block, _ := pem.Decode(v)

			if block == nil {
				return core.NullValue, fmt.Errorf("error decoding public key")
			}

			enc := x509.IsEncryptedPEMBlock(block)

			b := block.Bytes

			var err error
			if enc {
				b, err = x509.DecryptPEMBlock(block, nil)
				if err != nil {
					return core.NullValue, fmt.Errorf("error decrypting public key")
				}
			}
			ifc, err := x509.ParsePKIXPublicKey(b)
			if err != nil {
				return core.NullValue, fmt.Errorf("error parsing public key: %v", err)
			}

			key, ok := ifc.(*rsa.PublicKey)
			if !ok {
				return core.NullValue, fmt.Errorf("not an RSA public key")
			}

			return core.NewObject(&rsaPublicKey{key}), nil
		},
	},
	core.NativeFunction{
		Name:      "rsa.signPKCS1v15",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			key, ok := args[0].ToObjectOrNil().(*rsaPrivateKey)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a rsa key, got %v", args[0].TypeName())
			}

			message := args[1].ToBytes()

			// Only small messages can be signed directly; thus the hash of a
			// message, rather than the message itself, is signed. This requires
			// that the hash function be collision resistant. SHA-256 is the
			// least-strong hash function that should be used for this at the time
			// of writing (2016).
			hashed := sha256.Sum256(message)

			rng := rand.Reader

			signature, err := rsa.SignPKCS1v15(rng, key.key, crypto.SHA256, hashed[:])
			if err != nil {
				return core.NullValue, err
			}

			return core.NewBytes(signature), nil
		},
	},
	core.NativeFunction{
		Name:      "rsa.verifyPKCS1v15",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			key, ok := args[0].ToObjectOrNil().(*rsaPublicKey)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a rsa key, got %v", args[0].TypeName())
			}

			message := args[1].ToBytes()
			signature := args[2].ToBytes()

			// Only small messages can be signed directly; thus the hash of a
			// message, rather than the message itself, is signed. This requires
			// that the hash function be collision resistant. SHA-256 is the
			// least-strong hash function that should be used for this at the time
			// of writing (2016).
			hashed := sha256.Sum256(message)

			err := rsa.VerifyPKCS1v15(key.key, crypto.SHA256, hashed[:], signature)
			if err != nil {
				return core.FalseValue, err
			}

			return core.TrueValue, err
		},
	},
}

type rsaPrivateKey struct {
	key *rsa.PrivateKey
}

func (k *rsaPrivateKey) Type() string {
	return "RSA_Private_Key"
}

func (k *rsaPrivateKey) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "publicKey":
		return core.NewObject(&rsaPublicKey{&k.key.PublicKey}), nil
	}

	return core.UndefinedValue, nil
}

func (k *rsaPrivateKey) GetMethod(name string) core.NativeMethod {
	switch name {
	case "encodePEMKey":
		return k.encodePEMKey
	case "encodePublicPEMKey":
		return k.encodePublicPEMKey
	}
	return nil
}

func (k *rsaPrivateKey) encodePEMKey(args []core.Value, vm *core.VM) (core.Value, error) {
	b := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k.key),
		},
	)
	return core.NewBytes(b), nil
}

func (k *rsaPrivateKey) encodePublicPEMKey(args []core.Value, vm *core.VM) (core.Value, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(k.key.Public())
	if err != nil {
		return core.NullValue, err
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	return core.NewBytes(b), nil
}

type rsaPublicKey struct {
	key *rsa.PublicKey
}

func (k *rsaPublicKey) Type() string {
	return "RSA_Private_Key"
}
