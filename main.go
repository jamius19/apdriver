package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
)

//go:embed assets/artistpad_icon.ico
var ArtistpadIcon []byte

type PotSignal struct {
	mode  byte
	value int
}

var stop = make(chan os.Signal, 1)

func main() {
	go systray.Run(onReady, onExit)
	beeep.AppName = "Artistpad Driver"

	wg := &sync.WaitGroup{}
	pot := make(chan PotSignal, 3)

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

func onReady() {
	systray.SetIcon(ArtistpadIcon)
	systray.SetTitle("Artistpad Driver")
	systray.SetTooltip("Artistpad Driver")

	mquit := systray.AddMenuItem("Quit", "Quit the driver")

	go func() {
		<-mquit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	stop <- os.Interrupt
}
