package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	ratelimiter "gihub.com/Saurav1999/sliding-window-rate-limiter/RateLimiter"
)

type Response struct {
	Message   string
	ErrorCode int
}

func hello(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	resp := Response{
		Message:   "Hello there!",
		ErrorCode: 200,
	}
	payload, err := json.Marshal(resp)

	if err != nil {
		fmt.Println("Error marshaling")
	}
	w.Write([]byte(payload))
}
func main() {
	// mux := http.NewServeMux()
	ratelimiter.Init()
	helloHandler := http.HandlerFunc(hello)
	http.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByUser, r.Header.Get("X-User-ID")).ServeHTTP(w, r)
	}))
	http.ListenAndServe(":5000", nil)

}
