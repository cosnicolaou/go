// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/luismesas/goPi/piface"
	"github.com/luismesas/goPi/spi"
)

func main() {
	pfd := piface.NewPiFaceDigital(spi.DEFAULT_HARDWARE_ADDR, spi.DEFAULT_BUS, spi.DEFAULT_CHIP)
	if err := pfd.InitBoard(); err != nil {
		fmt.Printf("Error on init board: %s", err)
		return
	}
	relayPin := 0
	pfd.Relays[relayPin].AllOn()
	time.Sleep(400 * time.Millisecond)
	pfd.Relays[relayPin].AllOff()
}
