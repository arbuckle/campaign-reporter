package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rakyll/globalconf"

	"github.com/arbuckle/campaign-reporter/pkg/api"
	"github.com/arbuckle/campaign-reporter/pkg/types"
)

var (
	config    string
	authToken string
	apiKey    string

	getTopDomains int
	getTopClicks  int
	daysBack      int

	prevReport string

	debug bool
)

func init() {
	flag.StringVar(&config, "config", "config.ini", "config filename")

	flag.StringVar(&authToken, "cc.token", "", "Bearer token for authentication")
	flag.StringVar(&apiKey, "cc.key", "", "API Key")

	flag.StringVar(&prevReport, "run.fromfile", "", "Generate report from stored file")
	flag.BoolVar(&debug, "run.debug", false, "verbose output")
	flag.IntVar(&getTopDomains, "run.domains", 0, "Number of email domains to display")
	flag.IntVar(&daysBack, "run.daysback", 0, "Days back to look for data")
	flag.IntVar(&getTopClicks, "run.links", 0, "Number of links to display ")
}

// TODO:  html template
// TODO:  email this out
// TODO:  better comments
func main() {
	// loading configuration
	flag.Parse()
	conf, err := globalconf.NewWithOptions(&globalconf.Options{
		Filename: config,
	})
	if err != nil {
		log.Fatal(err)
	}
	conf.ParseAll()

	ccAPI, _ := api.NewAPI(config, authToken, apiKey, daysBack, debug)

	// Generate a report from a stored file
	if prevReport != "" {
		c := types.Load(prevReport)
		c.BuildCampaignReport(getTopDomains, getTopClicks)
		fmt.Println(types.Render(c))
		os.Exit(0)
	}

	runTime := time.Now().Format("2006-01-02T15:04")
	saveTo := fmt.Sprintf("./logs/%s.gob", runTime)

	log.Print(runTime, saveTo)

	camps, err := ccAPI.GetCampaigns()
	if err != nil {
		log.Fatal(err)
	}
	log.Print(camps)
	log.Print("retrieved campaigns: ", len(camps.Campaigns))

	for _, c := range camps.Campaigns {
		err = ccAPI.GetCampaignDetail(c)
		log.Print(err)
		log.Print(c)

		err = ccAPI.GetCampaignPreview(c)
		log.Print(err)

		err = ccAPI.GetCampaignTracking(c)

		b, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(b))
	}
	types.Save(camps, saveTo)
	camps.BuildCampaignReport(getTopDomains, getTopClicks)
	fmt.Println(camps)
	fmt.Println(types.Render(camps))
}
