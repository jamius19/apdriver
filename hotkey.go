package main

import (
	"context"
	"strings"
	"sync"

	"github.com/adam-lavrik/go-imath/ix"
	"github.com/go-vgo/robotgo"
)

const SLEEP_TIME_MS = 5

const PS_PATH = "C:\\Program Files\\Adobe\\Adobe Photoshop 2025\\Photoshop.exe"

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
				if mode != 'x' {
					absValue := ix.Abs(value)
					direction := "right"
					if value < 0 {
						direction = "left"
					}

					if absValue > 25 {
						for i := 0; i < 12; i++ {
							robotgo.KeyTap(direction, "shift")
							robotgo.MilliSleep(SLEEP_TIME_MS)
						}
					} else if absValue > 16 {
						for i := 0; i < 8; i++ {
							robotgo.KeyTap(direction, "shift")
							robotgo.MilliSleep(SLEEP_TIME_MS)
						}
					} else if absValue > 9 {
						for i := 0; i < 3; i++ {
							robotgo.KeyTap(direction, "shift")
							robotgo.MilliSleep(SLEEP_TIME_MS)
						}
					} else if absValue > 0 {
						for i := 0; i < 6; i++ {
							robotgo.KeyTap(direction)
							robotgo.MilliSleep(SLEEP_TIME_MS)
						}
					}
				} else {
					// btn click
					for i := 0; i < 12; i++ {
						robotgo.KeyTap("right", "shift")
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				}

				//else if mode == 'b' {
				//	if value > 0 {
				//		for i := 0; i < 1; i++ {
				//			robotgo.KeyTap("]")
				//		}
				//	} else if value < 0 {
				//		for i := 0; i < 1; i++ {
				//			robotgo.KeyTap("[")
				//		}
				//	}
				//}
			} else {
				// not ps, ignoring
			}
		}
	}
}
