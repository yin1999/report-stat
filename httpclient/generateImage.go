package httpclient

import (
	"bufio"
	"context"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var fonts *sfnt.Font

type status struct {
	LastModified int64          `json:"lastModified"`
	Remains      map[string]int `json:"remains"`
}

func init() {
	data, err := os.ReadFile("font.otf")
	if err != nil {
		data, err = os.ReadFile("font.ttf")
		if err != nil {
			panic(err)
		}
	}
	fonts, err = opentype.Parse(data)
	if err != nil {
		panic(err)
	}
}

// generateImage generate image from detail array
//
// Note: detail must be sorted
func generateImage(ctx context.Context, detail detailArray, account *Account, lastModified int64) (err error) {
	if !sort.StringsAreSorted(account.Class) {
		sort.Strings(account.Class)
	}
	if err = os.MkdirAll(account.Out, 0755); err != nil {
		return
	}
	end := 0
	stat := status{
		LastModified: lastModified,
		Remains:      make(map[string]int, len(account.Class)),
	}
	for _, classname := range account.Class {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var data detailArray
		if classname == "全部" {
			data = detail
		} else if end != len(detail) {
			index := sort.Search(len(detail), func(i int) bool {
				return detail[i].class() > classname
			})
			if index != 0 && detail[index-1].class() == classname {
				data = detail[end:index]
				end = index
			}
		}
		stat.Remains[classname] = len(data)
		if err = toPic(data, filepath.Join(account.Out, classname+".webp"), classname == "全部"); err != nil {
			return
		}
	}
	return storeJson(stat, filepath.Join(account.Out, "status.json"))
}

func toPic(detail detailArray, filename string, showClass bool) error {
	// Initialize the context.
	fg, bg := image.Black, image.White
	ruler := color.RGBA{204, 204, 204, 0xff}
	const (
		idWidth    = 122
		nameWidth  = 86
		classWidth = 102
		height     = 25
		fontSize   = 18
	)
	W := idWidth + nameWidth
	if showClass {
		W += classWidth
	}
	H := len(detail)*height + height
	rgba := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, draw.Src)

	// Draw the guidelines.
	for h := 0; h < H; h += height {
		drawLine(rgba, 0, h, W, h, ruler)
	}
	drawLine(rgba, 0, H-1, W, H-1, ruler)

	drawLine(rgba, 0, 0, 0, H, ruler)
	drawLine(rgba, W-1, 0, W-1, H, ruler)
	drawLine(rgba, idWidth-1, 0, idWidth-1, H, ruler)

	if showClass {
		drawLine(rgba, idWidth+nameWidth, 0, idWidth+nameWidth, H, ruler)
	}

	// Draw the text.
	face, _ := opentype.NewFace(fonts, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	d := font.Drawer{
		Dst:  rgba,
		Src:  fg,
		Face: face,
	}
	width1 := fixed.I(idWidth >> 1)
	width2 := fixed.I(idWidth + (nameWidth >> 1))
	width3 := fixed.I(idWidth + nameWidth + (classWidth >> 1))

	print := func(x, y fixed.Int26_6, text string) {
		d.Dot.X = x - d.MeasureString(text)/2
		d.Dot.Y = y
		d.DrawString(text)
	}
	fixedHeight := fixed.I(height)
	y := fixed.I(20)
	print(width1, y, "学号")
	print(width2, y, "姓名")
	if showClass {
		print(width3, y, "班级")
	}
	for i := range detail {
		y += fixedHeight
		print(width1, y, detail[i].id())
		print(width2, y, detail[i].name())
		if showClass {
			print(width3, y, detail[i].class())
		}
	}
	// Save that RGBA image to disk.
	op, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 85)
	if err != nil {
		return err
	}
	var outFile *os.File
	outFile, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = webp.Encode(b, rgba, op)
	if err != nil {
		return err
	}
	return b.Flush()
}

func drawLine(rgba *image.RGBA, x1, y1, x2, y2 int, color color.RGBA) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		rgba.Set(x1, y1, color)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}
