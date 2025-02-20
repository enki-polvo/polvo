package main

import (
	"fmt"
	"polvo/pipeline"
)

func main() {
	var (
		pipe pipeline.Pipeline
		err  error
	)

	pipe, err = pipeline.NewPipeline("sensor")
	if err != nil {
		panic(err)
	}
	pipe.Start()
	pipe.Stop()
	fmt.Println("hello world")
}
