package pkg

import (
	"fmt"

	"github.com/oidc-mytoken/utils/httpclient"
)

// AuthorizationChecker is an interface type that provides a way to check if the client used proper authorization
type AuthorizationChecker interface {
	CheckAuthorization(auth string) (bool, error)
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
func (c IntrospectionAuthChecker) CheckAuthorization(auth string) (bool, error) {
	httpResp, err := httpclient.Do().R().
		SetFormData(
			map[string]string{
				"token": "dummy",
			},
		).
		SetHeader("Authorization", auth).
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
