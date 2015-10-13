package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rakyll/globalconf"
)

type apiPagination struct {
	Prev string `json:"prev_link,omitempty"`
	Next string `json:"next_link,omitempty"`
}

type apiMeta struct {
	Pagination apiPagination `json:"pagination,omitempty"`
}

type Tracking struct {
	Meta    apiMeta           `json:"meta"`
	Results []*trackingAction `json:"results"`
}

type Campaigns struct {
	Meta struct {
		Pagination struct {
			Next string `json:"next_link"`
		} `json:"pagination"`
	} `json:"meta"`
	Campaigns []*Campaign `json:"results"`
}

type campaignSummary struct {
	Sends        int `json:"sends"`
	Opens        int `json:"opens"`
	Clicks       int `json:"clicks"`
	Forwards     int `json:"forwards"`
	Unsubscribes int `json:"unsubscribes"`
	Bounces      int `json:"bounces"`
	Spam_count   int `json:"spam_count"`
}

type click struct {
	ID     string `json:"url_uid"`
	URL    string `json:"url"`
	Clicks int    `json:"click_count"`
}

type trackingAction struct {
	// Common
	ActivityType string `json:"activity_type"`
	ContactID    string `json:"contact_id"`
	Email        string `json:"email_address"`

	// Clicks
	LinkId    string `json:"link_id"`
	ClickDate string `json:"click_date"`

	// Opens
	OpenDate string `json:"open_date"`

	// Sends
	SendDate string `json:"send_date"`

	// Unsubscribes
	UnsubDate   string `json:"unsubscribe_date"`
	UnsubSource string `json:"unsubscribe_source"`
	UnsubReason string `json:"unsubscribe_reason"`

	// Bounces
	BounceCode string `json:"bounce_code"`
	BounceDesc string `json:"bounce_description"`
	BounceMsg  string `json:"bounce_message"`
	BounceDate string `json:"bounce_date"`
}

type Campaign struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	Status       string `json:"status"`
	ModifiedDate string `json:"modified_date"`
	RunDate      string `json:"last_run_date"`
	PermalinkUrl string `json:"permalink_url"`

	TrackingSummary campaignSummary `json:"tracking_summary"`

	Clickthroughs []click `json:"click_through_details"`

	Tracking []*trackingAction `json:",omitempty"`
}

func (c *Campaign) RunDateAsTime() (time.Time, error) {
	return time.Parse(time.RFC3339, c.RunDate)
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
		trackingResp := &Tracking{
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

func save(c Campaigns, filename string) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(c)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(filename, b.Bytes(), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func load(filename string) Campaigns {
	b, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	dec := gob.NewDecoder(b)

	out := Campaigns{}
	err = dec.Decode(&out)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

var (
	config    string
	authToken string
	apiKey    string
)

func init() {
	flag.StringVar(&config, "config", "config.ini", "config filename")

	flag.StringVar(&authToken, "token", "", "Bearer token for authentication")
	flag.StringVar(&apiKey, "key", "", "API Key")
}

func main() {
	// TODO: aggregate the retrieved data
	// encode and decode the retreived data for storage using gob
	// create a template to render the data

	flag.Parse()
	conf, err := globalconf.NewWithOptions(&globalconf.Options{
		Filename: config,
	})
	if err != nil {
		log.Fatal(err)
	}
	conf.ParseAll()

	camps := load("heh.txt")

	fmt.Println(camps)
	for _, campaign := range camps.Campaigns {
		fmt.Println(campaign)
	}

	/*
		camps, err := getCampaigns()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(camps)
		fmt.Println("retrieved campaigns: ", len(camps.Campaigns))

		for _, c := range camps.Campaigns {
			err = getCampaignDetail(c)
			fmt.Println(err)
			fmt.Println(c)

			err = getCampaignTracking(c)

			b, _ := json.MarshalIndent(c, "", "  ")
			fmt.Println(string(b))
		}
		save(camps, "heh.txt")
	*/
}
