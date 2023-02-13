package workers

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type LimitItems struct {
	Name   string `json:"name"`
	Limit  int    `json:"limit"`
	Window int    `json:"window"`
	Unit   string `json:"unit"`
}

type Config struct {
	LimitApis  []LimitItems `json:"limitsAPI"`
	LimitUsers LimitItems   `json:"limitsUser"`
	LimitIPs   LimitItems   `json:"limitByIp"`
}

func readAndWriteConfig(file *os.File, redisKey string, redisClient *redis.Client) {
	decoder := json.NewDecoder(file)
	file.Seek(0, 0) // reset pointer to the beginning
	config := &Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return
	}

	writeToRedis(redisClient, config, redisKey)
}
func writeToRedis(client *redis.Client, config *Config, redisKey string) {
	log.Println("key value", redisKey, config)
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Fatal("Error Marshaling config data:", err)
		return
	}
	err = client.Set(context.Background(), redisKey, configBytes, 0).Err()
	if err != nil {
		log.Fatal("Error writing to Redis:", err)
		return
	}
	log.Println("Wrote to Redis:", redisKey, config)
}

func LoadConfig(redisClient *redis.Client, redisKey string, filepath string, waitConfigLoad chan bool) {
	log.Println("Loading Config...")
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	readAndWriteConfig(file, redisKey, redisClient)
	info, err := file.Stat()

	if err != nil {
		log.Fatal(err)
	}

	// Done with initial config load
	waitConfigLoad <- true
	modTime := info.ModTime()
	for {
		info, err = file.Stat()
		if err != nil {
			log.Println("Config update failed", err)
		}

		if modTime != info.ModTime() {
			log.Println("Modified file:", file.Name())
			modTime = info.ModTime()
			// update the cache in Redis with the updated configuration from config file
			readAndWriteConfig(file, redisKey, redisClient)

		}
		time.Sleep(time.Second)
	}
}
