package middlewares

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt"

	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrNoAllowedResources = errors.New("no allowed resources")
	ErrSubjectNotDefined  = errors.New(`"sub" is not defined`)
)

type claims struct {
	jwt.StandardClaims
	Audience        multiString               `json:"aud,omitempty"`
	Subject         types.UserID              `json:"sub"`
	ResourcesAccess map[string]ResourceAccess `json:"resource_access"`
}

type ResourceAccess struct {
	Roles []string `json:"roles"`
}

type multiString string

func (ms *multiString) UnmarshalJSON(data []byte) error {
	if len(data) > 0 {
		switch data[0] {
		case '"':
			var s string
			if err := json.Unmarshal(data, &s); err != nil {
				return err
			}
			*ms = multiString(s)
		case '[':
			var s []string
			if err := json.Unmarshal(data, &s); err != nil {
				return err
			}
			*ms = multiString(strings.Join(s, ","))
		}
	}
	return nil
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
