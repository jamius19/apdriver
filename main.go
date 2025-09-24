package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type PotSignal struct {
	mode  byte
	value int
}

const PS_PATH = "C:\\Program Files\\Adobe\\Adobe Photoshop 2025\\Photoshop.exe"

func main() {
	wg := &sync.WaitGroup{}
	pot := make(chan PotSignal, 3)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go HandleDeviceInput(ctx, wg, pot)

	wg.Add(1)
	go HandleForeground(ctx, wg)

	wg.Add(1)
	go HandleSend(ctx, wg, pot)

	<-stop
	fmt.Println("Received interrupt signal, shutting down...")
	cancel()
	wg.Wait()
}
