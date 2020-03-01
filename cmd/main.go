package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/tjan147/logparser"
)

const (
	targetNameFile = "../analyser/target.txt"

	defaultInput  = "../logs/cv2.log"
	defaultOutput = "../analyser/data"
)

var (
	input  = flag.String("i", defaultInput, "the path of the input log file")
	output = flag.String("o", defaultOutput, "the path of the output folder")

	filterDate = flag.String("date", "", "the date of selected log items")
)

func recordTargetName(target string) {
	if info, err := os.Stat(targetNameFile); !os.IsNotExist(err) {
		if !info.IsDir() {
			if err := os.Remove(targetNameFile); err != nil {
				panic(err)
			}
		}
	}

	if err := ioutil.WriteFile(targetNameFile, []byte(target), 0644); err != nil {
		panic(err)
	}
}

func main() {
	// gen filter using parameter
	flag.Parse()

	if len(*filterDate) > 0 {
		date, err := time.Parse("2006-01-02", *filterDate)
		if err != nil {
			fmt.Printf("by-date: error parsing date: %s", err.Error())
			os.Exit(1)
		}

		logparser.RegisterItemFilter(func(i logparser.Item) bool {
			iDate := i.Stamp()
			return (date.Day() == iDate.Day()) && (date.Month() == iDate.Month()) && (date.Year() == iDate.Year())
		})
		logparser.SetCurrentHeightStamp(date)

		fmt.Printf("%s as log item data filter added\n", *filterDate)
	}

	// parse as tendermint-like log
	logparser.RegisterTMPrefix()
	// parse the self-made benchmark log
	logparser.RegisterBSPrefix()

	fmt.Printf("start parsing %s ...\n", *input)
	res, cnt, err := logparser.ParseByLine(*input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d lines successfully parsed\n", cnt)

	datePart := ""
	if len(*filterDate) > 0 {
		datePart = "." + *filterDate
	}
	outName := filepath.Base(*input) + datePart
	recordTargetName(outName)

	fmt.Println("start exporting parsed data ...")
	for name, data := range res {
		outFile := path.Join(*output, outName+"."+name+".csv")

		if err := logparser.SaveAsCSV(outFile, data); err != nil {
			fmt.Printf("error exporting data to %s: %s\n", outFile, err.Error())
			continue
		}
		fmt.Printf("data successfully exported to %s\n", outFile)
	}

	fmt.Println("DONE")
}
