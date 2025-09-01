package main

import (
	"fmt"
	"log"

	"github.com/yourusername/cfg"
)

type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Debug   bool   `yaml:"debug"`
		APIURL  string `yaml:"api_url"`
		Timeout int    `yaml:"timeout"`
		Secret  string `yaml:"secret"`
	}
	Domain struct {
		Api string `yaml:"api"`
		Web string `yaml:"web"`
	}
}

func main() {
	var config Config

	// Use path prefix with {stage} placeholder
	loader := cfg.NewWithPrefix("/cfgtool/{stage}", cfg.LoadOptions{
		Stage:             "dev",
		AWSRegion:         "ap-northeast-1",
		StagePrefixBlanks: []string{"prod"},
	})

	err := loader.LoadFromFile("config-relative.yaml", &config)
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