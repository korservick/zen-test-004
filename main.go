package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	wex "github.com/onuryilmaz/go-wex"
)

type metric struct {
	value    [maxPosition]float64
	position int
	time     int64
}

type calc struct {
	Avg     float64
	Updated int64
}

const maxPosition = 600 // aggregation period in sec

var (
	api           wex.API
	pairs         []string
	ignoreInvalid = true
	tickAccum     *time.Ticker
	metrics       = make(map[string]metric)
	avgSync       = sync.RWMutex{}
	// avg           = make(map[string]float64)
	pairsCalc = make(map[string]calc)
	pairCalc  calc
	mapB      []byte
)

func mod(a, b int) int {
	return a - (b * int(math.Floor(float64(a)/float64(b))))
}

func accum() {
	for range tickAccum.C {
		ticker, err := api.Public.Ticker(pairs, ignoreInvalid)
		if err == nil {
			now := time.Now().Unix()
			for k, v := range ticker {
				m := metrics[k]
				m.value[m.position] = v.Last
				m.position = mod(m.position+1, maxPosition)
				m.time = now
				sum := 0.0
				metrics[k] = m
				for _, v := range m.value {
					sum += v
				}
				a := sum / maxPosition
				pairCalc.Avg = a
				pairCalc.Updated = now
				// avgSync.Lock()
				pairsCalc[k] = pairCalc
				// avg[k] = a
				// avgSync.Unlock()
				// if k == "btc_usd" {
				// 	fmt.Printf("pair:%s t:%d v:%f avg:%v \n", k, now, v.Last, sum/maxPosition)
				// }
			}
			avgSync.Lock()
			mapB, _ = json.Marshal(pairsCalc)
			// if err != nil {
			// fmt.Printf("Error: %s\n", err)
			// }
			avgSync.Unlock()
		}
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./favicon.ico")
}

func handler(w http.ResponseWriter, r *http.Request) {
	// log.Println("Received Request: ", r.URL.Path)
	// fmt.Fprintf(w, "Hello World!\n")
	avgSync.RLock()
	out := mapB
	avgSync.RUnlock()
	fmt.Fprintf(w, "%s", out)
}

func main() {

	info, err := api.Public.Info()
	if err == nil {
		for pairName := range info.Pairs {
			pairs = append(pairs, pairName)
		}
		// fmt.Printf("Pair: %v\n", pairs)
	}

	tickAccum = time.NewTicker(1000 * time.Millisecond)
	go accum()

	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/", handler)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable was not set")
	}
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Could not listen: ", err)
	}
}
