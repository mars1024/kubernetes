package swarm

import (
	"net/http"
)

// Delete sends delete request to the server with custom request options.
func Delete(endpoint string, opts ...Option) (*http.Response, error) {
	fullPath := SwarmURL + endpoint
	req, err := newRequest(http.MethodDelete, fullPath, opts...)
	if err != nil {
		return nil, err
	}

	// By default, if Content-Type in header is not set, set it to application/json
	if req.Header.Get("Content-Type") == "" {
		WithHeader("Content-Type", "application/json")(req)
	}
	return httpsDo(req)
}

// Get sends get request to the server with custom request options.
func Get(endpoint string, opts ...Option) (*http.Response, error) {
	fullPath := SwarmURL + endpoint
	req, err := newRequest(http.MethodGet, fullPath, opts...)
	if err != nil {
		return nil, err
	}

	// By default, if Content-Type in header is not set, set it to application/json
	if req.Header.Get("Content-Type") == "" {
		WithHeader("Content-Type", "application/json")(req)
	}
	return httpsDo(req)
}

// Post sends post request to the server with custom request options.
func Post(endpoint string, opts ...Option) (*http.Response, error) {
	fullPath := SwarmURL + endpoint
	req, err := newRequest(http.MethodPost, fullPath, opts...)
	if err != nil {
		return nil, err
	}

	// By default, if Content-Type in header is not set, set it to application/json
	if req.Header.Get("Content-Type") == "" {
		WithHeader("Content-Type", "application/json")(req)
	}
	return httpsDo(req)
}
