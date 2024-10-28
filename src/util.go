package src

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
)

func RandomU64() uint64 {
	var provider_index [8]byte
	_, err := rand.Read(provider_index[:])
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(provider_index[:])
}

func VerifyJson(b []byte) error {
	var j any
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	return nil
}
