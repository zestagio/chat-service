package middlewares

import (
	"errors"

	"github.com/golang-jwt/jwt"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrNoAllowedResources = errors.New("no allowed resources")
	ErrSubjectNotDefined  = errors.New(`"sub" is not defined`)
)

type claims struct {
	jwt.StandardClaims
	Audience        keycloakclient.StringOrSlice `json:"aud,omitempty"`
	Subject         types.UserID                 `json:"sub,omitempty"`
	ResourcesAccess resourceAccess               `json:"resource_access"`
}

// Valid returns errors:
// - from StandardClaims validation;
// - ErrNoAllowedResources, if claims doesn't contain `resource_access` map or it's empty;
// - ErrSubjectNotDefined, if claims doesn't contain `sub` field or subject is zero UUID.
func (c claims) Valid() error {
	if err := c.StandardClaims.Valid(); err != nil {
		return err
	}

	if len(c.ResourcesAccess) == 0 {
		return ErrNoAllowedResources
	}

	if c.Subject.IsZero() {
		return ErrSubjectNotDefined
	}

	return nil
}

func (c claims) UserID() types.UserID {
	return c.Subject
}

type resourceAccess map[string]struct {
	Roles []string `json:"roles"`
}

func (ra resourceAccess) HasResourceRole(resource, role string) bool {
	access, ok := ra[resource]
	if !ok {
		return false
	}

	for _, r := range access.Roles {
		if r == role {
			return true
		}
	}
	return false
}
