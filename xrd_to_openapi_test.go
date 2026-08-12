package xconvert_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/sn-atmos/xconvert"

	apiextv2 "github.com/crossplane/crossplane/apis/v2/apiextensions/v2"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
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

	schema, ok := generated.Components.Schemas["io.crossplane.example.v1.NetworkingStack"]
	if !ok {
		t.Fatal("expected io.crossplane.example.v1.NetworkingStack schema")
	}

	gvk, ok := schema.Extensions["x-kubernetes-group-version-kind"].([]interface{})
	if !ok || len(gvk) != 1 {
		t.Fatalf("expected x-kubernetes-group-version-kind extension, got %#v", schema.Extensions["x-kubernetes-group-version-kind"])
	}

	gvkEntry, ok := gvk[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GVK extension entry to be an object, got %#v", gvk[0])
	}

	if gvkEntry["group"] != "example.crossplane.io" || gvkEntry["version"] != "v1" || gvkEntry["kind"] != "NetworkingStack" {
		t.Fatalf("unexpected GVK extension: %#v", gvkEntry)
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

func TestConvertBasicXRDs(t *testing.T) {
	xrds, err := xconvert.LoadXRD(basicXRD)
	if err != nil {
		t.Fatal(err)
	}

	if len(xrds) != 2 {
		t.Fatalf("expected 2 XRDs, got %d", len(xrds))
	}

	generated, err := xconvert.XRDsToOpenAPI(xrds)
	if err != nil {
		t.Fatal(err)
	}

	if len(generated) != 1 {
		t.Fatalf("expected 1 OpenAPI document, got %d", len(generated))
	}

	schemas := map[string]*spec.Schema{}
	for _, doc := range generated {
		for name, schema := range doc.Components.Schemas {
			schemas[name] = schema
		}
	}

	if schemas["io.crossplane.example.v1.NetworkingStack"] == nil {
		t.Fatal("expected io.crossplane.example.v1.NetworkingStack schema")
	}

	if schemas["io.crossplane.example.v1.SomethingElse"] == nil {
		t.Fatal("expected io.crossplane.example.v1.SomethingElse schema")
	}
}

func TestConvertAllowsOmittedSchema(t *testing.T) {
	xrd := &apiextv2.CompositeResourceDefinition{}
	xrd.Spec.Group = "example.crossplane.io"
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

	schema := generated.Components.Schemas["io.crossplane.example.v1.Minimal"]
	if schema == nil {
		t.Fatal("expected io.crossplane.example.v1.Minimal schema")
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
