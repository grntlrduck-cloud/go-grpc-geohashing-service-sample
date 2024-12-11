package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega" //nolint:stylecheck
)

const (
	basePath      = "api/v1/pois"
	proximityPath = "proximity"
	bboxPath      = "bbox"
	routePath     = "route"
	infoPath      = "info" // the only endpoint where we have a path variable
)

type HTTPResponse[T any] struct {
	Ok         *T
	Err        *HTTPErrorResponse
	StatusCode int
}

type PoIsHTTPResponse struct {
	Items []PoIHTTP `json:"items"`
}

type HTTPErrorResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type PoIHTTPResponse struct {
	Poi PoIHTTP `json:"poi"`
}

type PoIHTTP struct {
	ID         string          `json:"id"`
	Coordinate CoordinatesHTTP `json:"coordinate"`
	Entrance   CoordinatesHTTP `json:"entrance"`
	Address    AddressHTTP     `json:"address"`
	Features   []string        `json:"features"`
}

type CoordinatesHTTP struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type AddressHTTP struct {
	Street       string `json:"street"`
	StreetNumber string `json:"street_number"`
	City         string `json:"city"`
	ZipCode      string `json:"zip_code"`
	Country      string `json:"country"`
}

type PoIHTTPProxyClient struct {
	host    string
	port    int32
	baseURI string
	client  http.Client
}

func (p *PoIHTTPProxyClient) Info(
	id string,
	correlation,
	apiKey bool,
	apiKeyOverride string,
) *HTTPResponse[PoIHTTPResponse] {
	url := fmt.Sprintf("%s/%s/%s", p.baseURI, infoPath, id)
	req, err := http.NewRequest( //nolint:noctx // no production code
		http.MethodGet,
		url,
		http.NoBody,
	)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation, apiKey, apiKeyOverride)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHTTPResponse[PoIHTTPResponse](res)
}

func (p *PoIHTTPProxyClient) Bbox(
	ne, sw CoordinatesHTTP,
	correlation,
	apiKey bool,
	apiKeyOverride string,
) *HTTPResponse[PoIsHTTPResponse] {
	url := fmt.Sprintf(
		"%s/%s?bbox.ne.lat=%f&bbox.ne.lon=%f&bbox.sw.lat=%f&bbox.sw.lon=%f",
		p.baseURI,
		bboxPath,
		ne.Lat,
		ne.Lon,
		sw.Lat,
		sw.Lon,
	)
	//nolint:noctx // no production code
	req, err := http.NewRequest(
		http.MethodGet,
		url,
		http.NoBody,
	)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation, apiKey, apiKeyOverride)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHTTPResponse[PoIsHTTPResponse](res)
}

func (p *PoIHTTPProxyClient) Prxoimity(
	center CoordinatesHTTP,
	radiusMeters float64,
	correlation,
	apiKey bool,
	apiKeyOverride string,
) *HTTPResponse[PoIsHTTPResponse] {
	url := fmt.Sprintf(
		"%s/%s?center.lat=%f&center.lon=%f&radius_meters=%f",
		p.baseURI,
		proximityPath,
		center.Lat,
		center.Lon,
		radiusMeters,
	)
	req, err := http.NewRequest( //nolint:noctx // no production code
		http.MethodGet,
		url,
		http.NoBody,
	)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation, apiKey, apiKeyOverride)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHTTPResponse[PoIsHTTPResponse](res)
}

func (p *PoIHTTPProxyClient) Route(
	route []CoordinatesHTTP,
	correlation,
	apiKey bool,
	apiKeyOverride string,
) *HTTPResponse[PoIsHTTPResponse] {
	url := fmt.Sprintf("%s/%s", p.baseURI, routePath)
	encRoute, err := json.Marshal(route)
	Expect(err).To(Not(HaveOccurred()))
	reader := bytes.NewReader(encRoute)
	req, err := http.NewRequest(http.MethodPost, url, reader) //nolint:noctx // no production code
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation, apiKey, apiKeyOverride)

	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHTTPResponse[PoIsHTTPResponse](res)
}

func NewPoIHTTPProxyClient(host string, port int32) *PoIHTTPProxyClient {
	uri := fmt.Sprintf("http://%s:%d/%s", host, port, basePath)
	client := http.Client{Timeout: 5 * time.Second}
	return &PoIHTTPProxyClient{host: host, port: port, baseURI: uri, client: client}
}

func withdHeaders(req *http.Request, correlation, apiKey bool, apiKeyOverride string) {
	if correlation {
		req.Header.Set("X-Correlation-Id", uuid.NewString())
	}
	if apiKey && apiKeyOverride != "" {
		req.Header.Set("X-Api-Key", apiKeyOverride)
	}
	if apiKey && apiKeyOverride == "" {
		req.Header.Set("X-Api-Key", "test")
	}

	req.Header.Set("Content-Type", "application/json")
}

func handleHTTPResponse[T any](res *http.Response) *HTTPResponse[T] {
	if res.StatusCode != http.StatusOK {
		var errRes HTTPErrorResponse
		err := json.NewDecoder(res.Body).Decode(&errRes)
		Expect(err).To(Not(HaveOccurred()))
		return &HTTPResponse[T]{Ok: nil, Err: &errRes, StatusCode: res.StatusCode}
	}
	var pois T
	err := json.NewDecoder(res.Body).Decode(&pois)
	Expect(err).To(Not(HaveOccurred()))
	return &HTTPResponse[T]{Ok: &pois, Err: nil, StatusCode: res.StatusCode}
}
