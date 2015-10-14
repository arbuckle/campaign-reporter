package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"

	"github.com/rakyll/globalconf"
)

var getTopDomains int = 10
var getTopClicks int = 5

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

// generates a template
func render(c Campaigns) string {
	funcs := template.FuncMap{"perc": func(a, b int) int {
		if a == 0 {
			return 0
		}
		return int((float64(a) / float64(b)) * 100)
	}}

	t, _ := template.New("email").Funcs(funcs).ParseFiles("email.template")
	fmt.Println(t)

	b := bytes.NewBuffer([]byte{})
	t.ExecuteTemplate(b, "email", c)
	return b.String()
}

func main() {
	// TODO: aggregate the retrieved data
	// encode and decode the retreived data for storage using gob
	// create a template to render the data
	// TODO:  store runs in a normal form
	// TODO:  html template
	// TODO:  email this out
	// TODO:  better comments

	flag.Parse()
	conf, err := globalconf.NewWithOptions(&globalconf.Options{
		Filename: config,
	})
	if err != nil {
		log.Fatal(err)
	}
	conf.ParseAll()

	camps := load("./logs/heh.gob")
	camps.BuildCampaignReport()

	s := render(camps)
	fmt.Println(s)

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
		save(camps, "./logs/heh.gob")
	*/
}
