// It looks like epd2in13x is the nearest one
package epd2in9bwr // import "tinygo.org/x/drivers/waveshare-epd/epd2in9"

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers"
)

type Config struct {
	Width        int16 // Width is the display resolution
	Height       int16
	LogicalWidth int16    // LogicalWidth must be a multiple of 8 and same size or bigger than Width
	Rotation     Rotation // Rotation is clock-wise
}

type Device struct {
	bus          drivers.SPI
	cs           machine.Pin
	dc           machine.Pin
	rst          machine.Pin
	busy         machine.Pin
	logicalWidth int16
	width        int16
	height       int16
	buffer       [][]uint8
	bufferLength uint32
	rotation     Rotation
}

type Rotation uint8
type Color uint8

const (
	NoRotation  Rotation = 0
	Rotation90  Rotation = 1 // 90 degrees clock-wise rotation
	Rotation180 Rotation = 2
	Rotation270 Rotation = 3
)

// New returns a new epd2in9 driver. Pass in a fully configured SPI bus.
func New(bus drivers.SPI, csPin, dcPin, rstPin, busyPin machine.Pin) Device {
	csPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	dcPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	rstPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	busyPin.Configure(machine.PinConfig{Mode: machine.PinInput})
	return Device{
		bus:  bus,
		cs:   csPin,
		dc:   dcPin,
		rst:  rstPin,
		busy: busyPin,
	}
}

// Configure sets up the device.
func (d *Device) Configure(cfg Config) {
	if cfg.LogicalWidth != 0 {
		d.logicalWidth = cfg.LogicalWidth
	} else {
		d.logicalWidth = 128
	}
	if cfg.Width != 0 {
		d.width = cfg.Width
	} else {
		d.width = 128
	}
	if cfg.Height != 0 {
		d.height = cfg.Height
	} else {
		d.height = 296
	}
	d.rotation = cfg.Rotation
	d.bufferLength = (uint32(d.logicalWidth) * uint32(d.height)) / 8
	d.buffer = make([][]uint8, 2)
	for i := range d.buffer {
		d.buffer[i] = make([]uint8, d.bufferLength)
	}
	for i := range d.buffer {
		for j := uint32(0); j < d.bufferLength; j++ {
			d.buffer[i][j] = 0xFF
		}
	}
	d.Reset()

	d.SendCommand(0x04)
	d.WaitUntilIdle()

	println("Panel setting...")
	d.SendCommand(PANEL_SETTING)
	d.SendData(0x0f)
	d.SendData(0x89)

	println("Resolution setting...")
	d.SendCommand(RESOLUTION_SETTING)
	d.SendData(0x80)
	d.SendData(0x01)
	d.SendData(0x28)

	println("Vcom setting...")
	d.SendCommand(VCOM_AND_DATA_INTERVAL_SETTING)
	d.SendData(0x77)

}

// Reset resets the device
func (d *Device) Reset() {
	d.rst.High()
	time.Sleep(50 * time.Millisecond)
	d.rst.Low()
	time.Sleep(2 * time.Millisecond)
	d.rst.High()
	time.Sleep(50 * time.Millisecond)
}

// DeepSleep puts the display into deepsleep
func (d *Device) DeepSleep() {
	d.SendCommand(POWER_OFF)
	d.WaitUntilIdle()
	d.SendCommand(DEEP_SLEEP)
	d.SendData(0xa5)

	time.Sleep(2_000 * time.Millisecond)
	d.rst.Low()
}

// SendCommand sends a command to the display
func (d *Device) SendCommand(command uint8) {
	d.dc.Low()
	d.cs.Low()
	d.bus.Transfer(command)
	d.cs.High()
}

// SendData sends a data byte to the display
func (d *Device) SendData(data uint8) {
	d.dc.High()
	d.cs.Low()
	d.bus.Transfer(data)
	d.cs.High()
}

// Display sends the buffer (if any) to the screen.
func (d *Device) Display() error {
	d.SendCommand(DATA_START_TRANSMISSION_1) // black
	time.Sleep(2 * time.Millisecond)
	for i := uint32(0); i < d.bufferLength; i++ {
		d.SendData(d.buffer[BLACK-1][i])
	}
	time.Sleep(2 * time.Millisecond)
	d.SendCommand(DATA_START_TRANSMISSION_2) // red
	time.Sleep(2 * time.Millisecond)
	for i := uint32(0); i < d.bufferLength; i++ {
		d.SendData(d.buffer[COLORED-1][i])
	}
	time.Sleep(2 * time.Millisecond)
	d.TurnOnDisplay()
	return nil
}

