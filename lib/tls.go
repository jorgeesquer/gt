package lib

import (
	"crypto/tls"
	"fmt"
	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(TLS, `

declare namespace tls {
    export function newConfig(insecureSkipVerify?: boolean): Config

    export interface Config {
        insecureSkipVerify: boolean
        loadCertificate(certPath: string, keyPath: string): void
        loadCertificateData(cert: byte[] | string, key: byte[] | string): void
        buildNameToCertificate(): void
    }
}

`)
}

var TLS = []core.NativeFunction{
	core.NativeFunction{
		Name:      "tls.newConfig",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgRange(args, 0, 1); err != nil {
				return core.NullValue, err
			}
			if err := ValidateOptionalArgs(args, core.Bool); err != nil {
				return core.NullValue, err
			}

			tc := &tlsConfig{
				conf: &tls.Config{
					MinVersion:               tls.VersionTLS12,
					CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
					PreferServerCipherSuites: true,
				},
			}

			if len(args) == 1 {
				tc.conf.InsecureSkipVerify = args[0].ToBool()
			}

			return core.NewObject(tc), nil
		},
	},
}

type tlsConfig struct {
	conf *tls.Config
}

func (t *tlsConfig) Type() string {
	return "tls.Config"
}

func (t *tlsConfig) GetMethod(name string) core.NativeMethod {
	switch name {
	case "loadCertificate":
		return t.loadCertificate
	case "loadCertificateData":
		return t.loadCertificateData
	case "buildNameToCertificate":
		return t.buildNameToCertificate
	}
	return nil
}

func (t *tlsConfig) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "insecureSkipVerify":
		return core.NewBool(t.conf.InsecureSkipVerify), nil
	}
	return core.UndefinedValue, nil
}

func (t *tlsConfig) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "insecureSkipVerify":
		if v.Type != core.Bool {
			return fmt.Errorf("invalid type, expected bool")
		}
		t.conf.InsecureSkipVerify = v.ToBool()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (t *tlsConfig) buildNameToCertificate(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	t.conf.BuildNameToCertificate()

	return core.NullValue, nil
}

func (t *tlsConfig) loadCertificate(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	fs := vm.FileSystem
	if fs == nil {
		return core.NullValue, fmt.Errorf("there is no filesystem set")
	}

	certPath := args[0].ToString()
	keyPath := args[1].ToString()

	certPEMBlock, err := filesystem.ReadAll(fs, certPath)
	if err != nil {
		return core.NullValue, fmt.Errorf("error reading cert %s: %v", certPath, err)
	}

	keyPEMBlock, err := filesystem.ReadAll(fs, keyPath)
	if err != nil {
		return core.NullValue, fmt.Errorf("error reading key %s: %v", keyPath, err)
	}

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return core.NullValue, fmt.Errorf("error creating X509KeyPair: %v", err)
	}

	t.conf.Certificates = append(t.conf.Certificates, cert)

	return core.NullValue, nil
}

func (t *tlsConfig) loadCertificateData(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 2, 2); err != nil {
		return core.NullValue, err
	}

	certPEMBlock := args[0]
	keyPEMBlock := args[1]

	switch certPEMBlock.Type {
	case core.String, core.Bytes:
	default:
		return core.NullValue, fmt.Errorf("expected cert of type string or bytes, got: %s", certPEMBlock.TypeName())
	}

	switch keyPEMBlock.Type {
	case core.String, core.Bytes:
	default:
		return core.NullValue, fmt.Errorf("expected key of type string or bytes, got: %s", keyPEMBlock.TypeName())
	}

	cert, err := tls.X509KeyPair(certPEMBlock.ToBytes(), keyPEMBlock.ToBytes())
	if err != nil {
		return core.NullValue, fmt.Errorf("error creating X509KeyPair: %v", err)
	}

	t.conf.Certificates = append(t.conf.Certificates, cert)

	return core.NullValue, nil
}
