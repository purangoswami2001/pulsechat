package main

import (
	"context"
	"log"

	"github.com/pulsechat/backend/internal/app"
)

func main() {
	ctx := context.Background()
	deps, err := app.NewDependencies(ctx)
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}
	defer deps.Close()

	application := app.NewApp(deps)
	if err := application.Start(); err != nil {
		log.Fatalf("failed to start application: %v", err)
	}
}
