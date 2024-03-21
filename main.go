package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/jackmordaunt/icns/v3"
)

type icondir struct {
	reserved  uint16
	imageType uint16
	numImages uint16
}

type icondirentry struct {
	imageWidth   uint8
	imageHeight  uint8
	numColors    uint8
	reserved     uint8
	colorPlanes  uint16
	bitsPerPixel uint16
	sizeInBytes  uint32
	offset       uint32
}

func newIcondir(numImages uint16) icondir {
	var id icondir
	id.imageType = 1
	id.numImages = numImages
	return id
}

// https://en.wikipedia.org/wiki/ICO_(file_format)
func newIcondirentry() icondirentry {
	var ide icondirentry
	ide.colorPlanes = 1   // windows is supposed to not mind 0 or 1, but other icon files seem to have 1 here
	ide.bitsPerPixel = 32 // can be 24 for bitmap or 24/32 for png. Set to 32 for now
	ide.offset = 22       //6 icondir + 16 icondirentry, next image will be this image size + 16 icondirentry, etc
	return ide
}

func main() {
	// 定义命令行参数
	var inputPath string
	var outputName string
	var icoName string
	var icnsName string

	flag.StringVar(&inputPath, "i", "input.png", "输入的PNG文件路径")
	flag.StringVar(&outputName, "o", "output.png", "生成的PNG文件名")
	flag.StringVar(&icoName, "w", "app.ico", "生成的ICO文件名")
	flag.StringVar(&icnsName, "m", "AppIcon.icns", "生成的ICNS文件名")
	flag.Parse()

	// 读取输入的PNG文件
	srcImage, err := imaging.Open(inputPath)
	if err != nil {
		panic(err)
	}

	// *******   生成Linux hicolor  *******
	// 删除旧的 icons 目录
	err = os.RemoveAll("icons")
	if err != nil {
		fmt.Println("failed to remove path [icons]", err)
	}

	err = os.RemoveAll("pixmaps")
	if err != nil {
		fmt.Println("failed to remove path [pixmaps]", err)
	}

	err = os.MkdirAll("icons/hicolor", os.ModePerm)
	if err != nil {
		fmt.Println("failed to create path [icons/hicolor]", err)
	}

	err = os.MkdirAll("pixmaps", os.ModePerm)
	if err != nil {
		fmt.Println("failed to create path [pixmaps]", err)
	}

	// 将图像调整为不同的尺寸，并保存为PNG
	sizes := []int{16, 24, 32, 48, 64, 96, 128, 256, 512}
	for _, size := range sizes {
		sizeDir := filepath.Join("icons/hicolor", fmt.Sprintf("%dx%d", size, size), "apps")
		err = os.MkdirAll(sizeDir, os.ModePerm)
		if err != nil {
			fmt.Printf("failed to create %s, %v\n", sizeDir, err)
		} else {
			outputPath := filepath.Join(sizeDir, outputName)
			resizedImage := imaging.Resize(srcImage, size, size, imaging.Lanczos)
			err = imaging.Save(resizedImage, outputPath)
			if err != nil {
				panic(err)
			}
			if size == 128 {
				pixmapPath := filepath.Join("pixmaps", outputName)
				err = imaging.Save(resizedImage, pixmapPath)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	// *******   制作windows ico  *******
	iconPath := map[int]string{
		16:  "icons/hicolor/16x16/apps/",
		24:  "icons/hicolor/24x24/apps/",
		32:  "icons/hicolor/32x32/apps/",
		48:  "icons/hicolor/48x48/apps/",
		64:  "icons/hicolor/64x64/apps/",
		96:  "icons/hicolor/96x96/apps/",
		128: "icons/hicolor/128x128/apps/",
		256: "icons/hicolor/256x256/apps/",
	}

	iconDir := newIcondir(uint16(len(iconPath)))
	buf := new(bytes.Buffer)
	// 写ico的头
	binary.Write(buf, binary.LittleEndian, iconDir)

	var firstIteration bool = true
	var globalOffset uint32 = 0
	var pngAll = new(bytes.Buffer)
	for _, path := range iconPath {
		// 读取PNG图像文件
		pngFile, err := os.Open(filepath.Join(path, outputName))
		if err != nil {
			panic(err)
		}

		img, err := png.Decode(pngFile)
		if err != nil {
			panic(err)
		}

		err = pngFile.Close()
		if err != nil {
			panic(err)
		}

		ide := newIcondirentry()
		pngBuf := new(bytes.Buffer)
		pngWriter := bufio.NewWriter(pngBuf)
		png.Encode(pngWriter, img)
		pngWriter.Flush()
		ide.sizeInBytes = uint32(len(pngBuf.Bytes()))

		imgBounds := img.Bounds()
		ide.imageWidth = uint8(imgBounds.Dx())
		ide.imageHeight = uint8(imgBounds.Dy())
		if firstIteration {
			ide.offset = 6 + uint32(len(iconPath))*16
			firstIteration = false
		} else {
			ide.offset = globalOffset
		}
		globalOffset = ide.offset + ide.sizeInBytes
		binary.Write(buf, binary.LittleEndian, ide)
		pngAll.Write(pngBuf.Bytes())
	}

	outputFile, err := os.Create(icoName)
	if err != nil {
		panic(err)
	}

	outputFile.Write(buf.Bytes())
	outputFile.Write(pngAll.Bytes())
	err = outputFile.Close()
	if err != nil {
		panic(err)
	}

	// *******   制作macos icn  *******
	icnsFile, err := os.Create(icnsName)
	if err != nil {
		panic(err)
	}
	defer icnsFile.Close()
	if err := icns.Encode(icnsFile, srcImage); err != nil {
		fmt.Printf("encoding icns: %v", err)
	}
	fmt.Println("Conversion completed.")
}