// ClearDisplay erases the device SRAM
func (d *Device) ClearDisplay() {
	d.SendCommand(DATA_START_TRANSMISSION_1) // black
	time.Sleep(2 * time.Millisecond)
	for i := uint32(0); i < d.bufferLength; i++ {
		d.SendData(0xFF)
	}
	time.Sleep(2 * time.Millisecond)
	d.SendCommand(DATA_START_TRANSMISSION_2) // red
	time.Sleep(2 * time.Millisecond)
	for i := uint32(0); i < d.bufferLength; i++ {
		d.SendData(0xFF)
	}
	time.Sleep(2 * time.Millisecond)
	d.TurnOnDisplay()
}

// ClearBuffer sets the buffer to 0xFF (white)
func (d *Device) ClearBuffer() {
	for i := uint8(0); i < uint8(len(d.buffer)); i++ {
		for j := uint32(0); j < d.bufferLength; j++ {
			d.buffer[i][j] = 0xFF
		}
	}
}

// WaitUntilIdle waits until the display is ready
func (d *Device) WaitUntilIdle() {
	println("busy")
	for {
		d.SendCommand(0x71)
		if d.busy.Get() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	println("Idle")
}

// IsBusy returns the busy status of the display
func (d *Device) IsBusy() bool {
	return d.busy.Get()
}

func (d *Device) TurnOnDisplay() {
	d.SendCommand(0x12)
	d.WaitUntilIdle()
}

// SetRotation changes the rotation (clock-wise) of the device
func (d *Device) SetRotation(rotation Rotation) {
	d.rotation = rotation
}

// SetPixel modifies the internal buffer in a single pixel.
// The display have 3 colors: black, white and a third color that could be red or yellow
// We use RGBA(0,0,0, 255) as white (transparent)
// RGBA(1-255,0,0,255) as colored (red)
// Anything else as black
func (d *Device) SetPixel(x int16, y int16, c color.RGBA) {
	x, y = d.xy(x, y)
	if x < 0 || x >= d.logicalWidth || y < 0 || y >= d.height {
		return
	}
	if x < 0 || x >= d.width || y < 0 || y >= d.height {
		return
	}
	if c.R != 0 && c.G == 0 && c.B == 0 { // COLORED
		d.SetEPDPixel(x, y, COLORED)
	} else if c.G != 0 || c.B != 0 { // BLACK
		d.SetEPDPixel(x, y, BLACK)
	} else { // WHITE / EMPTY
		d.SetEPDPixel(x, y, WHITE)
	}
}

func (d *Device) Size() (x, y int16) {
	return d.width, d.height
}

func (d *Device) SetEPDPixel(x int16, y int16, c Color) {
	if x < 0 || x >= d.width || y < 0 || y >= d.height {
		return
	}
	byteIndex := (x + y*d.width) / 8
	println("x", x, "y", y, "byteIndex", byteIndex, "c", c)

	if c == WHITE {
		d.buffer[BLACK-1][byteIndex] |= 0x80 >> uint8(x%8)
		d.buffer[COLORED-1][byteIndex] |= 0x80 >> uint8(x%8)
	} else if c == COLORED {
		d.buffer[BLACK-1][byteIndex] |= 0x80 >> uint8(x%8)
		d.buffer[COLORED-1][byteIndex] &^= 0x80 >> uint8(x%8)
	} else { // BLACK
		d.buffer[COLORED-1][byteIndex] |= 0x80 >> uint8(x%8)
		d.buffer[BLACK-1][byteIndex] &^= 0x80 >> uint8(x%8)
	}
}

// xy changes the coordinates according to the rotation
func (d *Device) xy(x, y int16) (int16, int16) {
	switch d.rotation {
	case NoRotation:
		return x, y
	case Rotation90:
		return d.width - y - 1, x
	case Rotation180:
		return d.width - x - 1, d.height - y - 1
	case Rotation270:
		return y, d.height - x - 1
	}
	return x, y
}
