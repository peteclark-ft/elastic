// Copyright 2012-2015 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	// ErrPluginNotFound is returned when using a service that requires a plugin that is not available.
	ErrPluginNotFound = errors.New("elastic: plugin not found")
)

// checkResponse will return an error if the request/response indicates
// an error returned from Elasticsearch.
//
// HTTP status codes between in the range [200..299] are considered successful.
// Also, HTTP requests with method HEAD that return 404 are considered
// to be no errors. All other request/response combinations return an error.
//
// The func tries to parse error details as returned from Elasticsearch
// and encapsulates them in type elastic.Error.
func checkResponse(req *http.Request, res *http.Response) error {
	// 200-299 are valid status codes
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}
	// HEAD requests with 404 are not an error
	if req.Method == "HEAD" && res.StatusCode == 404 {
		return nil
	}
	if res.Body == nil {
		return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
	}
	slurp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("elastic: Error %d (%s) when reading body: %v", res.StatusCode, http.StatusText(res.StatusCode), err)
	}
	errReply := new(Error)
	err = json.Unmarshal(slurp, errReply)
	if err != nil {
		return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
	}
	if errReply != nil {
		if errReply.Status == 0 {
			errReply.Status = res.StatusCode
		}
		return errReply
	}
	return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
}

// Error encapsulates error details as returned from Elasticsearch.
type Error struct {
	Status  int           `json:"status"`
	Details *ErrorDetails `json:"error,omitempty"`
}

// ErrorDetails encapsulate error details from Elasticsearch.
// It is used in e.g. elastic.Error and elastic.BulkResponseItem.
type ErrorDetails struct {
	Type      string                 `json:"type"`
	Reason    string                 `json:"reason"`
	Index     string                 `json:"index,omitempty"`
	CausedBy  map[string]interface{} `json:"caused_by,omitempty"`
	RootCause []*ErrorDetails        `json:"root_cause,omitempty"`
}

// Error returns a string representation of the error.
func (e *Error) Error() string {
	if e.Details != nil && e.Details.Reason != "" {
		return fmt.Sprintf("elastic: Error %d (%s): %s [type=%s]", e.Status, http.StatusText(e.Status), e.Details.Reason, e.Details.Type)
	} else {
		return fmt.Sprintf("elastic: Error %d (%s)", e.Status, http.StatusText(e.Status))
	}
}

// IsNotFound returns true if the given error indicates that Elasticsearch
// returned HTTP status 404. The err parameter can be of type *elastic.Error,
// elastic.Error, *http.Response or int (indicating the HTTP status code).
func IsNotFound(err interface{}) bool {
	switch e := err.(type) {
	case *http.Response:
		return e.StatusCode == http.StatusNotFound
	case *Error:
		return e.Status == http.StatusNotFound
	case Error:
		return e.Status == http.StatusNotFound
	case int:
		return e == http.StatusNotFound
	}
	return false
}