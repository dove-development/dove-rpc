package src

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	HOST            = "127.0.0.1"
	PORT            = 22163
	RPC_PROVIDERS   = "./priv/providers.json"
	RL_MAX_REQUESTS = 60
	RL_WINDOW_SECS  = 60
	ALLOWED_ORIGIN  = "dove.money"
)

func sendJson(w http.ResponseWriter, v any) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func onRpc(rl *Ratelimit, rpc *Rpc, w http.ResponseWriter, r *http.Request) {
	if !rl.Allow(r) {
		sendJson(w, ErrorResponse{
			Error: "Too many requests",
		})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJson(w, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		res, err := rpc.Call(body)
		if err == nil {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, res)
			return
		}
		lastErr = err
		if i < 2 {
			time.Sleep(time.Duration(250+RandomU64()%750) * time.Millisecond)
		}
	}

	sendJson(w, ErrorResponse{
		Error: lastErr.Error(),
	})
}

func App() {
	rpc, err := RpcNew(RPC_PROVIDERS)
	if err != nil {
		fmt.Println(err)
		return
	}
	rl := RatelimitNew(RL_MAX_REQUESTS, RL_WINDOW_SECS)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if ALLOWED_ORIGIN != "" {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			originUrl, err := url.Parse(origin)
			if err != nil || originUrl.Host != ALLOWED_ORIGIN {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				sendJson(w, ErrorResponse{
					Error: "Invalid origin",
				})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "https://"+ALLOWED_ORIGIN)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			sendJson(w, ErrorResponse{
				Error: "Method not allowed",
			})
			return
		}

		onRpc(&rl, &rpc, w, r)
	})

	addr := HOST + ":" + strconv.Itoa(PORT)
	log.Println("Listening on http://" + addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
	}
}
