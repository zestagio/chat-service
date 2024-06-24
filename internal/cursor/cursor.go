package cursor

import (
	"encoding/base64"
	"encoding/json"
)

func Encode(data any) (string, error) {
	j, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(j), nil
}

func Decode(in string, to any) error {
	dec, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(dec, to)
}
