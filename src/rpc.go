package src

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Rpc struct {
	providers     []RpcProvider
	isWorking     []bool
	providersFile string
	lastChecked   time.Time
	modifiedTime  time.Time
}

func RpcNew(providersFile string) (Rpc, error) {
	rpc := Rpc{providersFile: providersFile}
	err := rpc.updateProviders()
	if err != nil {
		return Rpc{}, err
	}
	return rpc, nil
}

func (r *Rpc) updateProviders() error {
	if time.Since(r.lastChecked) < 10*time.Second {
		return nil
	}
	r.lastChecked = time.Now()
	fileInfo, err := os.Stat(r.providersFile)
	if err != nil {
		return err
	}
	if fileInfo.ModTime().Equal(r.modifiedTime) {
		return nil
	}
	file, err := os.Open(r.providersFile)
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
	r.modifiedTime = fileInfo.ModTime()
	r.isWorking = make([]bool, len(r.providers))
	for i := range r.isWorking {
		r.isWorking[i] = true
	}

	return nil
}

func (r *Rpc) CheckProviders() error {
	type SlotResponse struct {
		Result uint64 `json:"result"`
	}

	maxSlot := uint64(0)
	providerSlots := make([]uint64, len(r.providers))

	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	for providerIndex, provider := range r.providers {
		waitGroup.Add(1)
		go func(providerIndex int, provider RpcProvider) {
			defer waitGroup.Done()
			rpcRequest := RpcRequest{
				JsonRpc: "2.0",
				Method:  "getSlot",
				Params:  []any{},
				Id:      1,
			}
			requestBytes, err := json.Marshal(rpcRequest)
			if err != nil {
				return
			}
			request, err := http.NewRequest("POST", provider.Url, bytes.NewReader(requestBytes))
			if err != nil {
				return
			}

			if provider.HeaderKey != "" && provider.HeaderValue != "" {
				request.Header.Set(provider.HeaderKey, provider.HeaderValue)
			}
			request.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 5 * time.Second}
			response, err := client.Do(request)
			if err != nil {
				return
			}
			defer response.Body.Close()

			if response.StatusCode != 200 {
				return
			}

			var slotResponse SlotResponse
			if err := json.NewDecoder(response.Body).Decode(&slotResponse); err != nil {
				return
			}

			mutex.Lock()
			providerSlots[providerIndex] = slotResponse.Result
			if slotResponse.Result > maxSlot {
				maxSlot = slotResponse.Result
			}
			mutex.Unlock()
		}(providerIndex, provider)
	}

	waitGroup.Wait()

	for providerIndex := range r.providers {
		r.isWorking[providerIndex] = providerSlots[providerIndex] > 0 && (maxSlot-providerSlots[providerIndex]) < 120
	}

	return nil
}

func (r *Rpc) Call(requestBody []byte, ip string) (string, error) {
	err := r.updateProviders()
	if err != nil {
		return "", fmt.Errorf("failed to update providers: %w", err)
	}

	if len(r.providers) == 0 {
		return "", errors.New("no providers")
	}

	err = VerifyJson(requestBody)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	hasher.Write([]byte(ip))
	hashResult := hasher.Sum(nil)

	startIndex := binary.BigEndian.Uint64(hashResult[:8]) % uint64(len(r.providers))
	currentIndex := startIndex

	for !r.isWorking[currentIndex] {
		currentIndex = (currentIndex + 1) % uint64(len(r.providers))
		if currentIndex == startIndex {
			return "", errors.New("no working providers")
		}
	}

	selectedProvider := r.providers[currentIndex]
	request, err := http.NewRequest("POST", selectedProvider.Url, bytes.NewReader(requestBody))
	if err != nil {
		return "", err
	}

	if selectedProvider.HeaderKey != "" && selectedProvider.HeaderValue != "" {
		request.Header.Set(selectedProvider.HeaderKey, selectedProvider.HeaderValue)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New("upstream returned " + strconv.Itoa(response.StatusCode))
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	err = VerifyJson(responseBody)
	if err != nil {
		return "", err
	}

	return string(responseBody), nil
}
