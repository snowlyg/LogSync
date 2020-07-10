package utils

import (
	"fmt"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func ResizePng(imgPath, imgName string) error {
	file, err := os.Open(imgPath)
	if err != nil {
		fmt.Println("文件打开失败", err)
		return err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		fmt.Println("文件解码失败", err)
		return err
	}

	m := resize.Resize(800, 0, img, resize.Lanczos3)
	out, err := os.Create(imgName)
	if err != nil {
		fmt.Println("文件新建失败", err)
		return err
	}
	defer out.Close()

	// write new image to file
	png.Encode(out, m)

	return nil
}
