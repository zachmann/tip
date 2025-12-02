package pkg

import (
	"fmt"
	"net/url"

	"github.com/oidc-mytoken/utils/httpclient"
)

// AuthorizationChecker is an interface type that provides a way to check if the client used proper authorization
type AuthorizationChecker interface {
	CheckAuthorization(req TokenIntrospectionRequest) (bool, error)
}

// NewIntrospectionAuthChecker creates a new IntrospectionAuthChecker with the passed introspectionEndpoint
func NewIntrospectionAuthChecker(introspectionEndpoint string) IntrospectionAuthChecker {
	return IntrospectionAuthChecker{endpoint: introspectionEndpoint}
}

// IntrospectionAuthChecker is an AuthorizationChecker that uses the linked native introspection endpoint to send a
// dummy token introspection request with the same authorization
type IntrospectionAuthChecker struct {
	endpoint string
}

// CheckAuthorization implements the AuthorizationChecker interface
func (c IntrospectionAuthChecker) CheckAuthorization(req TokenIntrospectionRequest) (bool, error) {
	values, err := url.ParseQuery(string(req.Body))
	if err != nil {
		return false, internalServerError("could not parse request body for auth check")
	}
	values.Set("token", "dummy")
	request := httpclient.Do().R().
		SetFormDataFromValues(values)
	if req.Authorization != "" {
		request.SetHeader("Authorization", req.Authorization)
	}
	httpResp, err := request.
		SetResult(&TokenIntrospectionResponse{}).
		Post(c.endpoint)
	if err != nil {
		return false, internalServerError(fmt.Sprintf("failed to check auth at local issuer: %s", err))
	}
	code := httpResp.StatusCode()
	if code >= 200 && code < 300 {
		return true, nil
	}
	return false, nil
}
