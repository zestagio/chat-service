package cursor

import (
	"encoding/base64"
	"encoding/json"
)

var encoding = base64.URLEncoding

func Encode(data any) (string, error) {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return encoding.EncodeToString(marshalled), nil
}

func Decode(in string, to any) error {
	decoded, err := encoding.DecodeString(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, to)
}
