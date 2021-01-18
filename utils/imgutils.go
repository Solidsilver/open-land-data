package utils

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
)

var WHITE = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
var TRANSP = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
var TRANSP2 = color.NRGBA{R: 255, G: 255, B: 255, A: 0}

func CombineImages(img1 string, img2 string, outImg string) error {
	imgFile1, err := os.Open(img1)
	if err != nil {
		log.Error().Msg("Could not open img1")
		return err
	}
	defer imgFile1.Close()
	imgFile2, err := os.Open(img2)
	if err != nil {
		log.Error().Msg("Could not open img2")
		return err
	}
	defer imgFile2.Close()
	img1D, err := png.Decode(imgFile1)
	if err != nil {
		log.Error().Msg("Could not decode img1")
		return err
	}
	img2D, err := png.Decode(imgFile2)
	if err != nil {
		log.Error().Msg("Could not decode img2")
		return err
	}

	bgWidth, bgHeight := 256, 256
	bgImg := image.NewRGBA(image.Rect(0, 0, bgWidth, bgHeight))

	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)

	draw.Draw(bgImg, img1D.Bounds(), img1D, image.ZP, draw.Over)
	draw.Draw(bgImg, img2D.Bounds(), img2D, image.ZP, draw.Over)

	out, err := os.Create(outImg)
	if err != nil {
		log.Error().Msg("Could not create output image")
		return err
	}
	defer out.Close()
	// var opt jpeg.Options
	// opt.Quality = 80
	// err = jpeg.Encode(out, bgImg, &opt)
	err = png.Encode(out, bgImg)
	if err != nil {
		log.Error().Msg("Could not encode output image")
	}
	return err
}

func GenerateOverviewTile(outName string, img1 string, img2 string, img3 string, img4 string) error {
	imgLocs := []string{img1, img2, img3, img4}
	imgs := make([]image.Image, 4)
	for i, imgLoc := range imgLocs {
		img, err := os.Open(imgLoc)
		defer img.Close()
		var imgDec image.Image
		if err != nil {
			log.Debug().Msgf("Could not open image, using white: %v", imgLoc)
			imgDec = image.NewUniform(TRANSP)
		} else {
			imgDec, err = png.Decode(img)
		}
		imgs[i] = imgDec
	}

	bgWidth, bgHeight := 512, 512
	bgImg := image.NewRGBA(image.Rect(0, 0, bgWidth, bgHeight))
	// draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)

	draw.Draw(bgImg, image.Rect(0, 0, 256, 256), imgs[0], image.ZP, draw.Over)
	draw.Draw(bgImg, image.Rect(256, 0, 512, 256), imgs[1], image.ZP, draw.Over)
	draw.Draw(bgImg, image.Rect(0, 256, 256, 512), imgs[2], image.ZP, draw.Over)
	draw.Draw(bgImg, image.Rect(256, 256, 512, 512), imgs[3], image.ZP, draw.Over)

	imgOut := resize.Resize(256, 256, bgImg, resize.NearestNeighbor)

	os.MkdirAll(filepath.Dir(outName), 0755)
	err := encodePNGToPath(outName, imgOut)

	return err

}

// MergeNTiles takes a list of image paths and produces a direct
// composite output of these images to outImg path with a transparent
// background
func MergeNTiles(imgPaths []string, outImg string) error {
	imgs := make([]image.Image, len(imgPaths))
	for i, imgPath := range imgPaths {
		img, err := decodePNGFromPath(imgPath)
		if err != nil {
			log.Debug().Msgf("Could not open image, using transparent: %v", imgPath)
			img = image.NewUniform(TRANSP)
		}
		imgs[i] = img
	}

	bgWidth, bgHeight := 256, 256
	bgImg := image.NewRGBA(image.Rect(0, 0, bgWidth, bgHeight))

	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{TRANSP}, image.ZP, draw.Src)

	for _, img := range imgs {
		draw.Draw(bgImg, img.Bounds(), img, image.ZP, draw.Over)
	}

	err := encodePNGToPath(outImg, bgImg)
	return err
}

