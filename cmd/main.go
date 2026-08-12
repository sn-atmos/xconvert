package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/sn-atmos/xconvert"
)

func main() {
	data, err := os.ReadFile("xrds/basic.yaml")
	if err != nil {
		log.Fatal(err)
	}

	xrds, err := xconvert.LoadXRD(data)
	if err != nil {
		log.Fatal(err)
	}

	docs, err := xconvert.XRDsToOpenAPI(xrds)
	if err != nil {
		log.Fatal(err)
	}

	out, err := json.MarshalIndent(docs, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll("out", 0o755)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("out/basic.json", out, 0o644)
	if err != nil {
		log.Fatal(err)
	}
}
