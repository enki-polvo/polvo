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

const logo = "         _nnnn_                      \n" +
	"        dGGGGMMb     ,\"\"\"\"\"\"\"\"\"\"\"\"\".\n" +
	"       @p~qp~~qMb    | Polvo LINUX! |\n" +
	"       M|@||@) M|   _;..............'\n" +
	"       @,----.JM| -'\n" +
	"      JS^\\\\__/  qKL\n" +
	"     dZP        qKRb\n" +
	"    dZP          qKKb\n" +
	"   fZP            SMMb\n" +
	"   HZM            MMMM\n" +
	"   FqM            MMMM\n" +
	" __| \\\".        |\\\\dS\\\"qML\n" +
	" |    `.       | `' \\\\Zq\n" +
	"_)      \\\\.___.,|     .'\n" +
	"\\\\____   )MMMMMM|   .'\n" +
	"     `-'       `--' ascii by hjm\n" +
	"[Polvo_0.0.0 - ENKI WHITEHAT 2025]\n\n"

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
		fmt.Println("Shutting down...")
		err = svc.Stop()
		if err != nil {
			fmt.Printf("error while stop service %v", err)
		}
	}()

	// print logo
	fmt.Print(logo)

	err = svc.Wait()
	if err != nil {
		fmt.Printf("error while waiting service %v", err)
		// call stop to ensure all resources are released
		stop()
	}
	fmt.Println("Service stopped")
	loger.Close()
}
