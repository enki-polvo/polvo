package main

import (
	"fmt"
	"os"
	_ "polvo/logger"
	_ "polvo/pipeline"
)

func main() {
	// var (
	// 	pipe pipeline.Pipeline
	// 	err  error
	// )

	// pipe, err = pipeline.NewPipeline("sensor")
	// if err != nil {
	// 	panic(err)
	// }
	// pipe.Start()
	// pipe.Stop()
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	fmt.Println(hostname)
	fmt.Println("hello world")
}
