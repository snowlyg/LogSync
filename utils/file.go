package utils

import (
	"os"
)

func CreateFile(filePath string, b []byte) error {
	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(b)

	return nil
}

func OpenFile(filePath string) ([]byte, error) {
	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sourcebuffer := make([]byte, 500000)
	var n int
	n, err = f.Read(sourcebuffer)
	if err != nil {
		return nil, err
	}
	return sourcebuffer[:n], nil
}
