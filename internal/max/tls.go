package max

import(
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed certs/russian_trusted_root_ca.cer
var rootCA []byte

//go:embed certs/russian_trusted_sub_ca.cer
var subCA []byte

func NewTrustedHTTPClient() (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}

	if !pool.AppendCertsFromPEM(rootCA) {
		return nil, fmt.Errorf("failed to append root CA")
	}
	if !pool.AppendCertsFromPEM(subCA) {
		return nil, fmt.Errorf("failed to append sub CA")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}

	return &http.Client{Transport: transport}, nil
}