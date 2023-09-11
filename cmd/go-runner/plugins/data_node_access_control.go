package plugins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
)

type DataNodeAccessControl struct {
	v AccessVerifier
}

type DataNodeAccessControlConf struct {
	// Route level config
	VerifyURL           string `json:"verify-url"`
	ServiceAccountToken string `json:"service-account-token"`
}

type DataNodeAccessControlResponse struct {
	NodeURLS []string `json:"nodeUrls"`
}

type AccessVerifier interface {
	Verify(conf *DataNodeAccessControlConf, path, apiKey string) (bool, error)
}

func init() {
	err := plugin.RegisterPlugin(&DataNodeAccessControl{&APIVerify{}})
	if err != nil {
		log.Fatalf("failed to register plugin data-node-access-control: %s", err)
	}
}

func NewDataNodeAccessControl(v AccessVerifier) *DataNodeAccessControl {
	return &DataNodeAccessControl{v}
}

func (p *DataNodeAccessControl) Name() string {
	return "data-node-access-control"
}

func (p *DataNodeAccessControl) ParseConf(in []byte) (interface{}, error) {
	conf := DataNodeAccessControlConf{}
	err := json.Unmarshal(in, &conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (p *DataNodeAccessControl) RequestFilter(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
	parsedConf := conf.(DataNodeAccessControlConf)

	apiKey := getAPIKey(r)
	isAllowed, err := p.v.Verify(&parsedConf, string(r.Path()), apiKey)
	if err != nil {
		writeHeader(w, http.StatusServiceUnavailable, "service unavailable", err)
		return
	}

	if !isAllowed {
		writeHeader(w, http.StatusForbidden, "forbidden", nil)
		return
	}
}

func (p *DataNodeAccessControl) ResponseFilter(conf interface{}, w pkgHTTP.Response) {}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func writeHeader(w http.ResponseWriter, status int, msg string, err error) {
	log.Errorf("%s: %s", msg, err)
	w.WriteHeader(status)
}

// getAPIKey either gets from query param apikey=xxx or get from header apikey
func getAPIKey(r pkgHTTP.Request) string {
	apiKey := r.Args().Get("apikey")
	if apiKey == "" {
		apiKey = r.Header().Get("apikey")
	}

	return apiKey
}

type APIVerify struct{}

func (v *APIVerify) Verify(conf *DataNodeAccessControlConf, path, apiKey string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := conf.VerifyURL
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	req.Header.Set("Authorization", "Bearer "+conf.ServiceAccountToken)
	req.Header.Set("x-api-key", apiKey)
	if err != nil {
		return false, err
	}

	// make GET request to VerifyURL with ServiceAccountToken
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	if res != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != http.StatusOK {
		return false, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	var s *DataNodeAccessControlResponse
	err = json.Unmarshal(body, &s)
	if err != nil {
		return false, err
	}

	if !contains(s.NodeURLS, path) {
		return false, nil
	}

	return true, nil
}
