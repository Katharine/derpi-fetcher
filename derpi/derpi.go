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

package derpi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type searchResult struct {
	Search []json.RawMessage `json:"images"`
}

func urlForQuery(query string, filterID, page int) string {
	v := url.Values{
		"q":         []string{query},
		"per_page":  []string{"50"},
		"page":      []string{strconv.Itoa(page)},
		"filter_id": []string{strconv.Itoa(filterID)},
	}
	u := url.URL{
		Scheme:   "https",
		Host:     "derpibooru.org",
		Path:     "/api/v1/json/search/images",
		RawQuery: v.Encode(),
	}
	return u.String()
}

func Search(ctx context.Context, search string, filterID int) chan json.RawMessage {
	ch := make(chan json.RawMessage, 10)
	go func() {
		defer func() {
			close(ch)
		}()
		page := 1
		retries := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			resp, err := http.Get(urlForQuery(search, filterID, page))
			if err != nil {
				log.Printf("Request failed! %v\n", err)
				return
			}
			r := searchResult{}
			d := json.NewDecoder(resp.Body)
			if err := d.Decode(&r); err != nil {
				log.Printf("JSON decoding failed: %v\n", err)
				if retries < 5 {
					log.Println("Retrying...")
					retries += 1
					time.Sleep(5 * time.Second)
					continue
				}
				log.Println("Retries exhausted, giving up.")
				return
			}
			retries = 0
			if len(r.Search) == 0 {
				return
			}
			page += 1
			for _, image := range r.Search {
				select {
				case <-ctx.Done():
					return
				default:
				}
				ch <- image
			}
		}
	}()
	return ch
}
