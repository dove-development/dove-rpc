package src

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"
)

type Rpc struct {
	providers      []RpcProvider
	providers_file string
	last_checked   time.Time
	mod_time       time.Time
}

func RpcNew(providers_file string) (Rpc, error) {
	rpc := Rpc{providers_file: providers_file}
	err := rpc.updateProviders()
	if err != nil {
		return Rpc{}, err
	}
	return rpc, nil
}

func (r *Rpc) updateProviders() error {
	if time.Since(r.last_checked) < 10*time.Second {
		return nil
	}
	r.last_checked = time.Now()
	file_info, err := os.Stat(r.providers_file)
	if err != nil {
		return err
	}
	if file_info.ModTime().Equal(r.mod_time) {
		return nil
	}
	file, err := os.Open(r.providers_file)
	if err != nil {
		return err
	}
	defer file.Close()

	providers := []RpcProvider{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&providers)
	if err != nil {
		return err
	}
	r.providers = providers
	r.mod_time = file_info.ModTime()
	return nil
}

func RandomU64() uint64 {
	var provider_index [8]byte
	_, err := rand.Read(provider_index[:])
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(provider_index[:])
}

func (r *Rpc) Call(request []byte) (string, error) {
	err := r.updateProviders()
	if err != nil {
		return "", err
	}

	if len(r.providers) == 0 {
		return "", errors.New("no providers")
	}

	var req_json any
	if err := json.Unmarshal(request, &req_json); err != nil {
		return "", err
	}

	json_str, err := json.Marshal(req_json)
	if err != nil {
		return "", err
	}

	provider := r.providers[RandomU64()%uint64(len(r.providers))]
	req, err := http.NewRequest("POST", provider.Url, bytes.NewBuffer(json_str))
	if err != nil {
		return "", err
	}

	if provider.HeaderKey != "" && provider.HeaderValue != "" {
		req.Header.Set(provider.HeaderKey, provider.HeaderValue)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var resp_json map[string]any
	if err := json.Unmarshal(body, &resp_json); err != nil {
		return "", err
	}

	resp_str, err := json.Marshal(resp_json)
	if err != nil {
		return "", err
	}

	return string(resp_str), nil
}
