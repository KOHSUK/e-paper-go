# e-paper-go

Go helpers for driving e-paper displays from Raspberry Pi projects.

Currently included:

- `epd2in13bv4`: Waveshare 2.13inch e-Paper B V4 driver and image-to-buffer conversion.
- `render`: small canvas and text drawing helpers for e-paper screen rendering.

## Tested hardware

This library has been tested with the following e-paper module:

- Amazon.co.jp product page: https://www.amazon.co.jp/dp/B08FYTNZNN
- ASIN: `B08FYTNZNN`
- Product listing title includes `2.13インチ`, `スターターセット`, `Raspberry`, and `ラズベリーパイゼロ`.

Reference manuals:

- Waveshare 2.13inch e-Paper HAT Manual: https://www.waveshare.com/wiki/2.13inch_e-Paper_HAT
- Waveshare 2.13inch e-Paper HAT (B) Manual: https://www.waveshare.com/wiki/2.13inch_e-Paper_HAT_(B)_Manual

The `epd2in13bv4` package targets the Waveshare 2.13inch e-Paper HAT (B) V4 compatible module. It assumes a 250x122 landscape drawing area, a 122x250 physical display buffer, SPI mode 0, and separate black/white and red/white color planes.

The default Raspberry Pi wiring follows the Waveshare HAT pinout:

- `RST`: `GPIO17`
- `DC`: `GPIO25`
- `BUSY`: `GPIO24`
- `PWR`: `GPIO18`
- `DIN`: `MOSI`
- `CLK`: `SCLK`
- `CS`: `CE0`
- SPI: `/dev/spidev0.0`, mode 0, 4 MHz

If you use different wiring, pass a custom `epd2in13bv4.Config` to `epd2in13bv4.NewWithConfig`.

```go
display, err := epd2in13bv4.New()
if err != nil {
	log.Fatal(err)
}
defer display.Close()

black, red := render.NewPair(epd2in13bv4.LandscapeWidth, epd2in13bv4.LandscapeHeight)
blackBuf := epd2in13bv4.BufferFromImage(black)
redBuf := epd2in13bv4.BufferFromImage(red)

if err := display.Display(blackBuf, redBuf); err != nil {
	log.Fatal(err)
}
```