func MergeTiles(img1 string, img2 string, outImg string) error {

	img1D, err := decodePNGFromPath(img1)
	if err != nil {
		return err
	}

	img2D, err := decodePNGFromPath(img2)
	if err != nil {
		return err
	}

	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	transp := color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	bgWidth, bgHeight := 256, 256
	bgImg := image.NewRGBA(image.Rect(0, 0, bgWidth, bgHeight))

	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{transp}, image.ZP, draw.Src)

	img1WhiteP := GetPixelPercent(img1D, white)
	img2WhiteP := GetPixelPercent(img2D, white)
	img1TransP := GetPixelPercent(img1D, transp)
	img2TransP := GetPixelPercent(img2D, transp)

	if math.Abs(img1WhiteP-img2WhiteP) < 0.25 {
		img1D = ReplaceColor(img1D, white, transp)
		img2D = ReplaceColor(img2D, white, transp)
	}

	// Determine drawing order by white content
	if img1WhiteP > img2WhiteP || img1TransP > img2TransP {
		draw.Draw(bgImg, img1D.Bounds(), img1D, image.ZP, draw.Over)
		draw.Draw(bgImg, img2D.Bounds(), img2D, image.ZP, draw.Over)
	} else {
		draw.Draw(bgImg, img2D.Bounds(), img2D, image.ZP, draw.Over)
		draw.Draw(bgImg, img1D.Bounds(), img1D, image.ZP, draw.Over)
	}
	err = encodePNGToPath(outImg, bgImg)
	return err
}

func GetPixelPercent(img image.Image, col color.Color) float64 {
	countColor := 0
	// bounds := img.
	size := img.Bounds().Max
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			pxCol := img.At(x, y)
			if pxCol == col {
				countColor++
			}
		}
	}
	return float64(countColor) / float64(size.X*size.Y)
}

func canDeleteImg(imgPath string) bool {
	img, err := decodePNGFromPath(imgPath)
	if err != nil {
		return false
	}
	notWhiteCount := 0
	size := img.Bounds().Max
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			pxCol := img.At(x, y)
			if pxCol != WHITE && !pixelIsTransparent(pxCol) {
				notWhiteCount++
			}
		}
	}
	return notWhiteCount == 0
}

func pixelIsTransparent(col color.Color) bool {
	_, _, _, a := col.RGBA()
	return a == 0
}

func CleanTileEdge(imgPath string, edge int) error {
	img, _ := decodePNGFromPath(imgPath)
	x, y := 0, 0
	pxRng := IntRange(0, 256)
	if edge%2 == 0 {
		pxRng = IntRange(256, 0)
	}
	outer := &x
	inner := &y
	if edge > 1 {
		outer = &y
		inner = &x
	}

	size := img.Bounds().Max
	m := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))

	for *outer = range pxRng {
		colorCount := 0
		for *inner = range pxRng {
			pxCol := img.At(x, y)
			if pxCol != WHITE && !pixelIsTransparent(pxCol) {
				colorCount++
			}
		}
		for *inner = range pxRng {
			pxCol := img.At(x, y)
			if colorCount == 0 {
				m.Set(x, y, TRANSP)
			} else {
				m.Set(x, y, pxCol)
			}
		}
	}

	os.Remove(imgPath)

	err := encodePNGToPath(imgPath, m)
	return err

}

func ReplaceColor(img image.Image, col color.Color, repl color.Color) image.Image {
	size := img.Bounds().Max
	m := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			pxCol := img.At(x, y)
			if pxCol == col {
				m.Set(x, y, repl)
			} else {
				m.Set(x, y, pxCol)
			}
		}
	}

	return m
}

func decodePNGFromPath(imgPath string) (image.Image, error) {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		log.Error().Err(err).Msgf("Could not open img: %v", imgPath)
		return nil, err
	}
	defer imgFile.Close()
	img, err := png.Decode(imgFile)
	if err != nil {
		log.Error().Err(err).Msgf("Could not decode img: %v", imgPath)
		return nil, err
	}
	return img, nil
}

func encodePNGToPath(imgPath string, img image.Image) error {
	out, err := os.Create(imgPath)
	if err != nil {
		log.Error().Msg("Could not create output image")
		return err
	}
	defer out.Close()
	err = png.Encode(out, img)
	if err != nil {
		log.Error().Msg("Could not encode output image")
	}
	return err
}