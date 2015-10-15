package main

import (
	"fmt"
	"strings"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Campaigns is the master type - a representation of multiple campaigns
// and representations of aggregate stats for said campaigns.
type Campaigns struct {
	StartDate string
	Meta      struct {
		Pagination struct {
			Next string `json:"next_link"`
		} `json:"pagination"`
	} `json:"meta"`
	Campaigns []*Campaign `json:"results"`
	Report    interface{} `json:",omitmepty"`
}

// Generates campaign report data for all of the child campaigns, then
// creates the master report data for the complete campaign window.
func (c *Campaigns) BuildCampaignReport() error {
	if len(c.Campaigns) == 0 {
		return fmt.Errorf("No campaigns to report on")
	}

	for _, campaign := range c.Campaigns {
		_ = campaign.BuildCampaignReport()
	}

	return c.buildMegaReport()
}

// send counts by domain (i.e. clicks by company)
// click counts by domain.  / % of clicks on a per-domain basis
// top link clicks
// unsubscribe list (farewell~)
// bounce list (ignoring suspended bounce reasons)
func (c *Campaigns) buildMegaReport() error {
	report := map[string]interface{}{}
	report["combined"] = combineStats(c.Campaigns)
	report["summaries"] = combineSummaries(c.Campaigns)
	report["clicks"] = combineClicks(c.Campaigns)
	report["unsubscribes"] = combineUnsubscribes(c.Campaigns)
	report["bounces"] = combineBounces(c.Campaigns)
	c.Report = report
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Campaign represents extended data for a single campaign
type Campaign struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	Status       string `json:"status"`
	ModifiedDate string `json:"modified_date"`
	RunDate      string `json:"last_run_date"`
	PermalinkUrl string `json:"permalink_url"`

	// Complex types from JSON response
	TrackingSummary campaignSummary   `json:"tracking_summary"`
	Clickthroughs   ClickList         `json:"click_through_details"`
	Tracking        []*trackingAction `json:",omitempty"`

	// Generated aggregates
	PivotedSummary   map[string]*campaignSummary
	Bounces          []string
	Unsubscribes     []string
	OrderedSummaries SummaryList
}

func (c *Campaign) RunDateAsTime() (time.Time, error) {
	return time.Parse(time.RFC3339, c.RunDate)
}

// Generates the following summary data.
// - DONE an ordered list of clickthroughs by popularity
// - DONE the sent -> opened -> clicked funnel (already in the TrackingSummary)
// - DONE a sent->opened->clicked funnel grouped by top 5 email address domains
// - DONE a list of unsubscribe email/uid
// - DONE a list of bounces
func (c *Campaign) BuildCampaignReport() error {
	c.PivotedSummary = map[string]*campaignSummary{}

	c.Tracking = deduplicateTracking(c.Tracking)

	// Tabluating bonces, unsubs, and per-domain summaries
	c.Bounces = []string{}
	c.Unsubscribes = []string{}
	for _, action := range c.Tracking {
		domain := action.getEmailDomain()
		if _, ok := c.PivotedSummary[domain]; !ok {
			c.PivotedSummary[domain] = &campaignSummary{
				Domain: domain,
			}
		}

		switch action.ActivityType {
		case "EMAIL_SEND":
			c.PivotedSummary[domain].Sends++
		case "EMAIL_OPEN":
			c.PivotedSummary[domain].Opens++
		case "EMAIL_CLICK":
			c.PivotedSummary[domain].Clicks++
		case "EMAIL_BOUNCE":
			c.PivotedSummary[domain].Bounces++
			c.Bounces = append(c.Bounces, action.Email)
		case "EMAIL_UNSUBSCRIBE":
			c.PivotedSummary[domain].Unsubscribes++
			c.Unsubscribes = append(c.Unsubscribes, action.Email)
		}
	}

	// Generating top email domains
	c.OrderedSummaries = aggTopDomains(getTopDomains, c.PivotedSummary)

	// Generating top Clicks report
	c.Clickthroughs = aggTopClicks(getTopClicks, c.Clickthroughs)

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// TrackingAction represents a click, open, send, etc to a user.
// actions are not necessarily unique per user per email.
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

func (t *trackingAction) getEmailDomain() string {
	return strings.Split(t.Email, "@")[1]
}

////////////////////////////////////////////////////////////////////////////////
// campaignSummary represents the total aggregate actions performed by users
// against a campaign.  The nonstandard "Domain" key has been added to permit
// pivoting this field on the basis of user email address domains.
type campaignSummary struct {
	Domain       string
	Sends        int `json:"sends"`
	Opens        int `json:"opens"`
	Clicks       int `json:"clicks"`
	Forwards     int `json:"forwards"`
	Unsubscribes int `json:"unsubscribes"`
	Bounces      int `json:"bounces"`
	Spam_count   int `json:"spam_count"`
}

func (s *campaignSummary) Add(s2 *campaignSummary) {
	s.Sends += s2.Sends
	s.Opens += s2.Opens
	s.Clicks += s2.Clicks
	s.Forwards += s2.Forwards
	s.Unsubscribes += s2.Unsubscribes
	s.Bounces += s2.Bounces
	s.Spam_count += s2.Spam_count
}

type click struct {
	ID     string `json:"url_uid"`
	URL    string `json:"url"`
	Clicks int    `json:"click_count"`
}

// SummaryList Implements sort.Inteface in order to sort by summary.Sends
type SummaryList []*campaignSummary

func (s SummaryList) Len() int           { return len(s) }
func (s SummaryList) Less(i, j int) bool { return s[i].Sends > s[j].Sends }
func (s SummaryList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// ClickList Implements sort.Inteface in order to sort by summary.Sends
type ClickList []*click

func (c ClickList) Len() int           { return len(c) }
func (c ClickList) Less(i, j int) bool { return c[i].Clicks > c[j].Clicks }
func (c ClickList) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
