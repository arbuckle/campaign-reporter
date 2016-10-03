package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/arbuckle/campaign-reporter/pkg/types"
)

type API struct {
	config    string
	authToken string
	apiKey    string

	daysBack int

	debug bool
}

func NewAPI(cfg, token, key string, db int, d bool) (*API, error) {
	a := &API{}
	a.config = cfg
	a.authToken = token
	a.apiKey = key
	a.daysBack = db
	a.debug = d
	return a, nil
}

type List struct {
	Id           int    `json:"id"`
	CreatedDate  string `json:"created_date"`
	ModifiedDate string `json:"modified_date"`

	ContactCount int    `json:"contact_count"`
	Name         string `json:"name"`
	Status       string `json:"status"`
}

func (a *API) GetLists() ([]*List, error) {
	now := time.Now().Add(-time.Duration(a.daysBack*24) * time.Hour).Format(time.RFC3339)
	url := fmt.Sprintf("https://api.constantcontact.com/v2/lists?api_key=%s&modified_since=%s", a.apiKey, now)

	l := []*List{}
	err := a.getURLAndDecodeInto(url, l)
	return l, err
}

// Retrieve all campaigns for the last 7 days
func (a *API) GetCampaigns() (types.Campaigns, error) {
	now := time.Now().Add(-time.Duration(a.daysBack*24) * time.Hour).Format(time.RFC3339)
	url := fmt.Sprintf("https://api.constantcontact.com/v2/emailmarketing/campaigns?status=ALL&api_key=%s&modified_since=%s", a.apiKey, now)

	c := &types.Campaigns{
		DaysBack: a.daysBack,
	}
	err := a.getURLAndDecodeInto(url, c)
	return *c, err
}

// Gets extended metadata for a single campaign
func (a *API) GetCampaignDetail(c *types.Campaign) error {
	url := fmt.Sprintf("https://api.constantcontact.com/v2/emailmarketing/campaigns/%s?api_key=%s", c.ID, a.apiKey)
	return a.getURLAndDecodeInto(url, c)

}

func (a *API) GetCampaignPreview(c *types.Campaign) error {
	url := fmt.Sprintf("https://api.constantcontact.com/v2/emailmarketing/campaigns/%s/preview?api_key=%s", c.ID, a.apiKey)
	return a.getURLAndDecodeInto(url, c)
}

// Pull tracking info for Sends, Opens, Clicks, Bounces, and Unsubs
func (a *API) GetCampaignTracking(c *types.Campaign) error {
	c.Tracking = []*types.TrackingAction{}
	for _, t := range []string{"sends", "opens", "clicks", "bounces", "unsubscribes"} {
		url := fmt.Sprintf("/v2/emailmarketing/campaigns/%s/tracking/%s?api_key=%s", c.ID, t, a.apiKey)
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
			url = fmt.Sprintf("https://api.constantcontact.com%s&api_key=%s", trackingResp.Meta.Pagination.Next, a.apiKey)
			if url == oldUrl {
				break
			}
			err := a.getURLAndDecodeInto(url, trackingResp)
			log.Print(err)
			if err != nil {
				return err
			}
			for _, result := range trackingResp.Results {
				log.Printf("Appending result %s", result)
				c.Tracking = append(c.Tracking, result)
			}
			oldUrl = url
		}
	}
	return nil
}

type apiPagination struct {
	Prev string `json:"prev_link,omitempty"`
	Next string `json:"next_link,omitempty"`
}

type apiMeta struct {
	Pagination apiPagination `json:"pagination,omitempty"`
}

type apiTracking struct {
	Meta    apiMeta                 `json:"meta"`
	Results []*types.TrackingAction `json:"results"`
}

func (a *API) getURLAndDecodeInto(url string, i interface{}) error {
	log.Printf("opening %s", url)

	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)
	auth := fmt.Sprintf("Bearer %s", a.authToken)
	req.Header.Add("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if a.debug {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Print(string(b))
		err = json.Unmarshal(b, &i)
	} else {
		err = json.NewDecoder(resp.Body).Decode(&i)
		return err
	}
	return nil
}
