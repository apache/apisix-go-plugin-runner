package plugins_test

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/apache/apisix-go-plugin-runner/cmd/go-runner/plugins"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	"github.com/stretchr/testify/assert"
)

type MockAccessVerifier struct {
	T *testing.T
}

func (m *MockAccessVerifier) Verify(conf *plugins.DataNodeAccessControlConf, path, apiKey string) (bool, error) {
	t := m.T

	assert.Equal(t, conf.VerifyURL, "http://test.xyz")
	assert.Equal(t, conf.ServiceAccountToken, "test-token")
	assert.Equal(t, apiKey, "test-key")
	assert.Equal(t, path, "https://bazinga.xyz")

	return true, nil
}

// TODO: add test case for checking api key in header
func TestDataNodeAccessControl(t *testing.T) {
	in := []byte(`{"verify-url":"http://test.xyz","service-account-token":"test-token"}`)
	dnac := plugins.NewDataNodeAccessControl(&MockAccessVerifier{t})
	conf, err := dnac.ParseConf(in)
	if err != nil {
		t.Errorf("failed to parse conf: %s", err)
	}

	req := &MockHTTPRequest{}
	req.Args().Set("apikey", "test-key")
	dnac.RequestFilter(conf, httptest.NewRecorder(), req)
}

type MockHTTPRequest struct {
	pkgHTTP.Request
}

func (m *MockHTTPRequest) Args() url.Values {
	args := url.Values{}
	args.Set("apikey", "test-key")
	return args
}

func (m *MockHTTPRequest) Var(key string) ([]byte, error) {
	mock := map[string][]byte{
		"scheme": []byte("https"),
		"host":   []byte("bazinga.xyz"),
	}
	return mock[key], nil
}
