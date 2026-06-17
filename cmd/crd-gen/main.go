package main

import (
	"fmt"
	"os"
	"reflect"

	"go.yaml.in/yaml/v3"

	"github.com/yokecd/yoke/pkg/openapi"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/nestoca/joy/api/v1alpha1"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	type CRD struct {
		Names apiextv1.CustomResourceDefinitionNames
		Type  reflect.Type
		Scope apiextv1.ResourceScope
	}

	for _, item := range []CRD{
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "releases",
				Singular:   "release",
				ShortNames: []string{"rel"},
				Kind:       v1alpha1.KindRelease,
				ListKind:   "Releases",
			},
			Type:  reflect.TypeFor[v1alpha1.Release](),
			Scope: apiextv1.NamespaceScoped,
		},
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "environments",
				Singular:   "environment",
				ShortNames: []string{"env"},
				Kind:       v1alpha1.KindEnvironment,
				ListKind:   "Environments",
			},
			Type:  reflect.TypeFor[v1alpha1.Environment](),
			Scope: apiextv1.ClusterScoped,
		},
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "projects",
				Singular:   "project",
				ShortNames: []string{"proj"},
				Kind:       v1alpha1.KindProject,
				ListKind:   "Projects",
			},
			Type:  reflect.TypeFor[v1alpha1.Project](),
			Scope: apiextv1.ClusterScoped,
		},
	} {
		crd := apiextv1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiextv1.SchemeGroupVersion.Identifier(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        item.Names.Plural + ".joy.nesto.ca",
				Annotations: map[string]string{"helm.sh/resource-policy": "keep"},
			},
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "joy.nesto.ca",
				Names: item.Names,
				Scope: item.Scope,
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema:  &apiextv1.CustomResourceValidation{OpenAPIV3Schema: sanitizeSchema(openapi.SchemaFrom(item.Type))},
					},
				},
			},
		}

		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&crd)
		if err != nil {
			return fmt.Errorf("failed to convert %s unstructured format: %w", crd.Name, err)
		}

		delete(raw, "status")

		if err := encoder.Encode(raw); err != nil {
			return fmt.Errorf("failed to encode %s: %w", crd.Name, err)
		}
	}

	return nil
}

func sanitizeSchema(schema *apiextv1.JSONSchemaProps) *apiextv1.JSONSchemaProps {
	for _, prop := range []string{"apiVersion", "kind", "metadata"} {
		delete(schema.Properties, prop)
	}
	return schema
}
