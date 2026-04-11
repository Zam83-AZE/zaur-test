package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
)

func needsCertRegeneration(certFile string) bool {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return !cert.IsCA
}
