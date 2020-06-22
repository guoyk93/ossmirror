package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

type Conf struct {
	Workspace          string `json:"workspace"`
	OSSPublicURL       string `json:"oss_public_url"`
	OSSEndpoint        string `json:"oss_endpoint"`
	OSSAccessKeyID     string `json:"oss_access_key_id"`
	OSSAccessKeySecret string `json:"oss_access_key_secret"`
	OSSBucket          string `json:"oss_bucket"`
}

var (
	optConf string
	optURL  string
)

func main() {
	var err error
	defer exit(&err)

	flag.StringVar(&optConf, "c", "/etc/ossmirror/config.json", "configuration file")
	flag.StringVar(&optURL, "l", "", "url to download")
	flag.Parse()

	optURL = strings.TrimSpace(optURL)

	if optURL == "" {
		err = errors.New("invalid url")
		return
	}

	var buf []byte
	if buf, err = ioutil.ReadFile(optConf); err != nil {
		return
	}

	var conf Conf
	if err = json.Unmarshal(buf, &conf); err != nil {
		return
	}

	log.Println("oss go sdk version: ", oss.Version)

	var client *oss.Client
	if client, err = oss.New(conf.OSSEndpoint, conf.OSSAccessKeyID, conf.OSSAccessKeySecret); err != nil {
		return
	}

	log.Println("bucket:", conf.OSSBucket)

	var bucket *oss.Bucket
	if bucket, err = client.Bucket(conf.OSSBucket); err != nil {
		return
	}

	log.Println("url:", optURL)

	var uri *url.URL
	if uri, err = url.Parse(optURL); err != nil {
		return
	}

	filename := filepath.Join(conf.Workspace, path.Base(uri.Path))
	log.Println("local file:", filename)

	var fe bool
	if fe, err = fileExists(filename); err != nil {
		return
	}

	if !fe {
		defer os.Remove(filename)

		var f *os.File
		if f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0640); err != nil {
			return
		}
		defer f.Close()

		var res *http.Response
		if res, err = http.Get(optURL); err != nil {
			return
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("bad code: %d", res.StatusCode)
			return
		}

		if _, err = io.Copy(f, res.Body); err != nil {
			return
		}
	}

	key := path.Join(uri.Host, uri.Path)
	log.Println("remote file:", strings.TrimSuffix(conf.OSSPublicURL, "/")+"/"+strings.TrimPrefix(key, "/"))
	if err = bucket.PutObjectFromFile(key, filename); err != nil {
		return
	}

	log.Println("done")
}

func fileExists(filename string) (ok bool, err error) {
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return
	}
	ok = true
	return
}
