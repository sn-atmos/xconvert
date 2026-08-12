package xconvert

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	apiextv2 "github.com/crossplane/crossplane/apis/v2/apiextensions/v2"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

const OpenAPIVersion = "3.0.0"

// Loads XRDs from bytes
func LoadXRD(data []byte) ([]*apiextv2.CompositeResourceDefinition, error) {
	var xrds []*apiextv2.CompositeResourceDefinition
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
	for {
		xrd := &apiextv2.CompositeResourceDefinition{}
		if err := decoder.Decode(xrd); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		if xrd.TypeMeta.APIVersion == apiextv2.SchemeGroupVersion.String() &&
			xrd.TypeMeta.Kind == apiextv2.CompositeResourceDefinitionKind {
			xrds = append(xrds, xrd)
		}
	}

	return xrds, nil
}

// Converts a slice of XRDs to a slice of spec3.OpenAPI.
// Groups documents by the defined api-group/version cause crossplane will only select 1 openapi document based on that.
// ref https://github.com/crossplane/crossplane/blob/main/internal/xfn/required_schemas.go#L87
func XRDsToOpenAPI(xrds []*apiextv2.CompositeResourceDefinition) ([]spec3.OpenAPI, error) {
	result := map[schema.GroupVersion]spec3.OpenAPI{}

	for _, xrd := range xrds {
		doc, err := XRDToOpenAPI(xrd)
		if err != nil {
			return nil, fmt.Errorf("failed to convert xrd %q to openapi schema: %v", xrd.Spec.Names.Kind, err)
		}

		gv := schema.GroupVersion{Group: xrd.Spec.Group, Version: xrd.Spec.Versions[0].Name}
		existing, ok := result[gv]
		if !ok {
			result[gv] = *doc
			continue
		}

		maps.Copy(existing.Components.Schemas, doc.Components.Schemas)
		result[gv] = existing
	}

	return slices.Collect(maps.Values(result)), nil
}

// Converts an XRD to spec3.OpenAPI, adding kubernetes metadata, apiversion, and kind to the schema.
func XRDToOpenAPI(xrd *apiextv2.CompositeResourceDefinition) (*spec3.OpenAPI, error) {
	if xrd == nil {
		return nil, fmt.Errorf("xrd is nil")
	}
	if len(xrd.Spec.Versions) == 0 {
		return nil, fmt.Errorf("xrd has no versions")
	}

	version := xrd.Spec.Versions[0]
	schema := objectSchema()
	if version.Schema != nil && len(version.Schema.OpenAPIV3Schema.Raw) > 0 {
		if err := json.Unmarshal(version.Schema.OpenAPIV3Schema.Raw, schema); err != nil {
			return nil, fmt.Errorf("unmarshal openAPIV3Schema: %w", err)
		}
	}

	if schema.Properties == nil {
		schema.Properties = map[string]spec.Schema{}
	}
	schema.Properties["apiVersion"] = stringSchema()
	schema.Properties["kind"] = stringSchema()
	schema.Properties["metadata"] = objectMetaSchema()
	schema.Extensions = spec.Extensions{
		"x-kubernetes-group-version-kind": []map[string]string{
			{
				"group":   xrd.Spec.Group,
				"version": version.Name,
				"kind":    xrd.Spec.Names.Kind,
			},
		},
	}

	schemaName := openAPISchemaName(xrd.Spec.Group, version.Name, xrd.Spec.Names.Kind)

	return &spec3.OpenAPI{
		Version: OpenAPIVersion,
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:   xrd.Spec.Names.Kind,
				Version: version.Name,
			},
		},
		Paths: &spec3.Paths{
			Paths: map[string]*spec3.Path{},
		},
		Components: &spec3.Components{
			Schemas: map[string]*spec.Schema{
				schemaName: schema,
			},
		},
	}, nil
}

func openAPISchemaName(group, version, kind string) string {
	parts := strings.Split(group, ".")
	for i := 0; i < len(parts)/2; i++ {
		parts[i], parts[len(parts)-1-i] = parts[len(parts)-1-i], parts[i]
	}

	parts = append(parts, version, kind)
	return strings.Join(parts, ".")
}

func objectMetaSchema() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"object"},
			Properties: map[string]spec.Schema{
				"name":        stringSchema(),
				"namespace":   stringSchema(),
				"labels":      stringMapSchema(),
				"annotations": stringMapSchema(),
			},
		},
	}
}

func objectSchema() *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"object"},
		},
	}
}

func stringSchema() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"string"},
		},
	}
}

func stringMapSchema() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"object"},
			AdditionalProperties: &spec.SchemaOrBool{
				Allows: true,
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: spec.StringOrArray{"string"},
					},
				},
			},
		},
	}
}
