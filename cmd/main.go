package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/tjan147/logparser"
)

const (
	defaultInput  = "../logs/cv2.log"
	defaultOutput = "../analyser/data"
)

var (
	input  = flag.String("i", defaultInput, "the path of the input log file")
	output = flag.String("o", defaultOutput, "the path of the output folder")

	filterDate = flag.String("by_date", "", "the date of selected log items")
	// filterTime = flag.String("by_time", "", "the time of selected log items")
)

func main() {
	// gen filter using parameter
	if len(*filterDate) > 0 {
		date, err := time.Parse("2006-01-02", *filterDate)
		if err != nil {
			fmt.Printf("by_date: error parsing date: %s", err.Error())
			os.Exit(1)
		}

		logparser.RegisterItemFilter(func(i logparser.Item) bool {
			iDate := i.Stamp()
			return (date.Day() == iDate.Day()) && (date.Month() == iDate.Month()) && (date.Year() == iDate.Year())
		})
	}

	// TODO: add time filter

	// parse as tendermint-like log
	logparser.RegisterTMPrefix()

	fmt.Printf("start parsing %s ...\n", *input)
	res, cnt, err := logparser.ParseByLine(*input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d lines successfully parsed\n", cnt)

	fmt.Println("start exporting parsed data ...")
	for name, data := range res {
		outFile := path.Join(*output, name+".csv")

		if err := logparser.SaveAsCSV(outFile, data); err != nil {
			fmt.Printf("error exporting data to %s: %s\n", outFile, err.Error())
			continue
		}
		fmt.Printf("data successfully exported to %s\n", outFile)
	}

	fmt.Println("DONE")
}
