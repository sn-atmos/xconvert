package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/nkzk/xconvert"

	apiextv2 "github.com/crossplane/crossplane/apis/v2/apiextensions/v2"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	data, err := os.ReadFile("xrds/basic.yaml")
	if err != nil {
		log.Fatal(err)
	}

	xrd := &apiextv2.CompositeResourceDefinition{}
	if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(data)).Decode(xrd); err != nil {
		log.Fatal(err)
	}

	doc, err := xconvert.XRDToOpenAPI(xrd)
	if err != nil {
		log.Fatal(err)
	}

	out, err := json.MarshalIndent(doc, "", "    ")
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
