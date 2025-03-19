package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"polvo/compose"
	plogger "polvo/logger"
	"polvo/service"
	"syscall"
)

func main() {
	var (
		loger    plogger.PolvoLogger
		composer compose.ComposeFile
		svc      service.Service
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error while get working directory %v", err)
		os.Exit(1)
	}
	loger = plogger.NewLogger(pwd)
	composer, err = compose.NewComposeFile(filepath.Join(pwd, os.Args[1]))
	if err != nil {
		loger.Close()
		panic(err)
	}
	svc, err = service.NewService(composer.GetCompose(), loger)
	if err != nil {
		loger.Close()
		panic(err)
	}
	svc.Start()

	go func() {
		<-ctx.Done()
		svc.Stop()
		fmt.Println("Shutting down...")
	}()
	svc.Wait()
	fmt.Println("Service stopped")
	loger.Close()
}
