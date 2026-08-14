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
	// The generated file is a Helm template: wrap all CRDs in an installCRDs
	// guard so `task crd-gen` output is reproducible (see chart values.installCRDs).
	if _, err := fmt.Fprintln(os.Stdout, "{{- if .Values.installCRDs -}}"); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	type CRD struct {
		Names apiextv1.CustomResourceDefinitionNames
		Type  reflect.Type
		Scope apiextv1.ResourceScope
		// Status enables the /status subresource. Only resources whose
		// reconciler reports a status (see joy-operator/cmd/operator) set this.
		Status bool
	}

	for _, item := range []CRD{
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "releases",
				Singular:   "release",
				ShortNames: []string{"rel"},
				Kind:       v1alpha1.ReleaseKind,
				ListKind:   "ReleaseList",
			},
			Type:   reflect.TypeFor[v1alpha1.Release](),
			Scope:  apiextv1.NamespaceScoped,
			Status: true,
		},
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "environments",
				Singular:   "environment",
				ShortNames: []string{"env"},
				Kind:       v1alpha1.EnvironmentKind,
				ListKind:   "EnvironmentList",
			},
			Type:   reflect.TypeFor[v1alpha1.Environment](),
			Scope:  apiextv1.ClusterScoped,
			Status: true,
		},
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "projects",
				Singular:   "project",
				ShortNames: []string{"proj"},
				Kind:       v1alpha1.ProjectKind,
				ListKind:   "ProjectList",
			},
			Type:  reflect.TypeFor[v1alpha1.Project](),
			Scope: apiextv1.ClusterScoped,
		},
		{
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     "catalogs",
				Singular:   "catalog",
				ShortNames: []string{"cat"},
				Kind:       v1alpha1.CatalogKind,
				ListKind:   "CatalogList",
			},
			Type:   reflect.TypeFor[v1alpha1.Catalog](),
			Scope:  apiextv1.ClusterScoped,
			Status: true,
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
						Subresources: func() *apiextv1.CustomResourceSubresources {
							if !item.Status {
								return nil
							}
							return &apiextv1.CustomResourceSubresources{Status: &apiextv1.CustomResourceSubresourceStatus{}}
						}(),
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

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to flush encoder: %w", err)
	}

	if _, err := fmt.Fprintln(os.Stdout, "{{- end }}"); err != nil {
		return err
	}

	return nil
}

func sanitizeSchema(schema *apiextv1.JSONSchemaProps) *apiextv1.JSONSchemaProps {
	for _, prop := range []string{"apiVersion", "kind", "metadata"} {
		delete(schema.Properties, prop)
	}
	return schema
}
