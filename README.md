# e-paper-go

Go helpers for driving e-paper displays from Raspberry Pi projects.

Currently included:

- `epd2in13bv4`: Waveshare 2.13inch e-Paper B V4 driver and image-to-buffer conversion.
- `render`: small canvas and text drawing helpers for e-paper screen rendering.

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
