package ignite

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	testing2 "sbt.ru/ignite-go/ignite/internal/testing"
	"strings"
	"testing"
)

const (
	encryptedCertsPath = "./internal/testing/config/ssl/client_with_pass_full.pem"
	certsPath          = "./internal/testing/config/ssl/client_full.pem"
	certPwd            = "654321"
)

type TlsTestSuite struct {
	suite.Suite
	grids []testing2.IgniteInstance
}

func TestTlsTestSuite(t *testing.T) {
	suite.Run(t, new(TlsTestSuite))
}

func (suite *TlsTestSuite) SetupSuite() {
	var err error
	suite.grids = make([]testing2.IgniteInstance, 0)
	ign, err := testing2.StartIgnite(testing2.WithSsl(), testing2.WithAuth())
	if err != nil {
		suite.T().Errorf("Failed to startClient suite: %s", err.Error())
	}
	suite.grids = append(suite.grids, ign)
}

func (suite *TlsTestSuite) TearDownSuite() {
	if suite.grids != nil {
		for _, ign := range suite.grids {
			_ = ign.Kill()
		}
		clear(suite.grids)
	}
}

func (suite *TlsTestSuite) TestCreationPolicy() {
	fixtures := []struct {
		name     string
		supplier func() (*tls.Config, error)
	}{
		{"with_encrypted", createTlsSupplier(encryptedCertsPath, certPwd)},
		{"basic", createTlsSupplier(certsPath, "")},
	}

	for _, fixture := range fixtures {
		suite.T().Run(fixture.name, func(t *testing.T) {
			cli, err := Start(WithAddresses(defaultAddress), WithTls(fixture.supplier),
				WithClientAttribute("clientName", "ignite-go"),
				WithCredentials("ignite", "ignite"))
			defer func() {
				_ = cli.Close()
			}()

			assert.Nil(t, err)
			caches, err := cli.CacheNames(context.Background())
			assert.Nil(t, err)
			assert.True(t, len(caches) == 0)
		})
	}
}

func createTlsSupplier(certPath string, password string) func() (*tls.Config, error) {
	return func() (*tls.Config, error) {
		certFileName := certPath
		certData, err0 := os.ReadFile(certFileName)
		if err0 != nil {
			return nil, fmt.Errorf("failed to open caCert %s: %w", certFileName, err0)
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certData); !ok {
			return nil, fmt.Errorf("failed to load cert data %s", certFileName)
		}
		cert := tls.Certificate{}
		for {
			var derBlock *pem.Block
			derBlock, certData = pem.Decode(certData)
			if derBlock == nil {
				break
			}
			if derBlock.Type == "CERTIFICATE" {
				cert.Certificate = append(cert.Certificate, derBlock.Bytes)
			} else if derBlock.Type == "PRIVATE KEY" || strings.HasSuffix(derBlock.Type, " PRIVATE KEY") {
				var keyBlock []byte
				if x509.IsEncryptedPEMBlock(derBlock) {
					derBlock.Bytes, err0 = x509.DecryptPEMBlock(derBlock, []byte(password))
					if err0 != nil {
						return nil, fmt.Errorf("failed to decrypt private key %s: %w", certFileName, err0)
					}
					keyBlock = derBlock.Bytes
				} else {
					keyBlock = derBlock.Bytes
				}
				cert.PrivateKey, err0 = parsePrivateKey(keyBlock)
				if err0 != nil {
					return nil, fmt.Errorf("failed to parse private key %s: %w", certFileName, err0)
				}
			}
		}
		return &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            caCertPool,
			Certificates:       []tls.Certificate{cert},
		}, nil
	}
}

func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("tls: failed to parse private key")
}
