package services

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// KeyVaultAdapter unwraps encrypted symmetric keys from CLI using Marketplace private key.
type KeyVaultAdapter struct {
	PrivateKey *rsa.PrivateKey
}

func NewKeyVaultAdapter(pemBytes []byte) (*KeyVaultAdapter, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &KeyVaultAdapter{PrivateKey: key}, nil
}

func (k *KeyVaultAdapter) Unwrap(wrapped []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(nil, k.PrivateKey, wrapped)
}
