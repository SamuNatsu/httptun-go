package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

type KeyPair struct {
	Cert []byte
	Priv []byte
}

func GenerateKeyPair() (KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return KeyPair{}, err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.DNSNames = append(template.DNSNames, "localhost")
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return KeyPair{}, err
	}

	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pemPriv := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return KeyPair{Cert: pemCert, Priv: pemPriv}, nil
}

func WriteKeyPair(pair KeyPair, prefix string) error {
	certOut, err := os.Create(fmt.Sprintf("%s.cert", prefix))
	if err != nil {
		return err
	}

	_, err = certOut.Write(pair.Cert)
	if err != nil {
		return err
	}

	if err := certOut.Close(); err != nil {
		return err
	}

	keyOut, err := os.Create(fmt.Sprintf("%s.key", prefix))
	if err != nil {
		return err
	}

	_, err = keyOut.Write(pair.Priv)
	if err != nil {
		return err
	}

	if err := keyOut.Close(); err != nil {
		return err
	}

	return nil
}
