package keycloakclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type IntrospectTokenResult struct {
	Exp    int      `json:"exp"`
	Iat    int      `json:"iat"`
	Aud    []string `json:"aud"`
	Active bool     `json:"active"`
}

func (t *IntrospectTokenResult) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	obj := struct {
		Exp    int  `json:"exp"`
		Iat    int  `json:"iat"`
		Aud    any  `json:"aud"`
		Active bool `json:"active"`
	}{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	t.Exp = obj.Exp
	t.Iat = obj.Iat
	t.Active = obj.Active

	if obj.Aud == nil {
		t.Aud = nil
	} else if str, ok := obj.Aud.(string); ok {
		t.Aud = []string{str}
	} else if arr, ok := obj.Aud.([]any); ok {
		t.Aud = make([]string, 0, len(arr))
		for _, v := range arr {
			t.Aud = append(t.Aud, v.(string))
		}
	}

	return nil
}

// IntrospectToken implements
// https://www.keycloak.org/docs/latest/authorization_services/index.html#obtaining-information-about-an-rpt
func (c *Client) IntrospectToken(ctx context.Context, token string) (*IntrospectTokenResult, error) {
	url := fmt.Sprintf("realms/%s/protocol/openid-connect/token/introspect", c.realm)

	var introspectToken IntrospectTokenResult

	resp, err := c.auth(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"token_type_hint": "requesting_party_token",
			"token":           token,
		}).
		SetResult(&introspectToken).
		Post(url)
	if err != nil {
		return nil, fmt.Errorf("send request to keycloak: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("errored keycloak response: %v", resp.Status())
	}

	return &introspectToken, nil
}

func (c *Client) auth(ctx context.Context) *resty.Request {
	c.cli.SetBasicAuth(c.clientID, c.clientSecret)
	return c.cli.R().SetContext(ctx)
}
