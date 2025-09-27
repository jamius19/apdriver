package main

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func HandleDeviceInput(ctx context.Context, wg *sync.WaitGroup, pot chan<- PotSignal) {
	defer wg.Done()

DeviceLoop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var portName string

		for portName == "" {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if ports, _ := enumerator.GetDetailedPortsList(); ports != nil {
				for _, p := range ports {
					if p.VID == "10C4" && p.PID == "EA60" {
						portName = p.Name
						break
					}
				}
			}
		}

		fmt.Println("device:", portName)

		time.Sleep(1 * time.Second)

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
			fmt.Errorf("Failed to open serial port:", err)
			_ = beeep.Alert("Device Info", "Failed to connect. Restarting device.", ArtistpadIcon)

			continue DeviceLoop
		}

		_ = beeep.Alert("Device Info", "Artistpad is connected", ArtistpadIcon)

		err1 := port.SetDTR(false)
		err2 := port.SetRTS(false)
		if err1 != nil || err2 != nil {
			fmt.Errorf("Warning: Failed to set reset signals: DTR=%v, RTS=%v\n", err1, err2)
			_ = beeep.Alert("Device Info", "Failed to connect to Artistpad. Restarting device.", ArtistpadIcon)

			_ = port.Close()
			continue DeviceLoop
		}
		time.Sleep(100 * time.Millisecond) // Hold reset for 100ms

		err1 = port.SetDTR(true)
		err2 = port.SetRTS(true)
		if err1 != nil || err2 != nil {
			fmt.Errorf("Warning: Failed to release reset signals: DTR=%v, RTS=%v\n", err1, err2)
			_ = beeep.Alert("Device Info", "Failed to connect to Artistpad. Restarting device.", ArtistpadIcon)

			_ = port.Close()
			continue DeviceLoop
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
			fmt.Errorf("Error initiating handshake: %v\n", err)
			_ = beeep.Alert("Device Info", "Failed to connect to Artistpad. Restarting device.", ArtistpadIcon)

			_ = port.Close()
			continue DeviceLoop
		}

		// Reset timeout for normal operation
		err = port.SetReadTimeout(50 * time.Millisecond)
		if err != nil {
			fmt.Errorf("Warning: Failed to set read timeout: %v\n", err)
			_ = beeep.Alert("Device Info", "Failed to connect to Artistpad. Restarting device.", ArtistpadIcon)

			_ = port.Close()
			continue DeviceLoop
		}

		// 5) Create a scanner to read line by line
		scanner := bufio.NewScanner(port)
		errorCount := 0

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				_ = port.Close()
				return
			default:
			}

			line := scanner.Text()
			mode := line[0]
			if mode == 'x' {
				pot <- PotSignal{mode, -100}
				continue
			}

			valueStr := line[1:]
			value, err := strconv.Atoi(valueStr)
			if err != nil {
				fmt.Errorf("Invalid pot value: %s\n", valueStr)
				errorCount++
				if errorCount >= 5 {
					fmt.Println("Too many errors, restarting device read...")
					_ = beeep.Alert("Device Info", "Artistpad connection error. Restarting device.", ArtistpadIcon)

					_ = port.Close()
					continue DeviceLoop
				}

				continue
			}

			pot <- PotSignal{mode, value}
		}

		if err := scanner.Err(); err != nil {
			fmt.Errorf("Scanner error: %v\n", err)
			_ = beeep.Alert("Device Info", "Artistpad connection error. Restarting device.", ArtistpadIcon)

			_ = port.Close()
			continue DeviceLoop
		}
	}
}
