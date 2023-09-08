package plugins_test

import (
	"net/http/httptest"
	"testing"

	"github.com/apache/apisix-go-plugin-runner/cmd/go-runner/plugins"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	"github.com/stretchr/testify/assert"
)

type MockAccessVerifier struct {
	T *testing.T
}

func (m *MockAccessVerifier) Verify(conf *plugins.DataNodeAccessControlConf, path string) (bool, error) {
	t := m.T

	assert.Equal(t, conf.ConsumerName, "test")
	assert.Equal(t, conf.VerifyURL, "http://test.xyz")
	assert.Equal(t, conf.ServiceAccountToken, "test-token")

	return true, nil
}

func TestDataNodeAccessControl(t *testing.T) {
	in := []byte(`{"consumer-name":"test","verify-url":"http://test.xyz","service-account-token":"test-token"}`)
	dnac := plugins.NewDataNodeAccessControl(&MockAccessVerifier{t})
	conf, err := dnac.ParseConf(in)
	if err != nil {
		t.Errorf("failed to parse conf: %s", err)
	}

	req := &pkgHTTP.Request{}
	req.SetPath([]byte("http://test.xyz"))
	dnac.RequestFilter(conf, httptest.NewRecorder(), req)
}
