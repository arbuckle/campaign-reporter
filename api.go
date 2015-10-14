package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type apiPagination struct {
	Prev string `json:"prev_link,omitempty"`
	Next string `json:"next_link,omitempty"`
}

type apiMeta struct {
	Pagination apiPagination `json:"pagination,omitempty"`
}

type apiTracking struct {
	Meta    apiMeta           `json:"meta"`
	Results []*trackingAction `json:"results"`
}

func getURLAndDecodeInto(url string, i interface{}) error {
	log.Print("opening ", url)

	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)
	auth := fmt.Sprintf("Bearer %s", authToken)
	req.Header.Add("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//b, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(b))
	//err = json.Unmarshal(b, &i)

	err = json.NewDecoder(resp.Body).Decode(&i)
	return err
}

// Retrieve all campaigns for the last 7 days
func getCampaigns() (Campaigns, error) {
	now := time.Now().Add(-336 * time.Hour).Format(time.RFC3339)
	url := fmt.Sprintf("https://api.constantcontact.com/v2/emailmarketing/campaigns?status=ALL&api_key=%s&modified_since=%s", apiKey, now)

	c := &Campaigns{}
	err := getURLAndDecodeInto(url, c)
	return *c, err
}

// Gets extended metadata for a single campaign
func getCampaignDetail(c *Campaign) error {
	url := fmt.Sprintf("https://api.constantcontact.com/v2/emailmarketing/campaigns/%s?api_key=%s", c.ID, apiKey)
	return getURLAndDecodeInto(url, c)

}

// Pull tracking info for Sends, Opens, Clicks, Bounces, and Unsubs
func getCampaignTracking(c *Campaign) error {
	c.Tracking = []*trackingAction{}
	for _, t := range []string{"sends", "opens", "clicks", "bounces", "unsubscribes"} {
		url := fmt.Sprintf("/v2/emailmarketing/campaigns/%s/tracking/%s?api_key=%s", c.ID, t, apiKey)
		trackingResp := &apiTracking{
			Meta: apiMeta{
				Pagination: apiPagination{
					Next: url,
				},
			},
		}
		var oldUrl string

		// Recursion doesn't really make sense here, so instead making successive calls to
		for trackingResp.Meta.Pagination.Next != "" {
			url = fmt.Sprintf("https://api.constantcontact.com%s&api_key=%s", trackingResp.Meta.Pagination.Next, apiKey)
			if url == oldUrl {
				break
			}
			err := getURLAndDecodeInto(url, trackingResp)
			fmt.Println(err)
			if err != nil {
				return err
			}
			for _, result := range trackingResp.Results {
				log.Print("Appending result ", result)
				c.Tracking = append(c.Tracking, result)
			}
			oldUrl = url
		}
	}
	return nil
}
