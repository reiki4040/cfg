package main

import (
	"fmt"
	"log"

	"github.com/yourusername/cfg/pkg/cfg"
)

type Config struct {
	Database struct {
		Host     string `cfg:"database.host"`
		Password string `cfg:"database.password"`
		Port     int    `cfg:"database.port"`
		Name     string `cfg:"database.name"`
		URL      string `cfg:"database.url"`
	}
	App struct {
		Name    string `cfg:"app.name"`
		Debug   bool   `cfg:"app.debug"`
		APIURL  string `cfg:"app.api_url"`
		Timeout int    `cfg:"app.timeout"`
	}
	Redis struct {
		URL      string `cfg:"redis.url"`
		Password string `cfg:"redis.password"`
	}
}

func main() {
	var config Config

	loader := cfg.New(cfg.LoadOptions{
		Stage:     "prod",
		AWSRegion: "us-east-1",
	})

	err := loader.LoadFromFile("config.yaml", &config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("App Name: %s\n", config.App.Name)
	fmt.Printf("Database Host: %s\n", config.Database.Host)
	fmt.Printf("API URL: %s\n", config.App.APIURL)
	fmt.Printf("Debug Mode: %t\n", config.App.Debug)
	fmt.Printf("Timeout: %d\n", config.App.Timeout)
}