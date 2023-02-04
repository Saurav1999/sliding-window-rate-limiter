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
	helloHandler := http.HandlerFunc(hello)
	http.Handle("/hello", ratelimiter.RateLimiter(helloHandler))

	http.ListenAndServe(":5000", nil)

}
