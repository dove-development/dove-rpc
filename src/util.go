package src

import (
	"encoding/json"
)

func VerifyJson(b []byte) error {
	var j any
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	return nil
}
