package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/gookit/color"
	term "github.com/nsf/termbox-go"
)

type Image struct {
	pixels [][]Pixel
	W, H   int
	ratio  float64
}

type Pixel struct {
	r, g, b uint8
}

func convertImage(rawImage image.Image) *Image {
	w := rawImage.Bounds().Max.X
	h := rawImage.Bounds().Max.Y
	img := &Image{
		pixels: make([][]Pixel, h),
		W:      w,
		H:      h,
		ratio:  2 * float64(w) / float64(h),
	}
	for y := 0; y < h; y++ {
		img.pixels[y] = make([]Pixel, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := rawImage.At(x, y).RGBA()
			pixel := Pixel{uint8(r / 257), uint8(g / 257), uint8(b / 257)}
			img.pixels[y][x] = pixel
		}
	}
	return img
}

func LoadImage(path string) ([]*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	isGif := strings.HasSuffix(path, ".gif")
	var imgs []*Image
	if isGif {
		rawImage, err := gif.DecodeAll(f)
		if err != nil {
			return nil, err
		}
		fmt.Println(len(rawImage.Image))
		for _, img := range rawImage.Image {
			imgs = append(imgs, convertImage(img))
		}
	} else {
		rawImage, _, err := image.Decode(f)
		if err != nil {
			return nil, err
		}
		img := convertImage(rawImage)
		imgs = append(imgs, img)
	}
	return imgs, err
}

var (
	path   = flag.String("path", "", "image path")
	symbol = flag.String("symbol", "$", "symbol to draw")
	isBg   = flag.Bool("bg", true, "draw background")

	xl, yl, imgXl, imgYl, xlBounded, ylBounded, cursorX, cursorY, offsetX, offsetY, windowXl, windowYl int
	zoom                                                                                               int = 1
	consoleRatio                                                                                       float64
	imgs                                                                                               []*Image
	canZoom, canOffsetX, canOffsetY                                                                    bool
	imgIndex                                                                                           int = 0
	pixelsInWindow                                                                                     int
	showCursor                                                                                         bool
)

func resetConsole() {
	xl, yl = tm.Width(), tm.Height()
	zoom = 1
	consoleRatio = float64(xl) / float64(yl)
}

func bgFire(r, g, b uint8, symbol string) string {
	return color.RGB(r, g, b, true).Sprint(tm.Color(symbol, tm.BLACK))
}

func fgFire(r, g, b uint8, symbol string) string {
	return color.RGB(r, g, b, false).Sprint(symbol)
}

func recalcWindow() {
	img := imgs[0]
	xlBounded, ylBounded = xl, yl
	if consoleRatio > img.ratio {
		xlBounded = int(float64(yl) * (img.ratio))
	} else {
		ylBounded = int(float64(xl) * (1 / img.ratio))
	}
	xlBounded, ylBounded = xlBounded*zoom, ylBounded*zoom
	windowXl, windowYl = img.W/xlBounded, img.H/ylBounded
	pixelsInWindow = windowXl * windowYl
	if xlBounded*zoom > xl {
		xlBounded = xl
	}
	if ylBounded*zoom > yl {
		ylBounded = yl
	}
	canOffsetY = (ylBounded-1)*windowYl+offsetY < img.H
	canOffsetX = (xlBounded-1)*windowXl+offsetX < img.W
	if zoom == 1 {
		canOffsetX, canOffsetY = true, true
	}
	canZoom = windowXl/2 > 1 && windowYl/2 > 1
}

func zoomIn() {
	if !canZoom {
		return
	}
	zoom *= 2
	if zoom > 1 {
		offsetX = offsetX + windowXl*cursorX
		offsetY = offsetY + windowYl*cursorY
	}
	cursorX, cursorY = 0, 0
	term.SetCursor(cursorX, cursorY)
	recalcWindow()
	tm.Clear()
	draw()
}

func zoomOut() {
	if zoom <= 1 {
		return
	}
	zoom /= 2
	if zoom > 1 {
	} else {
		offsetX, offsetY = 0, 0
	}
	cursorX, cursorY = 0, 0
	term.SetCursor(cursorX, cursorY)
	recalcWindow()
	tm.Clear()
	draw()
}

func cursorUp() {
	if cursorY > 0 {
		cursorY--
		if zoom == 1 {
			term.SetCursor(cursorX, cursorY)
		}
	}
	if zoom > 1 {
		if offsetY > 0 {
			offsetY = offsetY - windowYl
			if offsetY < 0 {
				offsetY = 0
			}
			recalcWindow()
			tm.Clear()
			draw()
		}
	}
}

