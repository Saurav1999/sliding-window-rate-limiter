package ratelimiter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"gihub.com/Saurav1999/sliding-window-rate-limiter/RateLimiter/workers"
	"github.com/go-redis/redis/v8"
)

var REDIS_CONFIG_KEY string = "config"
var redisClient *redis.Client

type Response struct {
	Message   string
	ErrorCode int
}

const (
	LimitByApi = iota
	LimitByIp
	LimitByUser
)

func Init() {
	//creating redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0, // Redis uses database 0, but you can choose a different database by specifying a different index. In the example you provided, the "DB" option is set to 0, indicating that database 0 will be used
	})

	waitConfigLoad := make(chan bool)
	go workers.LoadConfig(redisClient, REDIS_CONFIG_KEY, "./RateLimiter/config/config.json", waitConfigLoad)
	log.Println("Waiting for initial config load")
	<-waitConfigLoad
	log.Println("config loaded")

}
func SlidingWindowRateLimiter(redisClient *redis.Client, r *http.Request, limitType int, key string) bool {
	var identifier string
	var intervalInSeconds int64
	var maximumRequests int64

	luaScriptToGetConfig := `
	local config = redis.call("GET", KEYS[1])
	return config`

	configJSON, err := redisClient.Eval(context.Background(), luaScriptToGetConfig, []string{"config"}).Result()

	if err != nil {
		log.Println("Error loading config from Redis:", err)
		return false
	}

	configJSONString, ok := configJSON.(string)
	if !ok {
		log.Println("Error: config is not a string")
		return false
	}

	var config workers.Config
	err = json.Unmarshal([]byte(configJSONString), &config)

	if err != nil {
		log.Println("Error parsing config JSON:", err)
		return false
	}

	switch limitType {
	case LimitByApi:
		if key != "" {
			host := r.Host
			endpoint := r.URL.Path
			URL := host + endpoint
			identifier = URL
		} else {
			identifier = key
		}

		index := -1
		for i, v := range config.LimitApis {
			if v.Name == identifier {
				index = i
				break
			}
		}

		if index == -1 {
			log.Println("Config for the identifier to limit the respective api is not found in the config:")
			return false
		}

		intervalInSeconds = int64(config.LimitApis[index].Window)
		maximumRequests = int64(config.LimitApis[index].Limit)

	case LimitByIp:
		if key != "" {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// Handle the error
				return false
			}

			identifier = ip
		} else {
			identifier = key
		}

		intervalInSeconds = int64(config.LimitIPs.Window)
		maximumRequests = int64(config.LimitIPs.Limit)
	case LimitByUser:
		identifier = key
		intervalInSeconds = int64(config.LimitUsers.Window)
		maximumRequests = int64(config.LimitUsers.Limit)
	default:
		log.Println("No limit specified")
		return false
	}

	now := time.Now().Unix()

	currentWindow := strconv.FormatInt(now/intervalInSeconds, 10)
	previousWindow := strconv.FormatInt(((now - intervalInSeconds) / intervalInSeconds), 10)
	windowBeforePrevious := strconv.FormatInt(((now - 2*intervalInSeconds) / intervalInSeconds), 10)

	luaScript := `
    local beforePrevTimestamp = ARGV[1]
    local prevTimestamp = ARGV[2]
	local currTimestamp = ARGV[3]
    redis.call("ZREMRANGEBYSCORE", KEYS[1], 0, beforePrevTimestamp)
	local prevCount = redis.call("ZCOUNT", KEYS[1], prevTimestamp, prevTimestamp)
    local currCount = redis.call("ZCOUNT", KEYS[1], currTimestamp, currTimestamp)

    return { prevCount, currCount }
	`

	result, err := redisClient.Eval(context.Background(), luaScript, []string{identifier}, windowBeforePrevious, previousWindow, currentWindow, intervalInSeconds).Result()

	if err != nil {
		log.Println("Error in executing lua script:", err)
		return false
	}

	returnArray, ok := result.([]interface{})
	if !ok {
		log.Println("Failed to cast lua result to []interface{}")
		return false
	}
	prevCount, currCount := returnArray[0].(int64), returnArray[1].(int64)
	log.Println("Lua script executed", prevCount, currCount)
	requestCountCurrentWindow := currCount
	if requestCountCurrentWindow >= maximumRequests {
		// drop request
		return false
	}

	requestCountLastWindow := prevCount
	elapsedTimePercentage := float64(now%intervalInSeconds) / float64(intervalInSeconds)

	// last window weighted count + current window count
	if (float64(requestCountLastWindow)*(1-elapsedTimePercentage))+float64(requestCountCurrentWindow) >= float64(maximumRequests) {
		// drop request
		return false
	}

	//run script to add this element to currentwindow count
	script := `
	return redis.call("ZADD", KEYS[1], ARGV[1], ARGV[2])
	`

	_, err = redisClient.Eval(context.Background(), script, []string{identifier}, currentWindow, now).Result()
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func RateLimiter(h http.Handler, limitType int, identifier string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !SlidingWindowRateLimiter(redisClient, r, limitType, identifier) {
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
	})
}
