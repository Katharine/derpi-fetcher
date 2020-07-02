// Copyright 2019 Katharine Berry
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fetcher/derpi"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func ext(p string) string {
	e := strings.ToLower(path.Ext(p))
	if e == ".jpeg" {
		e = ".jpg"
	}
	return e
}

type representations struct {
	Full string `json:"full"`
}

type imageData struct {
	ID              int             `json:"id"`
	Representations representations `json:"representations"`
	Tags            []string        `json:"tags"`
}

func getArtistFromTags(tags []string) string {
	var artists []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "artist:") {
			artists = append(artists, strings.ReplaceAll(strings.SplitN(tag, ":", 2)[1], "/", "_"))
		}
	}
	if len(artists) > 0 {
		sort.Strings(artists)
		s := strings.Join(artists, "-&-")
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	}
	return "unknown"
}

func saver(images <-chan json.RawMessage, counter chan<- struct{}) {
	for image := range images {
		retries := 0
		for {
			success := false
			func() {
				var d imageData
				if err := json.Unmarshal(image, &d); err != nil {
					log.Printf("Couldn't decode image data: %v.\n")
					return
				}
				url := d.Representations.Full

				dir := getArtistFromTags(d.Tags)
				p := dir + "/" + strconv.Itoa(d.ID) + ext(url)
				if _, err := os.Stat(p); err == nil {
					return
				}

				resp, err := http.Get(url)
				if err != nil {
					log.Printf("Failed to fetch %s: %v\n", url, err)
					return
				}
				defer resp.Body.Close()

				_ = os.Mkdir(dir, 0777)
				out, err := os.Create(p)
				if err != nil {
					log.Printf("failed to create output file: %v\n", err)
					return
				}
				defer out.Close()
				if _, err := io.Copy(out, resp.Body); err != nil {
					_ = os.Remove(p)
					log.Printf("failed to copy %s from internet to file: %v\n", url, err)
					return
				}
				success = true
				if err := ioutil.WriteFile(dir+"/"+strconv.Itoa(d.ID)+".json", image, 0644); err != nil {
					log.Printf("Failed to write metadata file: %v.\n", err)
				}
				if counter != nil {
					counter <- struct{}{}
				}
			}()
			if success {
				break
			}
			retries += 1
			if retries > 5 {
				break
			}
			log.Println("Retrying...")
			time.Sleep(1 * time.Second)
		}
	}
}

func asyncSaver(images <-chan json.RawMessage, counter chan<- struct{}, done chan<- struct{}) {
	go func() {
		saver(images, counter)
		done <- struct{}{}
	}()
}

func manySavers(count int, images <-chan json.RawMessage, counter chan<- struct{}) chan struct{} {
	done := make(chan struct{})
	go func() {
		ch := make(chan struct{})
		for i := 0; i < count; i++ {
			asyncSaver(images, counter, ch)
		}
		for i := 0; i < count; i++ {
			<-ch
		}
		done <- struct{}{}
	}()
	return done
}

type config struct {
	FilterID int
	Workers  int
	Query    string
}

func getConfig() config {
	c := config{}
	flag.IntVar(&c.FilterID, "filter-id", 56027, "filter ID to use (defaults to 'Everything', 100073 is 'Default')")
	flag.IntVar(&c.Workers, "workers", 100, "number of concurrent downloads")
	flag.Usage = func() {
		fmt.Printf("%s [flags] <query>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	c.Query = flag.Arg(0)
	return c
}

func main() {
	c := getConfig()
	fmt.Printf("Searching for \"%s\"...\n", c.Query)
	images := derpi.Search(context.Background(), c.Query, c.FilterID)
	counter := make(chan struct{}, 10)
	done := manySavers(c.Workers, images, counter)
	count := 0
	for {
		select {
		case <-done:
			close(counter)
			for range counter {
				count++
			}
			log.Printf("Done! Downloaded %d images.\n", count)
			return
		case <-counter:
			count++
			if count%100 == 0 {
				log.Printf("Downloaded %d images.\n", count)
			}
		}
	}
}
