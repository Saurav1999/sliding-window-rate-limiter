package ratelimiter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"gihub.com/Saurav1999/sliding-window-rate-limiter/RateLimiter/workers"
	"github.com/go-redis/redis/v8"
)

var REDIS_KEY string = "config"

type Response struct {
	Message   string
	ErrorCode int
}

var store = make(map[string]int)

func SlidingWindowRateLimiter(identifier string, intervalInSeconds int64, maximumRequests int) bool {
	now := time.Now().Unix()

	currentWindow := strconv.FormatInt(now/intervalInSeconds, 10)
	key := identifier + ":" + currentWindow // identifier + current time window
	requestCountCurrentWindow := store[key]
	if requestCountCurrentWindow >= maximumRequests {
		// drop request
		return false
	}
	lastWindow := strconv.FormatInt(((now - intervalInSeconds) / intervalInSeconds), 10)
	key = identifier + ":" + lastWindow // user userID + last time window
	requestCountLastWindow := store[key]
	elapsedTimePercentage := float64(now%intervalInSeconds) / float64(intervalInSeconds)

	// last window weighted count + current window count
	if (float64(requestCountLastWindow)*(1-elapsedTimePercentage))+float64(requestCountCurrentWindow) >= float64(maximumRequests) {
		// drop request
		return false
	}

	store[identifier+":"+currentWindow] = requestCountCurrentWindow + 1

	return true
}

func RateLimiter(h http.Handler) http.Handler {

	//creating redis client
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0, // Redis uses database 0, but you can choose a different database by specifying a different index. In the example you provided, the "DB" option is set to 0, indicating that database 0 will be used
	})

	go workers.LoadConfig(client, REDIS_KEY, "./RateLimiter/config/config.json")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !SlidingWindowRateLimiter("user1", 60, 5) {
			log.Println("limiting")
			resp := Response{
				Message:   "Too many requests",
				ErrorCode: http.StatusTooManyRequests,
			}
			payload, err := json.Marshal(resp)

			if err != nil {
				fmt.Println("Error marshaling")
			}
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(payload))
			return
		}
		log.Println("Not limiting")
		h.ServeHTTP(w, r) // call original
		log.Println("After")
	})
}
