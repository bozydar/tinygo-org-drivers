package main

import (
	"image/color"
	"machine"
	driver "tinygo.org/x/drivers/waveshare-epd/epd2in9bwr"

	"time"
)

var display driver.Device

const (
	width  = 128
	height = 296
)

func main() {
	machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 4_000_000,
		Mode:      0,
	})
	spi := machine.SPI1
	machine.GPIO9.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	csPin := machine.GPIO9

	machine.GPIO8.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	dcPin := machine.GPIO8

	machine.GPIO12.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	rstPin := machine.GPIO12

	machine.GPIO13.Configure(machine.PinConfig{
		Mode: machine.PinInput,
	})
	busyPin := machine.GPIO13

	time.Sleep(3 * time.Second)
	println("Clear the display")

	display = driver.New(spi, csPin, dcPin, rstPin, busyPin)
	display.Configure(driver.Config{
		Width:  width,
		Height: height,
	})

	black := color.RGBA{1, 1, 1, 255}

	display.ClearBuffer()
	println("Clear the display")
	display.ClearDisplay()
	display.WaitUntilIdle()
	println("Waiting for 2 seconds")
	time.Sleep(2 * time.Second)

	// Show a checkered board
	for i := int16(0); i < 10; i++ {
		for j := int16(0); j < 20; j++ {
			if (i+j)%2 == 0 {
				showRect(i*8, j*8, 8, 8, black)
			}
		}
	}
	println("Show checkered board")
	display.Display()
	display.WaitUntilIdle()
	println("Waiting for 2 seconds")
	time.Sleep(2 * time.Second)

	println("You could remove power now")
}

func showRect(x int16, y int16, w int16, h int16, c color.RGBA) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			display.SetPixel(i, j, c)
		}
	}
}
