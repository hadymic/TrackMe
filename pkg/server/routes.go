package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pagpeter/trackme/pkg/types"
	"github.com/pagpeter/trackme/pkg/utils"
)

// RouteResult is the HTTP payload produced by a route handler.
type RouteResult struct {
	Body        []byte
	ContentType string
	Status      int
	CORS        bool
}

// RouteHandler is the function signature for route handlers
type RouteHandler func(types.Response, url.Values) (RouteResult, error)

var (
	ErrTLSNotAvailable = errors.New("TLS details not available")
)

func staticFile(file string) RouteHandler {
	return func(types.Response, url.Values) (RouteResult, error) {
		b, err := utils.ReadFile(file)
		if err != nil {
			return RouteResult{}, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		return RouteResult{Body: b, ContentType: "text/html"}, nil
	}
}

func apiAll(res types.Response, _ url.Values) (RouteResult, error) {
	return RouteResult{Body: []byte(res.ToJson()), ContentType: "application/json"}, nil
}

func apiTLS(res types.Response, _ url.Values) (RouteResult, error) {
	return RouteResult{Body: []byte(types.Response{
		TLS: res.TLS,
	}.ToJson()), ContentType: "application/json"}, nil
}

func apiClean(res types.Response, _ url.Values) (RouteResult, error) {
	akamai := "-"
	hash := "-"
	if res.HTTPVersion == "h2" && res.Http2 != nil {
		akamai = res.Http2.AkamaiFingerprint
		hash = utils.GetMD5Hash(res.Http2.AkamaiFingerprint)
	} else if res.HTTPVersion == "h3" && res.Http3 != nil {
		akamai = res.Http3.AkamaiFingerprint
		hash = res.Http3.AkamaiFingerprintHash
	}

	smallRes := types.SmallResponse{
		Akamai:      akamai,
		AkamaiHash:  hash,
		HTTPVersion: res.HTTPVersion,
	}

	if res.TLS != nil {
		smallRes.JA3 = res.TLS.JA3
		smallRes.JA3Hash = res.TLS.JA3Hash
		smallRes.JA4 = res.TLS.JA4
		smallRes.JA4_r = res.TLS.JA4_r
		smallRes.JA4_ro = res.TLS.JA4_ro
		smallRes.PeetPrint = res.TLS.PeetPrint
		smallRes.PeetPrintHash = res.TLS.PeetPrintHash
	}

	return RouteResult{Body: []byte(smallRes.ToJson()), ContentType: "application/json"}, nil
}

func apiRecord(res types.Response, _ url.Values) (RouteResult, error) {
	akamai := "-"
	hash := "-"
	if res.HTTPVersion == "h2" && res.Http2 != nil {
		akamai = res.Http2.AkamaiFingerprint
		hash = utils.GetMD5Hash(res.Http2.AkamaiFingerprint)
	} else if res.HTTPVersion == "h3" && res.Http3 != nil {
		akamai = res.Http3.AkamaiFingerprint
		hash = res.Http3.AkamaiFingerprintHash
	}

	utc := time.Now().UTC()
	rec := types.FingerprintRecord{
		Akamai:      akamai,
		AkamaiHash:  hash,
		HTTPVersion: res.HTTPVersion,
		IP:          res.IP,
		UA:          res.UserAgent,
		EventDate:   utc.Format("20060102"),
		EventTime:   utc.Unix(),
	}

	if res.TLS != nil {
		rec.JA3 = res.TLS.JA3
		rec.JA3Hash = res.TLS.JA3Hash
		rec.JA4 = res.TLS.JA4
		rec.JA4_r = res.TLS.JA4_r
		rec.JA4_ro = res.TLS.JA4_ro
	}

	line, err := json.Marshal(rec)
	if err != nil {
		return RouteResult{}, err
	}
	line = append(line, '\n')
	if _, err := getFPLog().Write(line); err != nil {
		return RouteResult{}, err
	}
	return RouteResult{Status: http.StatusNoContent, CORS: true}, nil
}

func apiRaw(res types.Response, _ url.Values) (RouteResult, error) {
	if res.TLS == nil {
		return RouteResult{}, ErrTLSNotAvailable
	}
	return RouteResult{Body: []byte(fmt.Sprintf(`{"raw": "%s", "raw_b64": "%s"}`, res.TLS.RawBytes, res.TLS.RawB64)), ContentType: "application/json"}, nil
}

func index(r types.Response, v url.Values) (RouteResult, error) {
	rr, err := staticFile("static/index.html")(r, v)
	if err != nil {
		return RouteResult{}, err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return RouteResult{}, fmt.Errorf("failed to marshal response: %w", err)
	}
	return RouteResult{Body: []byte(strings.ReplaceAll(string(rr.Body), "/*DATA*/", string(data))), ContentType: rr.ContentType}, nil
}

func getAllPaths() map[string]RouteHandler {
	return map[string]RouteHandler{
		"/":           index,
		"/explore":    staticFile("static/explore.html"),
		"/api/all":    apiAll,
		"/api/tls":    apiTLS,
		"/api/clean":  apiClean,
		"/api/raw":    apiRaw,
		"/api/record": apiRecord,
	}
}
