package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var helpMessage = `NAME:
  epoxycerts - A tool for creating TLS certificates for the ePoxy server and
  clients.

DESCRIPTION:
  The ePoxy system uses a private root CA. This CA is used to issue all
  server and client certificates.

  For better security, the root CA should be kept offline.

EXAMPLES:
  epoxy_certs ca --hostname ca.example.com

  epoxy_certs server --hostname server.example.com

  epoxy_certs clientca --hostname client-ca.example.com

  epoxy_certs client --hostname client1.example.com

USAGE:
`

var Usage = func() {
	fmt.Fprint(os.Stderr, helpMessage)
	flag.PrintDefaults()
}

func init() {
	flag.Usage = Usage
}

var (
	// cert creation.
	hostname       = flag.String("hostname", "", "A single hostname (or IP) for the certificate CommonName.")
	extraHostnames = flag.String("extraNames", "", "Comma-separated list of hostnames or IPs in addition to hostname.")
	startDate      = flag.String("start-date", "", "Date when certificate becomes valid. Format as: Jan 1 15:04:05 2011")
	validFor       = flag.Duration("duration", 365*24*time.Hour, "Duration that the certificate will be valid.")
	orgName        = flag.String("org-name", "", "Name of organization that uses this certificate.")
	bitSize        = flag.Int("bit-size", 2048, "Size of RSA key to generate.")

	caCertFile = flag.String("ca-cert", "ca-cert.pem", "The CA certificate in PEM format.")
	caKeyFile  = flag.String("ca-key", "ca-key.pem", "The CA private key in PEM format.")

	serverCertFile = flag.String("server-cert", "server-cert.pem", "The server certificate in PEM format.")
	serverKeyFile  = flag.String("server-key", "server-key.pem", "The server private key in PEM format.")

	clientIssuerCertFile = flag.String("clientca-cert", "clientca-cert.pem", "The client issuer certificate in PEM format.")
	clientIssuerKeyFile  = flag.String("clientca-key", "clientca-key.pem", "The client issuer private key in PEM format.")

	clientCertFile = flag.String("client-cert", "client-cert.pem", "The client certificate in PEM format.")
	clientKeyFile  = flag.String("client-key", "client-key.pem", "The client private key in PEM format.")
)

func checkFlags() string {
	// Replaces flag.Parse()
	if len(os.Args) == 1 || os.Args[1][0] == '-' {
		flag.Usage()
		os.Exit(1)
	}
	opt := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])

	// Check for required values.
	if len(*hostname) == 0 && opt != "ca" {
		log.Fatalf("Missing required --hostname parameter.")
	}
	return opt
}

func parseStartDate(startDate string) (notBefore time.Time) {
	if len(startDate) == 0 {
		notBefore = time.Now()
		// Subtract 24 hours to prevent "certificate not yet valid" authentication errors.
		notBefore = notBefore.Add(-48 * time.Hour)
	} else {
		var err error
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", startDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse creation date: %s\n", err)
			os.Exit(1)
		}
	}
	return
}

func addHostsToCertificate(cert *x509.Certificate, hostnames []string) {
	for _, h := range hostnames {
		if len(h) == 0 {
			continue
		}
		if ip := net.ParseIP(h); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, h)
		}
	}
}

func getSerialNumber() *big.Int {
	// TODO: what's a better value here?
	t := time.Now()
	serialStr := fmt.Sprintf("%04d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	serialNumber, err := strconv.ParseInt(serialStr, 10, 64)
	if err != nil {
		log.Fatalf("Failed to generate serial number from %s: %s", serialStr, err)
	}
	return big.NewInt(serialNumber)
}

func getBasicCertificate(hostname string, extraHostnames []string) (c *x509.Certificate) {
	notBefore := parseStartDate(*startDate)
	c = &x509.Certificate{
		SerialNumber: getSerialNumber(),
		Subject: pkix.Name{
			Organization: []string{*orgName},
		},

		NotBefore: notBefore,
		NotAfter:  notBefore.Add(*validFor),

		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}
	if len(hostname) != 0 {
		c.Subject.CommonName = hostname
		addHostsToCertificate(c, []string{hostname})
	}
	addHostsToCertificate(c, extraHostnames)
	return
}

func selfSignCert(c *x509.Certificate, certFile, keyFile string) {
	newPrivKey, err := rsa.GenerateKey(rand.Reader, *bitSize)
	if err != nil {
		log.Fatalf("Failed to generate private key: %s", err)
	}

	derBytesCert, err := x509.CreateCertificate(rand.Reader, c, c, &newPrivKey.PublicKey, newPrivKey)
	if err != nil {
		log.Fatalf("Failed to xcreate certificate: %s", err)
	}

	// Write cert and key to file.
	WriteCertFile(derBytesCert, certFile)
	WriteRSAKeyFile(newPrivKey, keyFile)
}

func signCert(c *x509.Certificate, certFile, keyFile, signerCertFile, signerKeyFile string) {
	signerCert, err := ReadCertFile(signerCertFile)
	if err != nil {
		log.Fatalf("Failed to read signer certificate: %s", err)
	}

	signerKey, err := ReadRSAKeyFile(signerKeyFile)
	if err != nil {
		log.Fatalf("Failed to read signer private key: %s", err)
	}

	newPrivKey, err := rsa.GenerateKey(rand.Reader, *bitSize)
	if err != nil {
		log.Fatalf("Failed to generate private key: %s", err)
	}

	// Create a certificate signed by the signer.
	derBytesCert, err := x509.CreateCertificate(rand.Reader, c, signerCert, &newPrivKey.PublicKey, signerKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	// Write cert to file.
	WriteCertFile(derBytesCert, certFile)
	WriteRSAKeyFile(newPrivKey, keyFile)
}

func createRootCACert(c *x509.Certificate) {
	c.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	c.IsCA = true
	// Allow one intermediate CA below the root CA for issuing client certificates.
	c.MaxPathLen = 1

	selfSignCert(c, *caCertFile, *caKeyFile)
}

func createServerCert(c *x509.Certificate) {
	c.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	c.ExtKeyUsage = append(c.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	signCert(c, *serverCertFile, *serverKeyFile, *caCertFile, *caKeyFile)
}

func createIssuerClientCert(c *x509.Certificate) {
	c.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	c.IsCA = true
	// Restrict this certificate to only issuing certificates.
	c.MaxPathLenZero = true

	signCert(c, *clientIssuerCertFile, *clientIssuerKeyFile, *caCertFile, *caKeyFile)
}

func createClientCert(c *x509.Certificate) {
	c.ExtKeyUsage = append(c.ExtKeyUsage, x509.ExtKeyUsageClientAuth)

	signCert(c, *clientCertFile, *clientKeyFile, *clientIssuerCertFile, *clientIssuerKeyFile)
}

func main() {
	opt := checkFlags()

	c := getBasicCertificate(*hostname, strings.Split(*extraHostnames, ","))
	switch opt {
	case "ca":
		createRootCACert(c)
	case "clientca":
		createIssuerClientCert(c)
	case "client":
		createClientCert(c)
	case "server":
		createServerCert(c)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
