package lib

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
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

func RemoveFromList(list []string, itemToRemove string) []string {
	newList := []string{}
	for _, i := range list {
		if i != itemToRemove {
			newList = append(newList, i)
		}
	}
	return newList
}

func ParseURL(rawurl string) (domain string, scheme string, err error) {
	u, err := url.ParseRequestURI(rawurl)
	if err != nil || u.Host == "" {
		u, repErr := url.ParseRequestURI("https://" + rawurl)
		if repErr != nil {
			fmt.Printf("Could not parse url: %s, error: %v", rawurl, err)
			return
		}
		domain = u.Host
		err = nil
		return
	}

	domain = u.Host
	scheme = u.Scheme
	return
}

func CheckHttp2xx(url string, timeout int) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	return res.StatusCode >= 200 && res.StatusCode <= 299
}

func CheckHttpRespRange(start, end int, url string, timeout int) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	return res.StatusCode >= start && res.StatusCode <= end
}

func IsValidLabelName(lblName string) bool {

	regex := `^[a-zA-Z_][a-zA-Z0-9_]*$`
	match, _ := regexp.MatchString(regex, lblName)
	if match {
		return true
	}
	return false

}

func IsValidTargetName(targetName string) bool {

	regex := `^(([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])\.)*([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])(:[0-9]+)?$`
	match, _ := regexp.MatchString(regex, targetName)
	if match {
		return true
	}
	return false

}
