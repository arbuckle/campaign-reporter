package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"text/template"
)

// generates a template
func render(c Campaigns) string {
	funcs := template.FuncMap{"perc": func(a, b int) int {
		if a == 0 {
			return 0
		}
		return int((float64(a) / float64(b)) * 100)
	}}

	t, _ := template.New("email").Funcs(funcs).ParseFiles("email.template")
	log.Print(t)

	b := bytes.NewBuffer([]byte{})
	t.ExecuteTemplate(b, "email", c)
	return b.String()
}

// gob-encodes the input and saves to filename, killing the program if an error is
// encountered
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

// loads a gob-encoded input from file.
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

func combineStats(c []*Campaign) *campaignSummary {
	out := &campaignSummary{}
	for _, campaign := range c {
		out.Add(&campaign.TrackingSummary)
	}
	return out
}

func combineSummaries(c []*Campaign) SummaryList {
	pivoted := map[string]*campaignSummary{}
	for _, campaign := range c {
		for _, s := range campaign.PivotedSummary {
			if _, ok := pivoted[s.Domain]; !ok {
				pivoted[s.Domain] = s
				continue
			}
			pivoted[s.Domain].Add(s)
		}
	}
	return aggTopDomains(getTopDomains, pivoted)
}

func combineClicks(c []*Campaign) ClickList {
	links := map[string]*click{}
	out := ClickList{}
	for _, campaign := range c {
		for _, click := range campaign.Clickthroughs {
			if _, ok := links[click.ID]; !ok {
				links[click.ID] = click
			} else {
				links[click.ID].Clicks += click.Clicks
			}

		}
	}
	for _, click := range links {
		if click.Clicks > 0 {
			out = append(out, click)
		}
	}
	sort.Sort(out)
	return out
}

func combineUnsubscribes(c []*Campaign) []string {
	out := []string{}
	for _, campaign := range c {
		out = append(out, campaign.Unsubscribes...)
	}
	return out
}

func combineBounces(c []*Campaign) []string {
	out := []string{}
	dedup := map[string]bool{}
	for _, campaign := range c {
		for _, email := range campaign.Bounces {
			if _, ok := dedup[email]; !ok {
				out = append(out, email)
				dedup[email] = true
			}
		}
	}
	return out
}

func deduplicateTracking(t []*trackingAction) []*trackingAction {
	out := []*trackingAction{}
	seen := map[string]map[string]bool{}
	for _, action := range t {
		activity := action.ActivityType
		user := action.ContactID
		if _, ok := seen[activity]; !ok {
			seen[activity] = map[string]bool{}
		}
		if _, ok := seen[activity][user]; !ok {
			seen[activity][user] = true
			out = append(out, action)
		}
	}
	return out
}

// takes a map[string]campaignSummary, orders it by domain in a slice,
// and extracts the top N versions
func aggTopDomains(numDomains int, pivotedSummary map[string]*campaignSummary) SummaryList {
	// Preparing and normalizing the per-domain summary into an ordered summary and extracting
	// the top N domains, depositing all other reports into "other..."

	orderedSummaries := SummaryList{}
	otherSummaries := &campaignSummary{
		Domain: "other...",
	}

	for _, summary := range pivotedSummary {
		if summary.Domain == "other..." {
			otherSummaries.Add(summary)
			continue
		}
		orderedSummaries = append(orderedSummaries, summary)
	}

	newOs := SummaryList{}
	sort.Sort(orderedSummaries)
	for i, s := range orderedSummaries {
		if i <= numDomains {
			newOs = append(newOs, s)
			log.Print("top domain: ", s)
		} else {
			otherSummaries.Add(s)
		}
	}
	newOs[numDomains] = otherSummaries
	return newOs
}

// sorts a ClickList by top clicks and outputs a new, condensed clicklist.
// has the side effect of printing, reordering clicks list
func aggTopClicks(numClicks int, c ClickList) ClickList {
	newClicks := ClickList{}
	sort.Sort(c)
	for i, click := range c {
		if i <= numClicks {
			log.Print("top click: ", click)
			newClicks = append(newClicks, click)
			continue
		}
		break
	}
	return newClicks
}
