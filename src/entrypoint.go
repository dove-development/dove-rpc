package src

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	HOST             = "127.0.0.1"
	PORT             = 22163
	RPC_PROVIDERS    = "./priv/providers.json"
	RL_MAX_REQUESTS  = 200
	RL_WINDOW_SECS   = 60
	ALLOWED_ORIGIN   = "dove.money"
	RPC_MAX_ATTEMPTS = 3
)

func sendJson(w http.ResponseWriter, v any) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func Entrypoint() {
	rpc, err := RpcNew(RPC_PROVIDERS)
	if err != nil {
		fmt.Println(err)
		return
	}
	rl := RatelimitNew(RL_MAX_REQUESTS, RL_WINDOW_SECS)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendJson(w, ErrorResponseNew(err.Error(), ""))
			return
		}

		var id any = ""
		var req RpcRequest
		err = json.Unmarshal(body, &req)
		if err == nil {
			id = req.Id
		}

		if ALLOWED_ORIGIN != "" {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			originUrl, err := url.Parse(origin)
			if err != nil || originUrl.Host != ALLOWED_ORIGIN {
				sendJson(w, ErrorResponseNew("Invalid origin", id))
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "https://"+ALLOWED_ORIGIN)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, solana-client")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, solana-client")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			sendJson(w, ErrorResponseNew("Method not allowed", id))
			return
		}

		ip := r.Header.Get("CF-Connecting-IP")
		if ip == "" {
			ip = r.Header.Get("X-Forwarded-For")
		}
		if ip == "" {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr)
		}

		if !rl.Allow(ip) {
			sendJson(w, ErrorResponseNew("Too many requests", id))
			return
		}

		var res string
		attempt := 1

		for {
			res, err = rpc.Call(body, ip)
			if err == nil {
				break
			}

			if attempt >= RPC_MAX_ATTEMPTS {
				sendJson(w, ErrorResponseNew(err.Error(), id))
				return
			}

			attempt++

			// Wait briefly before retrying
			time.Sleep(100 * time.Millisecond)
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, res)
	})

	addr := HOST + ":" + strconv.Itoa(PORT)
	log.Println("Listening on http://" + addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
	}
}