func cursorDown() {
	img := imgs[0]
	if cursorY < ylBounded-1 {
		cursorY++
		if zoom == 1 {
			term.SetCursor(cursorX, cursorY)
		}
	}
	if zoom > 1 {
		if offsetY < img.H && canOffsetY {
			offsetY = offsetY + windowYl
			recalcWindow()
			tm.Clear()
			draw()
		}
	}
}

func cursorRight() {
	img := imgs[0]
	if cursorX < xlBounded-1 {
		cursorX++
		if zoom == 1 {
			term.SetCursor(cursorX, cursorY)
		}
	}
	if zoom > 1 {
		if offsetX < img.W && canOffsetX {
			offsetX = offsetX + windowXl
			recalcWindow()
			tm.Clear()
			draw()
		}
	}
}

func cursorLeft() {
	if cursorX > 0 {
		cursorX--
		if zoom == 1 {
			term.SetCursor(cursorX, cursorY)
		}
	}
	if zoom > 1 {
		if offsetX > 0 {
			offsetX = offsetX - windowXl
			if offsetX < 0 {
				offsetX = 0
			}
			recalcWindow()
			tm.Clear()
			draw()
		}
	}
}

func toggleCursor() {
	if len(imgs) > 1 {
		showCursor = !showCursor
		tm.Clear()
		draw()
	}
}

func draw() {
	img := imgs[imgIndex]
	for i := 0; i < ylBounded; i++ {
		if i*windowYl+offsetY >= img.H {
			break
		}
		for j := 0; j < xlBounded; j++ {
			if j*windowXl+offsetX >= img.W {
				break
			}
			sumR, sumG, sumB := 0, 0, 0
			for ii := 0; ii < windowYl; ii++ {
				for jj := 0; jj < windowXl; jj++ {
					if i*windowYl+ii+offsetY >= img.H || j*windowXl+jj+offsetX >= img.W {
						break
					}
					p := img.pixels[i*windowYl+ii+offsetY][j*windowXl+jj+offsetX]
					sumR, sumG, sumB = sumR+int(p.r), sumG+int(p.g), sumB+int(p.b)
				}
			}
			realR, realG, realB := sumR/pixelsInWindow, sumG/pixelsInWindow, sumB/pixelsInWindow
			if *isBg {
				tm.Print(bgFire(uint8(realR), uint8(realG), uint8(realB), *symbol))
			} else {
				tm.Print(fgFire(uint8(realR), uint8(realG), uint8(realB), *symbol))
			}
		}
		if i != ylBounded-1 {
			tm.Println()
		}
	}
	if showCursor && zoom == 1 {
		tm.MoveCursor(cursorX, cursorY)
		if *isBg {
			tm.Print(bgFire(255, 0, 0, " "))
		} else {
			tm.Print(fgFire(255, 0, 0, "_"))
		}
	}
	tm.Flush()
	term.SetCursor(0, 0)
}

func reset() {
	resetConsole()
	recalcWindow()
	draw()
}

func main() {
	flag.Parse()
	var err error
	imgs, err = LoadImage(*path)
	if err != nil {
		panic(err)
	}
	err = term.Init()
	if err != nil {
		panic(err)
	}
	defer term.Close()

	tm.Clear()
	tm.MoveCursor(0, 0)
	term.SetCursor(0, 0)
	reset()

	action := make(chan func())
	exit := make(chan struct{})

	go func() {
		for {
			newXl, newYl := tm.Width(), tm.Height()
			if xl != newXl || yl != newYl {
				action <- reset
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			switch ev := term.PollEvent(); ev.Type {
			case term.EventKey:
				switch ev.Key {
				case term.KeyArrowUp:
					action <- (cursorUp)
				case term.KeyArrowDown:
					action <- (cursorDown)
				case term.KeyArrowLeft:
					action <- (cursorLeft)
				case term.KeyArrowRight:
					action <- (cursorRight)
				case term.KeyCtrlC:
					close(exit)
				case term.KeyEnter:
					action <- (zoomIn)
				case term.KeyEsc:
					action <- (zoomOut)
				case term.KeySpace:
					action <- (toggleCursor)
				}
			}
		}
	}()

	go func() {
		if len(imgs) == 1 {
			action <- (draw)
		} else {
			for {
				for i := range imgs {
					imgIndex = i
					action <- (draw)
					time.Sleep(50 * time.Millisecond)
				}
			}
		}
	}()

mainLoop:
	for {
		select {
		case f := <-action:
			f()
		case <-exit:
			break mainLoop
		}
	}
}
