// Package epd2in13bv4 provides a driver for the Waveshare 2.13inch
// e-Paper B V4 display.
package epd2in13bv4

import (
	"fmt"
	"image"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

// Display dimensions in physical portrait pixels.
const (
	Width  = 122
	Height = 250
)

// Landscape dimensions used by BufferFromImage.
const (
	LandscapeWidth  = Height
	LandscapeHeight = Width
)

const defaultChunkSize = 4096

// Config describes the Raspberry Pi wiring and SPI settings for the display.
type Config struct {
	RSTPin    string
	DCPin     string
	BUSYPin   string
	PWRPin    string
	SPIPath   string
	SPISpeed  physic.Frequency
	SPIBits   int
	ChunkSize int
}

// DefaultConfig returns the wiring used by Waveshare's common Raspberry Pi
// examples: RST=GPIO17, DC=GPIO25, BUSY=GPIO24, PWR=GPIO18, SPI=/dev/spidev0.0.
func DefaultConfig() Config {
	return Config{
		RSTPin:    "GPIO17",
		DCPin:     "GPIO25",
		BUSYPin:   "GPIO24",
		PWRPin:    "GPIO18",
		SPIPath:   "/dev/spidev0.0",
		SPISpeed:  4 * physic.MegaHertz,
		SPIBits:   8,
		ChunkSize: defaultChunkSize,
	}
}

// Display represents the e-paper display.
type Display struct {
	rst       gpio.PinOut
	dc        gpio.PinOut
	pwr       gpio.PinOut
	busy      gpio.PinIn
	conn      spi.Conn
	port      spi.PortCloser
	chunkSize int
}

// New initializes GPIO and SPI using DefaultConfig.
func New() (*Display, error) {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig initializes GPIO and SPI using cfg.
func NewWithConfig(cfg Config) (*Display, error) {
	cfg = normalizeConfig(cfg)

	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("host init: %w", err)
	}

	getOut := func(name string) (gpio.PinOut, error) {
		p := gpioreg.ByName(name)
		if p == nil {
			return nil, fmt.Errorf("pin %s not found", name)
		}
		return p, nil
	}

	rst, err := getOut(cfg.RSTPin)
	if err != nil {
		return nil, err
	}
	dc, err := getOut(cfg.DCPin)
	if err != nil {
		return nil, err
	}
	pwr, err := getOut(cfg.PWRPin)
	if err != nil {
		return nil, err
	}

	busyPin := gpioreg.ByName(cfg.BUSYPin)
	if busyPin == nil {
		return nil, fmt.Errorf("pin %s not found", cfg.BUSYPin)
	}
	if err := busyPin.In(gpio.PullDown, gpio.NoEdge); err != nil {
		return nil, fmt.Errorf("set busy pin as input: %w", err)
	}

	port, err := spireg.Open(cfg.SPIPath)
	if err != nil {
		return nil, fmt.Errorf("open SPI: %w", err)
	}

	conn, err := port.Connect(cfg.SPISpeed, spi.Mode0, cfg.SPIBits)
	if err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("connect SPI: %w", err)
	}

	return &Display{
		rst:       rst,
		dc:        dc,
		pwr:       pwr,
		busy:      busyPin,
		conn:      conn,
		port:      port,
		chunkSize: cfg.ChunkSize,
	}, nil
}

