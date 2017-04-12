package tun

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/pkg/errors"
)

// generate:
//    openssl genrsa -out key.pem 2048
//    openssl req -new -x509 -days 365 -key key.pem -subj "/C=CN/CN=localhost" -out cert.crt
//      openssl req -new -key key.pem -subj "/C=CN/CN=localhost" -out req.csr
//      openssl x509 -req -days 365 -in req.csr -signkey key.pem -out cert.crt
// check: certutil -d "sql:$HOME/.pki/nssdb" -L -n Ghost
// import: certutil -d "sql:$HOME/.pki/nssdb" -A -n Ghost -i cert.pem -t "C,,"
// delete: certutil -d "sql:$HOME/.pki/nssdb" -D -n Ghost
func CreateCertificate(isCA bool, certFile, keyFile string) error {
	// key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return errors.Wrap(err, "failed to generate private key")
	}
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.WithStack(err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Println("written ", keyFile)

	// cert
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return errors.Wrap(err, "failed to generate serialNumber")
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:    []string{"CN"},
			CommonName: "localhost",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
	if isCA {
		template.BasicConstraintsValid = true
		template.IsCA = true
		log.Println("cert is CA")
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template,
		&priv.PublicKey, priv)
	if err != nil {
		return errors.Wrap(err, "failed to create certificate")
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return errors.WithStack(err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Println("written ", certFile)

	return nil
}
