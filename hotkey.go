package main

import (
	"context"
	"strings"
	"sync"

	"github.com/go-vgo/robotgo"
)

func HandleSend(ctx context.Context, wg *sync.WaitGroup, pot chan PotSignal) {
	defer wg.Done()
	isPs := false

	for {
		select {
		case <-ctx.Done():
			return
		case s := <-ForegroundChanged:
			if strings.EqualFold(s, PS_PATH) {
				isPs = true
			} else {
				isPs = false
			}
		case val := <-pot:
			mode := rune(val.mode)
			value := val.value

			if value == 0 {
				continue
			}

			if isPs {
				if mode == 'a' {
					if value > 14 {
						for i := 0; i < 5; i++ {
							robotgo.KeyTap("right", "shift")
						}
					} else if value > 7 {
						for i := 0; i < 3; i++ {
							robotgo.KeyTap("right", "shift")
						}
					} else if value > 0 {
						for i := 0; i < 7; i++ {
							robotgo.KeyTap("right")
						}
					} else if value < -14 {
						for i := 0; i < 5; i++ {
							robotgo.KeyTap("left", "shift")
						}
					} else if value < -7 {
						for i := 0; i < 3; i++ {
							robotgo.KeyTap("left", "shift")
						}
					} else if value < 0 {
						for i := 0; i < 7; i++ {
							robotgo.KeyTap("left")
						}
					}
				} else {
					if value > 0 {
						for i := 0; i < 1; i++ {
							robotgo.KeyTap("]")
						}
					} else if value < 0 {
						for i := 0; i < 1; i++ {
							robotgo.KeyTap("[")
						}
					}
				}
			} else {
				// not ps, ignoring
			}
		}
	}
}