// Close puts the display to sleep and releases hardware resources.
func (d *Display) Close() error {
	var firstErr error
	if err := d.Sleep(); err != nil {
		firstErr = err
	}
	if err := d.port.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := d.pwr.Out(gpio.Low); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := d.rst.Out(gpio.Low); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := d.dc.Out(gpio.Low); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// Init performs the hardware initialization sequence.
func (d *Display) Init() error {
	if err := d.pwr.Out(gpio.High); err != nil {
		return err
	}
	if err := d.reset(); err != nil {
		return err
	}

	d.waitBusy()
	if err := d.sendCommand(0x12); err != nil {
		return err
	}
	d.waitBusy()

	if err := d.sendCommand(0x01); err != nil {
		return err
	}
	if err := d.sendData(0xF9); err != nil {
		return err
	}
	if err := d.sendData(0x00); err != nil {
		return err
	}
	if err := d.sendData(0x00); err != nil {
		return err
	}

	if err := d.sendCommand(0x11); err != nil {
		return err
	}
	if err := d.sendData(0x03); err != nil {
		return err
	}

	if err := d.setWindows(0, 0, Width-1, Height-1); err != nil {
		return err
	}
	if err := d.setCursor(0, 0); err != nil {
		return err
	}

	if err := d.sendCommand(0x3C); err != nil {
		return err
	}
	if err := d.sendData(0x05); err != nil {
		return err
	}

	if err := d.sendCommand(0x18); err != nil {
		return err
	}
	if err := d.sendData(0x80); err != nil {
		return err
	}

	if err := d.sendCommand(0x21); err != nil {
		return err
	}
	if err := d.sendData(0x80); err != nil {
		return err
	}
	if err := d.sendData(0x80); err != nil {
		return err
	}

	d.waitBusy()
	return nil
}

// Clear fills both color planes with white and refreshes the display.
func (d *Display) Clear() error {
	buf := WhiteBuffer()
	if err := d.sendCommand(0x24); err != nil {
		return err
	}
	if err := d.sendDataBytes(buf); err != nil {
		return err
	}
	if err := d.sendCommand(0x26); err != nil {
		return err
	}
	if err := d.sendDataBytes(buf); err != nil {
		return err
	}
	if err := d.sendCommand(0x20); err != nil {
		return err
	}
	d.waitBusy()
	return nil
}

// Display sends the two 1-bit buffers to the display and triggers a refresh.
// black: 0-bit = black pixel; red: 0-bit = red/yellow pixel.
func (d *Display) Display(black, red []byte) error {
	if len(black) != BufferSize() {
		return fmt.Errorf("black buffer length %d, want %d", len(black), BufferSize())
	}
	if len(red) != BufferSize() {
		return fmt.Errorf("red buffer length %d, want %d", len(red), BufferSize())
	}

	if err := d.sendCommand(0x24); err != nil {
		return err
	}
	if err := d.sendDataBytes(black); err != nil {
		return err
	}
	if err := d.sendCommand(0x26); err != nil {
		return err
	}
	if err := d.sendDataBytes(red); err != nil {
		return err
	}
	if err := d.sendCommand(0x20); err != nil {
		return err
	}
	d.waitBusy()
	return nil
}

// Sleep switches the display controller into deep sleep.
func (d *Display) Sleep() error {
	if err := d.sendCommand(0x10); err != nil {
		return err
	}
	if err := d.sendData(0x01); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (d *Display) reset() error {
	if err := d.rst.Out(gpio.High); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	if err := d.rst.Out(gpio.Low); err != nil {
		return err
	}
	time.Sleep(2 * time.Millisecond)
	if err := d.rst.Out(gpio.High); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (d *Display) sendCommand(cmd byte) error {
	if err := d.dc.Out(gpio.Low); err != nil {
		return err
	}
	return d.conn.Tx([]byte{cmd}, nil)
}

func (d *Display) sendData(data byte) error {
	if err := d.dc.Out(gpio.High); err != nil {
		return err
	}
	return d.conn.Tx([]byte{data}, nil)
}

func (d *Display) sendDataBytes(data []byte) error {
	if err := d.dc.Out(gpio.High); err != nil {
		return err
	}
	for len(data) > 0 {
		n := len(data)
		if n > d.chunkSize {
			n = d.chunkSize
		}
		if err := d.conn.Tx(data[:n], nil); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (d *Display) waitBusy() {
	for d.busy.Read() == gpio.High {
		time.Sleep(10 * time.Millisecond)
	}
}

func (d *Display) setWindows(xstart, ystart, xend, yend int) error {
	if err := d.sendCommand(0x44); err != nil {
		return err
	}
	if err := d.sendData(byte((xstart >> 3) & 0xFF)); err != nil {
		return err
	}
	if err := d.sendData(byte((xend >> 3) & 0xFF)); err != nil {
		return err
	}

	if err := d.sendCommand(0x45); err != nil {
		return err
	}
	if err := d.sendData(byte(ystart & 0xFF)); err != nil {
		return err
	}
	if err := d.sendData(byte((ystart >> 8) & 0xFF)); err != nil {
		return err
	}
	if err := d.sendData(byte(yend & 0xFF)); err != nil {
		return err
	}
	return d.sendData(byte((yend >> 8) & 0xFF))
}

func (d *Display) setCursor(xstart, ystart int) error {
	if err := d.sendCommand(0x4E); err != nil {
		return err
	}
	if err := d.sendData(byte(xstart & 0xFF)); err != nil {
		return err
	}

	if err := d.sendCommand(0x4F); err != nil {
		return err
	}
	if err := d.sendData(byte(ystart & 0xFF)); err != nil {
		return err
	}
	return d.sendData(byte((ystart >> 8) & 0xFF))
}

func normalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.RSTPin == "" {
		cfg.RSTPin = def.RSTPin
	}
	if cfg.DCPin == "" {
		cfg.DCPin = def.DCPin
	}
	if cfg.BUSYPin == "" {
		cfg.BUSYPin = def.BUSYPin
	}
	if cfg.PWRPin == "" {
		cfg.PWRPin = def.PWRPin
	}
	if cfg.SPIPath == "" {
		cfg.SPIPath = def.SPIPath
	}
	if cfg.SPISpeed == 0 {
		cfg.SPISpeed = def.SPISpeed
	}
	if cfg.SPIBits == 0 {
		cfg.SPIBits = def.SPIBits
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = def.ChunkSize
	}
	return cfg
}

// BufferSize returns the required byte length for one display color plane.
func BufferSize() int {
	return ((Width + 7) / 8) * Height
}

// WhiteBuffer returns a display buffer with every pixel set to white.
func WhiteBuffer() []byte {
	buf := make([]byte, BufferSize())
	for i := range buf {
		buf[i] = 0xFF
	}
	return buf
}

// BufferFromImage converts a landscape 250x122 image to the packed 1-bit
// buffer expected by the display after a 90 degree counter-clockwise rotation.
//
// In the output buffer: 0-bit = ink, 1-bit = white.
func BufferFromImage(img image.Image) []byte {
	buf := WhiteBuffer()

	bounds := img.Bounds()
	if bounds.Dx() != LandscapeWidth || bounds.Dy() != LandscapeHeight {
		return buf
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if int(r)+int(g)+int(b) >= 3*32768 {
				continue
			}
			nx := y - bounds.Min.Y
			ny := LandscapeWidth - 1 - (x - bounds.Min.X)
			byteIdx := ny*((Width+7)/8) + nx/8
			bitIdx := uint(7 - nx%8)
			buf[byteIdx] &^= 1 << bitIdx
		}
	}
	return buf
}

// CountInk returns the number of pixels that are set to ink in a display buffer.
func CountInk(buf []byte) int {
	n := 0
	for _, b := range buf {
		for bit := 7; bit >= 0; bit-- {
			if b&(1<<uint(bit)) == 0 {
				n++
			}
		}
	}
	return n
}
