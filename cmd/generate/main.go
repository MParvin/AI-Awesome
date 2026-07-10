package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/mparvin/awesome-stars/internal/github"
	"github.com/mparvin/awesome-stars/internal/render"
)

func main() {
	output := flag.String("output", "README.md", "path to write the generated README")
	configPath := flag.String("config", "config.yaml", "path to project config YAML")
	envFile := flag.String("env", ".env", "path to .env file (optional; ignored if missing)")
	flag.Parse()

	if err := godotenv.Load(*envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load %s: %v", *envFile, err)
	}

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		log.Fatal("GH_TOKEN is required (set it in .env or the environment)")
	}

	cfg, err := render.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client := github.NewClient(token)
	lists, err := client.FetchLists(ctx)
	if err != nil {
		log.Fatalf("fetch lists: %v", err)
	}

	readme := render.RenderREADME(lists, cfg, time.Now().UTC())
	if err := os.WriteFile(*output, []byte(readme), 0o644); err != nil {
		log.Fatalf("write readme: %v", err)
	}

	filtered := render.FilterLists(lists, cfg)
	fmt.Printf("Wrote %s (%d lists)\n", *output, len(filtered))
}
