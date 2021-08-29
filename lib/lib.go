package lib

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GetFileContents(path string) (*[]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("No target store exists")
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return &bytes, nil
}

func CreateFile(path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}

func Contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}
