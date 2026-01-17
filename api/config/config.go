package config

import "os"

type Config struct {
	MongoURI  string
	RedisAddr string
	HTTPPort  string
	WSPort    string
}

func Load() *Config {
	return &Config{
		MongoURI:  getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		HTTPPort:  getEnv("HTTP_PORT", "8080"),
		WSPort:    getEnv("WS_PORT", "8081"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
