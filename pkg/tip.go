package pkg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/structs"
	"github.com/google/go-querystring/query"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/oidc-mytoken/utils/httpclient"
	"github.com/oidc-mytoken/utils/unixtime"
	"github.com/oidc-mytoken/utils/utils/issuerutils"
	"github.com/pkg/errors"
	"github.com/zachmann/go-oidfed/pkg"
)

func NewTokenProxy(conf TIPConfig) *TIP {
	for i, issuer := range conf.RemoteIssuers {
		issuer.discoverEndpoint()
		conf.RemoteIssuers[i] = issuer
	}
	return &TIP{conf: conf}
}

type TIP struct {
	conf TIPConfig
}

type TokenIntrospectionRequest struct {
	Token         string `json:"token" form:"token" query:"token" url:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty" form:"token_type_hint,omitempty" query:"token,omitempty" url:"token,omitempty"`
	Authorization string `json:"-" form:"-" query:"-" url:"-"`
}

type TokenIntrospectionResponse struct {
	Active     bool                           `json:"active"`
	Scope      string                         `json:"scope,omitempty"`
	ClientID   string                         `json:"client_id,omitempty"`
	Username   string                         `json:"username,omitempty"`
	TokenType  string                         `json:"token_type,omitempty"`
	Expiration unixtime.UnixTime              `json:"exp,omitempty"`
	IssuedAt   unixtime.UnixTime              `json:"iat,omitempty"`
	NotBefore  unixtime.UnixTime              `json:"nbf,omitempty"`
	Subject    string                         `json:"sub,omitempty"`
	Audience   pkg.SliceOrSingleValue[string] `json:"aud,omitempty"`
	Issuer     string                         `json:"iss,omitempty"`
	JTI        string                         `json:"jti,omitempty"`
	Extra      map[string]any                 `json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
// It also marshals extra fields.
func (r TokenIntrospectionResponse) MarshalJSON() ([]byte, error) {
	type Alias TokenIntrospectionResponse
	explicitFields, err := json.Marshal(Alias(r))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return extraMarshalHelper(explicitFields, r.Extra)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It also unmarshalls additional fields into the Extra claim.
func (r *TokenIntrospectionResponse) UnmarshalJSON(data []byte) error {
	type Alias TokenIntrospectionResponse
	rr := Alias(*r)
	extra, err := unmarshalWithExtra(data, &rr)
	if err != nil {
		return err
	}
	rr.Extra = extra
	*r = TokenIntrospectionResponse(rr)
	return nil
}

func extraMarshalHelper(explicitFields []byte, extra map[string]interface{}) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(explicitFields, &m); err != nil {
		return nil, err
	}
	for k, v := range extra {
		e, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = e
	}
	data, err := json.Marshal(m)
	return data, errors.WithStack(err)
}

func unmarshalWithExtra(data []byte, target interface{}) (map[string]interface{}, error) {
	if err := json.Unmarshal(data, target); err != nil {
		return nil, errors.WithStack(err)
	}
	extra := make(map[string]interface{})
	if err := json.Unmarshal(data, &extra); err != nil {
		return nil, errors.WithStack(err)
	}
	s := structs.New(target)
	for _, tag := range fieldTagNames(s.Fields(), "json") {
		delete(extra, tag)
	}
	if len(extra) == 0 {
		extra = nil
	}
	return extra, nil
}

// fieldTagNames returns a slice of the tag names for a []*structs.Field and the given tag
func fieldTagNames(fields []*structs.Field, tag string) (names []string) {
	for _, f := range fields {
		if f == nil {
			continue
		}
		t := f.Tag(tag)
		if i := strings.IndexRune(t, ','); i > 0 {
			t = t[:i]
		}
		if t != "" && t != "-" {
			names = append(names, t)
		}
	}
	return
}

func (t TIP) Introspect(req TokenIntrospectionRequest) (*TokenIntrospectionResponse, error) {
	token := req.Token
	if token == "" {
		return nil, invalidRequestError("required parameter 'token' not given")
	}
	m, err := jws.Parse([]byte(token))
	if err != nil {
		return t.fallbackIntrospection(req)
	}
	var claims map[string]any
	if err = json.Unmarshal(m.Payload(), &claims); err != nil {
		return t.fallbackIntrospection(req)
	}
	iss, ok := claims["iss"]
	if !ok {
		return t.fallbackIntrospection(req)
	}
	issuer, ok := iss.(string)
	if !ok {
		return t.fallbackIntrospection(req)
	}
	if issuerutils.CompareIssuerURLs(issuer, t.conf.LinkedIssuer.IssuerURL) {
		return t.linkedIntrospection(req)
	}
	return t.remoteIntrospection(issuer, req)
}

func (t TIP) findRemoteIssuer(iss string) *remoteIssuerConf {
	for _, c := range t.conf.RemoteIssuers {
		if issuerutils.CompareIssuerURLs(c.IssuerURL, iss) {
			return &c
		}
	}
	return nil
}

func (t TIP) remoteIntrospection(iss string, req TokenIntrospectionRequest) (*TokenIntrospectionResponse, error) {
	params, err := query.Values(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	conf := t.findRemoteIssuer(iss)
	if conf == nil {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	httpResp, err := httpclient.Do().R().
		SetFormDataFromValues(params).
		SetBasicAuth(conf.ClientID, conf.ClientSecret).
		SetResult(&TokenIntrospectionResponse{}).
		Post(conf.IntrospectionEndpoint)
	if err != nil {
		return nil, internalServerError(fmt.Sprintf("failed to introspect remote issuer: %s", err))
	}
	if httpResp.StatusCode() != 200 {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	resp, ok := httpResp.Result().(*TokenIntrospectionResponse)
	if !ok {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	finalResponse := &TokenIntrospectionResponse{
		Active:     resp.Active,
		ClientID:   resp.ClientID,
		TokenType:  resp.TokenType,
		Expiration: resp.Expiration,
		IssuedAt:   resp.IssuedAt,
		NotBefore:  resp.NotBefore,
		Issuer:     resp.Issuer,
		JTI:        resp.JTI,
		Audience:   conf.ClaimMapping.StringSlices.translate("aud", resp.Audience),
		Username:   conf.ClaimMapping.Strings.translate("username", resp.Username),
		Scope: strings.Join(
			conf.ClaimMapping.StringSlices.translate("scope", strings.Split(resp.Scope, " ")), " ",
		),
		Subject: conf.ClaimMapping.Strings.translate("sub", resp.Subject),
	}
	if l := len(resp.Extra); l > 0 {
		finalResponse.Extra = make(map[string]any, l)
	}
	for k, v := range resp.Extra {
		newK, ok := conf.ClaimRenaming[k]
		if !ok {
			newK = k
		}
		switch typedValue := v.(type) {
		case string:
			finalResponse.Extra[newK] = conf.ClaimMapping.Strings.translate(k, typedValue)
		case []string:
			finalResponse.Extra[newK] = conf.ClaimMapping.StringSlices.
				translate(k, typedValue)
		case []any:
			if len(typedValue) == 0 {
				finalResponse.Extra[newK] = v
			}
			if _, ok := typedValue[0].(string); !ok {
				finalResponse.Extra[newK] = v
			}
			var asStrings []string
			for _, vv := range typedValue {
				asStrings = append(asStrings, vv.(string))
			}
			finalResponse.Extra[newK] = conf.ClaimMapping.StringSlices.translate(k, asStrings)
		default:
			finalResponse.Extra[newK] = v
		}
	}
	return finalResponse, nil
}

func (t TIP) fallbackIntrospection(req TokenIntrospectionRequest) (*TokenIntrospectionResponse, error) {
	if t.conf.FallbackIssuer.IssuerURL == "" {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	return t.remoteIntrospection(t.conf.FallbackIssuer.IssuerURL, req)
}
func (t TIP) linkedIntrospection(req TokenIntrospectionRequest) (*TokenIntrospectionResponse, error) {
	params, err := query.Values(req)
	if err != nil {
		return nil, invalidRequestError("required parameter 'token' not given")
	}
	httpResp, err := httpclient.Do().R().
		SetFormDataFromValues(params).
		SetHeader("Authorization", req.Authorization).
		SetResult(&TokenIntrospectionResponse{}).
		Post(t.conf.LinkedIssuer.NativeIntrospectionEndpoint)
	if err != nil {
		return nil, internalServerError(fmt.Sprintf("failed to introspect local issuer: %s", err))
	}
	if httpResp.StatusCode() != 200 {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	resp, ok := httpResp.Result().(*TokenIntrospectionResponse)
	if !ok {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	return resp, nil
}
