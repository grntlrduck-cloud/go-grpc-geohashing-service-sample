package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const (
	basePath      = "api/v1/pois"
	proximityPath = "prxoimity"
	bboxPath      = "bbox"
	routePath     = "route"
	infoPath      = "info" // the only endpoint where we have a path variable
)

type HttpResponse[T any] struct {
	Ok         *T
	Err        *HttpErrorResponse
	StatusCode int
}

type PoIsHttpResponse struct {
	Items []PoIHttp `json:"items"`
}

type HttpErrorResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type PoIHttpResponse struct {
	Poi PoIHttp `json:"poi"`
}

type PoIHttp struct {
	Id         string          `json:"id"`
	Coordinate CoordinatesHttp `json:"coordinate"`
	Entrance   CoordinatesHttp `json:"entrance"`
	Address    AddressHttp     `json:"address"`
	Features   []string        `json:"features"`
}

type CoordinatesHttp struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type AddressHttp struct {
	Street       string `json:"street"`
	StreetNumber string `json:"street_number"`
	City         string `json:"city"`
	ZipCode      string `json:"zip_code"`
	Country      string `json:"country"`
}

type PoIHttpProxyClient struct {
	host    string
	port    int32
	baseUri string
	client  http.Client
}

func (p *PoIHttpProxyClient) Info(
	id string,
	correlation bool,
) *HttpResponse[PoIHttpResponse] {
	url := fmt.Sprintf("%s/%s/%s", p.baseUri, infoPath, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHttpResponse[PoIHttpResponse](res)
}

func (p *PoIHttpProxyClient) Bbox(
	ne, sw CoordinatesHttp,
	correlation bool,
) *HttpResponse[PoIsHttpResponse] {
	url := fmt.Sprintf(
		"%s/%s?bbox.ne.lat=%f&bbox.ne.lon=%f&bbox.sw.lat=%f&bbox.sw.lon=%f",
		p.baseUri,
		bboxPath,
		ne.Lat,
		ne.Lon,
		sw.Lat,
		sw.Lon,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHttpResponse[PoIsHttpResponse](res)
}

func (p *PoIHttpProxyClient) Prxoimity(
	center CoordinatesHttp,
	radiusMeters float64,
	correlation bool,
) *HttpResponse[PoIsHttpResponse] {
	url := fmt.Sprintf(
		"%s/%s?center.lat=%f&center.lon=%f&radius_meters=%f",
		p.baseUri,
		proximityPath,
		center.Lat,
		center.Lon,
		radiusMeters,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHttpResponse[PoIsHttpResponse](res)
}

func (p *PoIHttpProxyClient) Route(
	route []CoordinatesHttp,
	correlation bool,
) *HttpResponse[PoIsHttpResponse] {
	url := fmt.Sprintf("%s/%s", p.baseUri, routePath)
	encRoute, err := json.Marshal(route)
	Expect(err).To(Not(HaveOccurred()))
	reader := bytes.NewReader(encRoute)
	req, err := http.NewRequest(http.MethodGet, url, reader)
	Expect(err).To(Not(HaveOccurred()))

	withdHeaders(req, correlation)
	res, err := p.client.Do(req)
	Expect(err).To(Not(HaveOccurred()))
	defer res.Body.Close()

	return handleHttpResponse[PoIsHttpResponse](res)
}

func NewPoIHttpProxyClient(host string, port int32) *PoIHttpProxyClient {
	uri := fmt.Sprintf("http://%s:%d/%s", host, port, basePath)
	client := http.Client{Timeout: 5 * time.Second}
	return &PoIHttpProxyClient{host: host, port: port, baseUri: uri, client: client}
}

func withdHeaders(req *http.Request, correlation bool) {
	if correlation {
		req.Header.Set("X-Correlation-Id", uuid.NewString())
	}
	req.Header.Set("Content-Type", "application/json")
}

func handleHttpResponse[T any](res *http.Response) *HttpResponse[T] {
	if res.StatusCode != http.StatusOK {
		var errRes HttpErrorResponse
		err := json.NewDecoder(res.Body).Decode(&errRes)
		Expect(err).To(Not(HaveOccurred()))
		return &HttpResponse[T]{Ok: nil, Err: &errRes, StatusCode: res.StatusCode}
	}
	var pois T
	err := json.NewDecoder(res.Body).Decode(&pois)
	Expect(err).To(Not(HaveOccurred()))
	return &HttpResponse[T]{Ok: &pois, Err: nil, StatusCode: res.StatusCode}
}
