package main

import (
	"image/color"
	"machine"
	"math"
	"math/rand"
	"time"

	"tinygo.org/x/drivers/buzzer"
	"tinygo.org/x/drivers/lis3dh"
	"tinygo.org/x/drivers/ws2812"
)

const (
	NUMLEDS  = 53
	NUMMODES = 3

	SHAKE_THRESHOLD    = 13000.0
	MOVEMENT_THRESHOLD = 3000.0
	SMOOTH_FACTOR      = 0.8

	SHAKE_DURATION = 50
)
const (
	RAINBOW = iota
	FILLUP
	SHAKE
)

var (
	neo             machine.Pin
	leds            [NUMLEDS]color.RGBA
	adcInitComplete = false
	i2cInitComplete = false

	neopixelsPin     = machine.NEOPIXELS
	btnAPin          = machine.BUTTONA
	btnBPin          = machine.BUTTONB
	sclPin           = machine.SCL1_PIN
	sdaPin           = machine.SDA1_PIN
	ledPin           = machine.LED
	tempPin          = machine.TEMPSENSOR
	lightPin         = machine.LIGHTSENSOR
	sliderPin        = machine.SLIDER
	speakerEnablePin = machine.D11
	speakerPin       = machine.D12
	i2c              = machine.I2C0
	bigLed           ws2812.Device
	neoBack          ws2812.Device
	ledsBack         = make([]color.RGBA, 10)
	accel            lis3dh.Device
	bzr              buzzer.Device

	btnAPressed        = false
	btnBPressed        = false
	shakeTimer  uint32 = 0
	GREEN              = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	RED                = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	OFF                = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	ORANGE             = color.RGBA{R: 255, G: 165, B: 0, A: 255}
)

func main() {

	println("Configuring hardware...")
	setupHardware()
	startupEffect()
	println("Bootup completed")

	showShakeState()

	mode := RAINBOW
	var x, y, z int16
	var magnitude, smoothedMagnitude, accelerationChange, lastMagnitude float64

	n := uint8(0)
	var randColor color.RGBA
	for {

		if btnBPin.Get() {
			if !btnBPressed {
				mode++
				if mode >= NUMMODES {
					mode = 0
				}
				if mode == SHAKE {
					showOffState()
				}
				if mode == FILLUP {
					n = 0
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
			bigLed.WriteColors(leds[:])
			time.Sleep(100 * time.Millisecond)
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
			bigLed.WriteColors(leds[:])
			time.Sleep(100 * time.Millisecond)
			break
		case SHAKE:
			x, y, z = accel.ReadRawAcceleration()

			magnitude = calculateMagnitude(x, y, z)

			smoothedMagnitude = SMOOTH_FACTOR*lastMagnitude + (1-SMOOTH_FACTOR)*magnitude
			lastMagnitude = smoothedMagnitude

			accelerationChange = math.Abs(smoothedMagnitude - 16384)

			if accelerationChange > SHAKE_THRESHOLD {
				go Bleep()
				showShakeState()
				shakeTimer = SHAKE_DURATION
			} else if shakeTimer > 0 {
				shakeTimer--
				if shakeTimer == 0 {
					go Blip()
					showNormalState()
				}
			} else if accelerationChange > MOVEMENT_THRESHOLD {
				showMovementState()
			} else {
				showNormalState()
			}

			time.Sleep(20 * time.Millisecond)
			break
		}

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

func calculateMagnitude(x, y, z int16) float64 {
	fx := int64(x)
	fy := int64(y)
	fz := int64(z)
	return math.Sqrt(float64(fx*fx + fy*fy + fz*fz))
}

func setAllPixels(c color.RGBA) {
	for i := 0; i < len(ledsBack); i++ {
		ledsBack[i] = c
	}
}

func setPixel(index int, c color.RGBA) {
	if index >= 0 && index < len(ledsBack) {
		ledsBack[index] = c
	}
}

func updatePixels() {
	neoBack.WriteColors(ledsBack)
}

func showNormalState() {
	setAllPixels(GREEN)
	updatePixels()
}

func showShakeState() {
	for i := 0; i < 3; i++ {
		setAllPixels(RED)
		updatePixels()
		time.Sleep(150 * time.Millisecond)

		setAllPixels(OFF)
		updatePixels()
		time.Sleep(100 * time.Millisecond)
	}

	setAllPixels(RED)
	updatePixels()
}

func showMovementState() {
	setAllPixels(ORANGE)
	updatePixels()
}

func showOffState() {
	setAllPixels(OFF)
	updatePixels()
}

func startupEffect() {
	setAllPixels(OFF)
	updatePixels()

	for i := 0; i < len(ledsBack); i++ {
		setPixel(i, GREEN)
		updatePixels()
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	setAllPixels(OFF)
	updatePixels()
	time.Sleep(200 * time.Millisecond)

	showNormalState()
}

// Bleep makes a bleep sound using the speaker.
func Bleep() {
	bzr.Tone(buzzer.C3, buzzer.Eighth)
}

// Bloop makes a bloop sound using the speaker.
func Bloop() {
	bzr.Tone(buzzer.C5, buzzer.Quarter)
}

// Blip makes a blip sound using the speaker.
func Blip() {
	bzr.Tone(buzzer.C6, buzzer.Eighth/8)
}

func setupHardware() {
	neopixelsPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	neoBack = ws2812.New(neopixelsPin)

	speakerShutdown := speakerEnablePin
	speakerShutdown.Configure(machine.PinConfig{Mode: machine.PinOutput})
	speakerShutdown.High()

	speaker := speakerPin
	speaker.Configure(machine.PinConfig{Mode: machine.PinOutput})

	bzr = buzzer.New(speaker)

	btnAPin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	btnBPin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	neo = machine.A2
	neo.Configure(machine.PinConfig{Mode: machine.PinOutput})

	bigLed = ws2812.NewWS2812(neo)

	err := i2c.Configure(machine.I2CConfig{
		Frequency: machine.TWI_FREQ_400KHZ,
		SDA:       machine.SDA1_PIN,
		SCL:       machine.SCL1_PIN,
	})
	if err != nil {
		println("Error configurando I2C:", err.Error())
		// Mostrar error con parpadeo rojo rápido
		for i := 0; i < 10; i++ {
			setAllPixels(RED)
			updatePixels()
			time.Sleep(100 * time.Millisecond)
			setAllPixels(OFF)
			updatePixels()
			time.Sleep(100 * time.Millisecond)
		}
		return
	}

	accel = lis3dh.New(i2c)
	accel.Address = lis3dh.Address1 // Dirección por defecto
	accel.Configure()
	accel.SetRange(lis3dh.RANGE_2_G)

	setAllPixels(OFF)
	updatePixels()
}
