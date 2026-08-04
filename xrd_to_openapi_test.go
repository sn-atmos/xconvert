package xconvert_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/nkzk/xconvert"

	apiextv2 "github.com/crossplane/crossplane/apis/v2/apiextensions/v2"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-openapi/pkg/spec3"
)

//go:embed xrds/basic.yaml
var basicXRD []byte

func TestConvertBasicXRD(t *testing.T) {
	xrd := decodeBasicXRD(t)

	generated, err := xconvert.XRDToOpenAPI(xrd)
	if err != nil {
		t.Fatal(err)
	}

	generatedData, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	generated = &spec3.OpenAPI{}
	if err := json.Unmarshal(generatedData, generated); err != nil {
		t.Fatal(err)
	}

	if generated.Version != xconvert.OpenAPIVersion {
		t.Fatalf("expected OpenAPI version %q, got %q", xconvert.OpenAPIVersion, generated.Version)
	}

	schema, ok := generated.Components.Schemas["NetworkingStack"]
	if !ok {
		t.Fatal("expected NetworkingStack schema")
	}

	metadata, ok := schema.Properties["metadata"]
	if !ok {
		t.Fatal("expected metadata schema")
	}

	if !metadata.Properties["name"].Type.Contains("string") {
		t.Fatalf("expected metadata.name to be string, got %q", metadata.Properties["name"].Type)
	}

	labels := metadata.Properties["labels"].AdditionalProperties
	if labels == nil || labels.Schema == nil || !labels.Schema.Type.Contains("string") {
		t.Fatal("expected metadata.labels to be a string map")
	}
}

func TestConvertAllowsOmittedSchema(t *testing.T) {
	xrd := &apiextv2.CompositeResourceDefinition{}
	xrd.Spec.Names.Kind = "Minimal"
	xrd.Spec.Versions = []apiextv2.CompositeResourceDefinitionVersion{
		{
			Name:          "v1",
			Served:        true,
			Referenceable: true,
		},
	}

	generated, err := xconvert.XRDToOpenAPI(xrd)
	if err != nil {
		t.Fatal(err)
	}

	schema := generated.Components.Schemas["Minimal"]
	if schema == nil {
		t.Fatal("expected Minimal schema")
	}

	if !schema.Type.Contains("object") {
		t.Fatalf("expected omitted schema to become an object schema, got %q", schema.Type)
	}

	if _, ok := schema.Properties["metadata"]; !ok {
		t.Fatal("expected metadata schema")
	}
}

func TestConvertRequiresVersion(t *testing.T) {
	_, err := xconvert.XRDToOpenAPI(&apiextv2.CompositeResourceDefinition{})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func decodeBasicXRD(t *testing.T) *apiextv2.CompositeResourceDefinition {
	t.Helper()

	xrd := &apiextv2.CompositeResourceDefinition{}
	if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(basicXRD)).Decode(xrd); err != nil {
		t.Fatal(err)
	}

	return xrd
}
