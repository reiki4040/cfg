package main

import (
	"fmt"
	"log"

	"github.com/reiki4040/cfg"
)

type Config struct {
	App struct {
		Name    string `yaml:"app.name"`
		Debug   bool   `yaml:"app.debug"`
		APIURL  string `yaml:"app.api_url"`
		Timeout int    `yaml:"app.timeout"`
		Secret  string `yaml:"app.secret"`
	}
	Domain struct {
		Api string `yaml:"domain.api"`
		Web string `yaml:"domain.web"`
	}
}

func main() {
	var config Config

	loader := cfg.New(cfg.LoadOptions{
		Stage:     "dev",
		AWSRegion: "ap-northeast-1",
		StagePrefixBlanks: []string{"prod"},
	})

	err := loader.LoadFromFile("config.yaml", &config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("App Name: %s\n", config.App.Name)
	fmt.Printf("API URL: %s\n", config.App.APIURL)
	fmt.Printf("Debug Mode: %t\n", config.App.Debug)
	fmt.Printf("Timeout: %d\n", config.App.Timeout)
	fmt.Printf("Secret: %s\n", config.App.Secret)

	fmt.Printf("API domain: %s\n", config.Domain.Api)
	fmt.Printf("Web domain: %s\n", config.Domain.Web)
}
