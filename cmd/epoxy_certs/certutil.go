// Package main implements utility methods for managing x509 certificates and RSA keys.
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

var ErrNoCertFound = errors.New("no cert found")

// NewCertPool creates a new x509.CertPool from all PEM encoded certificates in pemFile.
func NewCertPool(pemFile string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(pemBytes); !ok {
		return nil, fmt.Errorf("no certificates loaded from %s", pemFile)
	}
	return certPool, nil
}

// ReadCertFile reads a PEM encoded certificate from certFile.
func ReadCertFileOld(certFile string) (*x509.Certificate, error) {
	derBytes, err := readPEMFile(certFile)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

type CertReader struct {
	err error
}

func (cr *CertReader) readFile(f string) []byte {
	pemBytes, err := os.ReadFile(f)
	if err != nil {
		cr.err = err
	}
	return pemBytes
}

func (cr *CertReader) decode(p []byte) []byte {
	if cr.err != nil {
		return nil
	}
	block, _ := pem.Decode(p)
	if block == nil {
		cr.err = ErrNoCertFound
		return nil
	}
	return block.Bytes
}

func (cr *CertReader) parseCertificate(b []byte) *x509.Certificate {
	if cr.err != nil {
		return nil
	}
	cert, err := x509.ParseCertificate(b)
	if err != nil {
		cr.err = err
	}
	return cert
}

func ReadCertFile(certFile string) (*x509.Certificate, error) {
	cr := &CertReader{}
	cert := cr.parseCertificate(cr.decode(cr.readFile(certFile)))
	return cert, cr.err
}

// WriteCertFile writes a PEM encoded certificate to certFile.
func WriteCertFile(derBytes []byte, certFile string) error {
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return err
	}
	return nil
}

// ReadRSAKeyFile reads a PEM encoded, PKCS1 format private RSA key from keyFile.
func ReadRSAKeyFile(keyFile string) (*rsa.PrivateKey, error) {
	derBytes, err := readPEMFile(keyFile)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(derBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// WriteRSAKeyFile writes a PEM encoded, PKCS1 format private RSA key to keyFile.
func WriteRSAKeyFile(privKey *rsa.PrivateKey, keyFile string) error {
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})
	if err != nil {
		return err
	}
	return nil
}

// readPEMFile returns the first PEM block from pemFile.
func readPEMFile(pemFile string) ([]byte, error) {
	pemBytes, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}
	// Only read the first block.
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", pemFile)
	}
	return block.Bytes, nil
}
