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
	ratelimiter.Init("./config/config.json", true)
	helloHandler := http.HandlerFunc(hello)
	helloHandlerWrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByUser, r.Header.Get("X-User-ID")).ServeHTTP(w, r)
	})
	http.Handle("/hello", helloHandlerWrapper)
	http.Handle("/hello-api", ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByApi, ""))
	http.Handle("/hello-ip", ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByIp, ""))

	http.ListenAndServe(":5000", nil)

}
