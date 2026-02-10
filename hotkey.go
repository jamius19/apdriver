package main

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adam-lavrik/go-imath/ix"
	"github.com/go-vgo/robotgo"
)

const SLEEP_TIME_MS = 5

const PS_PATH = "Photoshop.exe"
const REBELLE7_PATH = "Rebelle 7.exe"
const REBELLE8_PATH = "Rebelle 8.exe"
const KRITA_PATH = "krita.exe"

func HandleSend(ctx context.Context, wg *sync.WaitGroup, pot chan PotSignal) {
	defer wg.Done()
	isPs := false
	isRebelle := false
	isKrita := false

	for {
		select {
		case <-ctx.Done():
			return
		case s := <-ForegroundChanged:
			if strings.EqualFold(filepath.Base(s), PS_PATH) {
				isPs = true
				isRebelle = false
				isKrita = false
			} else if strings.EqualFold(filepath.Base(s), REBELLE7_PATH) || strings.EqualFold(filepath.Base(s), REBELLE8_PATH) {
				isPs = false
				isRebelle = true
				isKrita = false
			} else if strings.EqualFold(filepath.Base(s), KRITA_PATH) {
				isPs = false
				isRebelle = false
				isKrita = true
			} else {
				isPs = false
				isRebelle = false
				isKrita = false
			}
		case val := <-pot:
			mode := rune(val.mode)
			value := val.value

			if value == 0 {
				continue
			}
			absValue := ix.Abs(value)
			direction := "right"
			if value < 0 {
				direction = "left"
			}

			if isPs {
				if mode != 'x' {
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
			} else if isRebelle {
				if absValue > 16 {
					for i := 0; i < 13; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 9 {
					for i := 0; i < 6; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 0 {
					for i := 0; i < 2; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				}
			} else if isKrita {
				if absValue > 16 {
					for i := 0; i < 9; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 9 {
					for i := 0; i < 6; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 0 {
					for i := 0; i < 2; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				}
			} else {
				direction := "audio_vol_up"
				if value < 0 {
					direction = "audio_vol_down"
				}

				if absValue > 12 {
					for i := 0; i < 5; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 6 {
					for i := 0; i < 3; i++ {
						robotgo.KeyTap(direction)
						robotgo.MilliSleep(SLEEP_TIME_MS)
					}
				} else if absValue > 0 {
					robotgo.KeyTap(direction)
					robotgo.MilliSleep(SLEEP_TIME_MS)
				}
			}
		}
	}
}
