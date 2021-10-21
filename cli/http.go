package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const defaultBase = "https://svc.lab47.dev"

var baseURL = defaultBase

func currentServer() string {
	if baseURL == defaultBase {
		return "vcr.pub"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	return u.Host
}

type RemoteError struct {
	Code   int    `json:"code"`
	ErrorS string `json:"error"`
}

func (r *RemoteError) Error() string {
	return fmt.Sprintf("remote error: %s (%d)", r.ErrorS, r.Code)
}

type Status struct {
	Status string `json:"status"`
}

func perform(ctx context.Context, method, path string, hdrs http.Header, val interface{}, ret interface{}) error {
	var r io.Reader

	if val != nil {
		body, err := json.Marshal(val)
		if err != nil {
			return errors.Wrapf(err, "error marshaling request")
		}

		r = bytes.NewReader(body)
	}

	url := baseURL + path

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return err
	}

	for k, v := range hdrs {
		req.Header[k] = v
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error posting to: %s", path)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		if resp.Header.Get("Content-Type") == "application/json" {
			var er RemoteError

			err = json.NewDecoder(resp.Body).Decode(&er)
			if err != nil {
				return errors.Wrapf(err, "error decoding response")
			}

			return &er
		}

		return fmt.Errorf("Unexpected status: %d", resp.StatusCode)
	}

	if ret == nil {
		return nil
	}

	err = json.NewDecoder(resp.Body).Decode(ret)
	if err != nil {
		return errors.Wrapf(err, "error decoding response")
	}

	return nil
}

func Post(ctx context.Context, path string, req, ret interface{}) error {
	return perform(ctx, "POST", path, nil, req, ret)
}

func setAuthorization(hdrs http.Header, user, pass string) {
	hdrs.Set("Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
}

func TokenPost(ctx context.Context, token, path string, req, ret interface{}) error {
	hdrs := http.Header{}
	setAuthorization(hdrs, "cytoken", token)
	return perform(ctx, "POST", path, hdrs, req, ret)
}

func TokenPut(ctx context.Context, token, path string, req, ret interface{}) error {
	hdrs := http.Header{}
	setAuthorization(hdrs, "cytoken", token)
	return perform(ctx, "PUT", path, hdrs, req, ret)
}

func Get(ctx context.Context, path string, ret interface{}) error {
	return perform(ctx, "POST", path, nil, nil, ret)
}

func TokenGet(ctx context.Context, token, path string, ret interface{}) error {
	hdrs := http.Header{}
	setAuthorization(hdrs, "cytoken", token)
	return perform(ctx, "GET", path, hdrs, nil, ret)
}
