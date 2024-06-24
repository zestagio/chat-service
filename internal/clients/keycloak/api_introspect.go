package keycloakclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"

	"github.com/zestagio/chat-service/internal/buildinfo"
)

type IntrospectTokenResult struct {
	Exp    int           `json:"exp"`
	Iat    int           `json:"iat"`
	Aud    StringOrSlice `json:"aud"`
	Active bool          `json:"active"`
}

// IntrospectToken implements
// https://www.keycloak.org/docs/latest/authorization_services/index.html#obtaining-information-about-an-rpt
func (c *Client) IntrospectToken(ctx context.Context, token string) (*IntrospectTokenResult, error) {
	url := fmt.Sprintf("realms/%s/protocol/openid-connect/token/introspect", c.realm)

	var result IntrospectTokenResult

	resp, err := c.auth(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"token_type_hint": "requesting_party_token",
			"token":           token,
		}).
		SetResult(&result).
		Post(url)
	if err != nil {
		return nil, fmt.Errorf("send request to keycloak: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("errored keycloak response: %v", resp.Status())
	}

	return &result, nil
}

func (c *Client) auth(ctx context.Context) *resty.Request {
	authStr := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	return c.cli.R().
		SetContext(ctx).
		// Another way:
		// SetAuthScheme("Basic").SetAuthToken(authStr)
		SetHeader("Authorization", "Basic "+authStr).
		SetHeader("User-Agent", "chat-service/"+buildinfo.Version())
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	if len(data) > 1 && data[0] == '[' {
		return json.Unmarshal(data, (*[]string)(s))
	}

	str, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	*s = []string{str}
	return nil
}
