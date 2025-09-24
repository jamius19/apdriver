package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func HandleDeviceInput(ctx context.Context, wg *sync.WaitGroup, pot chan<- PotSignal) {
	defer wg.Done()

	// 1) find Silicon Labs CP210x (VID_10C4, PID_EA60)
	var portName string
	if ports, _ := enumerator.GetDetailedPortsList(); ports != nil {
		for _, p := range ports {
			if p.VID == "10C4" && p.PID == "EA60" {
				portName = p.Name
				break
			}
		}
	}

	if portName == "" {
		log.Fatal("Artistpad (CP210x) not found")
	}

	fmt.Println("device:", portName)

	// 2) Configure serial port
	mode := &serial.Mode{
		BaudRate: 921600,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	// 3) Open the serial port
	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Fatal("Failed to open serial port:", err)
	}

	defer port.Close()

	err1 := port.SetDTR(false)
	err2 := port.SetRTS(false)
	if err1 != nil || err2 != nil {
		fmt.Errorf("Warning: Failed to set reset signals: DTR=%v, RTS=%v\n", err1, err2)
	}
	time.Sleep(100 * time.Millisecond) // Hold reset for 100ms

	err1 = port.SetDTR(true)
	err2 = port.SetRTS(true)
	if err1 != nil || err2 != nil {
		fmt.Errorf("Warning: Failed to release reset signals: DTR=%v, RTS=%v\n", err1, err2)
	}

	time.Sleep(2 * time.Second) // Increased wait time after reset

	// Flush any existing data in the buffer
	buffer := make([]byte, 1024)
	for {
		port.SetReadTimeout(100 * time.Millisecond) // Short timeout
		n, err := port.Read(buffer)
		if err != nil || n == 0 {
			break // No more data to flush
		}
	}

	_, err = port.Write([]byte{101})
	if err != nil {
		fmt.Printf("Error initiating handshake: %v\n", err)
	}

	// Reset timeout for normal operation
	err = port.SetReadTimeout(50 * time.Millisecond)
	if err != nil {
		fmt.Errorf("Warning: Failed to set read timeout: %v\n", err)
	}

	// 5) Create a scanner to read line by line
	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		mode := line[0]
		valueStr := line[1:]
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			fmt.Errorf("Invalid pot value: %s\n", valueStr)
			continue
		}

		pot <- PotSignal{mode, value}
	}

	if err := scanner.Err(); err != nil {
		fmt.Errorf("Scanner error: %v\n", err)
	}
}
