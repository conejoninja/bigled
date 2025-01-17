// Connects to an WS2812 RGB LED strip with 10 LEDS.
//
// See either the others.go or digispark.go files in this directory
// for the neopixels pin assignments.
package main

import (
	"image/color"
	"machine"
	"math/rand"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const (
	NUMLEDS  = 53
	NUMMODES = 2
)
const (
	RAINBOW = iota
	FILLUP
)

var (
	neo             machine.Pin
	leds            [NUMLEDS]color.RGBA
	adcInitComplete = false
	i2cInitComplete = false

	neopixelsPin = machine.NEOPIXELS
	btnAPin      = machine.BUTTONA
	btnBPin      = machine.BUTTONB
	sclPin       = machine.SCL1_PIN
	sdaPin       = machine.SDA1_PIN
	ledPin       = machine.LED
	tempPin      = machine.TEMPSENSOR
	lightPin     = machine.LIGHTSENSOR
	sliderPin    = machine.SLIDER

	btnAPressed = false
	btnBPressed = false
)

func main() {
	//machine.InitADC()
	//machine.I2C1.Configure(machine.I2CConfig{SCL: sclPin, SDA: sdaPin})

	neo = machine.A2
	neo.Configure(machine.PinConfig{Mode: machine.PinOutput})

	ws := ws2812.NewWS2812(neo)
	mode := RAINBOW

	n := uint8(0)
	var randColor color.RGBA
	for {

		if !btnBPin.Get() {
			if btnBPressed {
				mode++
				if mode >= NUMMODES {
					mode = 0
				}
			}
			btnBPressed = true
		} else {
			btnBPressed = false
		}

		switch mode {
		case RAINBOW:
			for i := uint8(0); i < NUMLEDS; i++ {
				leds[i] = getRainbowRGB(n + i)
			}
			n++
			break
		case FILLUP:
			for i := uint8(0); i < n; i++ {
				leds[i] = randColor
			}
			n++
			if n >= NUMLEDS {
				n = 0
				randColor = getRainbowRGB(uint8(rand.Intn(255)))
			}
			break
		}

		ws.WriteColors(leds[:])
		time.Sleep(100 * time.Millisecond)
	}
}

func getRainbowRGB(i uint8) color.RGBA {
	if i < 85 {
		return color.RGBA{i * 3, 255 - i*3, 0, 255}
	} else if i < 170 {
		i -= 85
		return color.RGBA{255 - i*3, 0, i * 3, 255}
	}
	i -= 170
	return color.RGBA{0, i * 3, 255 - i*3, 255}
}
