package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yokecd/yoke/pkg/k8s"
	"go.yaml.in/yaml/v3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/pkg/helm"
)

// ChartArgs represent the inputs the joy-operator chart.
// If we were using yoke this would be baked in, instead of a copy we needed to keep in sync with the chart values.
// But we can only dream.
type ChartArgs struct {
	Image                    string                                   `json:"image"`
	Version                  string                                   `json:"version"`
	Controller               ControllerArgs                           `json:"controller"`
	Helm                     *HelmArgs                                `json:"helm,omitzero"`
	ExtraVolumes             []corev1.Volume                          `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts        []corev1.VolumeMount                     `json:"extraVolumeMounts,omitempty"`
	EnvironmentDestinations  map[string]argocd.ApplicationDestination `json:"environmentDestinations,omitempty"`
	EnvironmentSourcePattern string                                   `json:"environmentSourcePattern,omitempty"`
}
type ControllerArgs struct {
	PullMode bool `json:"pullMode"`
}

type HelmArgs struct {
	Registry    string    `json:"registry"`
	Credentials SecretRef `json:"credentials"`
}

type SecretRef struct {
	Secret    string `json:"secret"`
	Key       string `json:"key"`
	MountPath string `json:"mountPath"`
	Data      string `json:"-"`
}

func GetChartArgs() ChartArgs {
	values := ChartArgs{
		Image:      joyOperator,
		Version:    "test",
		Controller: ControllerArgs{PullMode: true},
		EnvironmentDestinations: map[string]argocd.ApplicationDestination{
			"staging": {
				Server:    "https://kubernetes.default.svc",
				Namespace: "default",
			},
		},
		ExtraVolumes: func() (volumes []corev1.Volume) {
			for _, name := range []string{"alpha", "beta"} {
				volumes = append(volumes, corev1.Volume{
					Name: "chart-" + name,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/data/chart/" + name,
							Type: new(corev1.HostPathDirectory),
						},
					},
				})
			}
			return volumes
		}(),
		ExtraVolumeMounts: func() (mounts []corev1.VolumeMount) {
			for _, name := range []string{"alpha", "beta"} {
				mounts = append(mounts, corev1.VolumeMount{
					Name:      "chart-" + name,
					MountPath: "/data/chart/" + name,
					ReadOnly:  true,
				})
			}
			return mounts
		}(),
	}
	if creds := os.Getenv("GAR_CREDS"); creds != "" {
		values.Helm = &HelmArgs{
			Registry: "northamerica-northeast1-docker.pkg.dev",
			Credentials: SecretRef{
				Secret:    "gar-creds",
				Key:       "cred",
				MountPath: "/secrets/creds/gar.creds",
				Data:      creds,
			},
		}
	}
	return values
}

func TestMain(m *testing.M) {
	must(withStdio(exec.Command("kind", "delete", "cluster", "--name=joy-operator")).Run())

	cwd := must2(os.Getwd())

	must(
		WithStandardInput(
			withStdio(exec.Command("kind", "create", "cluster", "--name=joy-operator", "--config", "-")),
			must2(
				json.Marshal(
					map[string]any{
						"apiVersion": "kind.x-k8s.io/v1alpha4",
						"kind":       "Cluster",
						"nodes": []any{
							map[string]any{
								"role": "control-plane",
								"extraMounts": []map[string]any{
									{
										"hostPath":      filepath.Join(cwd, "test_data/chart/alpha"),
										"containerPath": "/data/chart/alpha",
									},
									{
										"hostPath":      filepath.Join(must2(os.Getwd()), "test_data/chart/beta"),
										"containerPath": "/data/chart/beta",
									},
								},
							},
						},
					},
				),
			),
		).Run(),
	)
	must(withStdio(exec.Command("docker", "build", "--tag=joy-operator:test", "../..")).Run())
	must(withStdio(exec.Command("kind", "load", "docker-image", "joy-operator:test", "--name=joy-operator")).Run())

	client := must2(getKubeClient())

	must2(
		client.Clientset.CoreV1().Namespaces().Create(
			context.Background(),
			&corev1.Namespace{
				TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "argocd"},
			},
			metav1.CreateOptions{},
		),
	)

	crdIntf := k8s.TypedInterface[apiextensionsv1.CustomResourceDefinition](client, schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	})

	must2(crdIntf.Apply(context.Background(), argocd.ApplicationCRD, metav1.ApplyOptions{FieldManager: "operator-tests"}))

	must(k8s.WaitForReady(context.Background(), client, argocd.ApplicationCRD, k8s.WaitOptions{
		Timeout:  5 * time.Second,
		Interval: 250 * time.Millisecond,
	}))

	values := GetChartArgs()

	if helm := values.Helm; helm != nil {
		must2(
			client.Clientset.CoreV1().Secrets("default").Create(
				context.Background(),
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: helm.Credentials.Secret,
					},
					Data: map[string][]byte{
						helm.Credentials.Key: must2(base64.StdEncoding.DecodeString(helm.Credentials.Data)),
					},
					Type: corev1.SecretTypeOpaque,
				},
				metav1.CreateOptions{},
			),
		)
	}

	must(
		WithStandardInput(
			withStdio(exec.Command("helm", "install", joyOperator, "../../chart", "-f", "-")),
			must2(json.Marshal(values)),
		).Run(),
	)

	must(
		k8s.WaitForReady(
			context.Background(),
			client,
			&appsv1.Deployment{
				TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "joy-operator", Namespace: "default"},
			},
			k8s.WaitOptions{
				Interval: 250 * time.Millisecond,
				Timeout:  15 * time.Second,
			},
		),
	)

	os.Exit(m.Run())
}

func TestHappyReconciliations(t *testing.T) {
	client, err := getKubeClient()
	require.NoError(t, err)

	projectIntf := k8s.TypedInterface[v1alpha1.Project](client, schema.GroupVersionResource{
		Group:    v1alpha1.ProjectGVK.Group,
		Version:  "v1alpha1",
		Resource: "projects",
	})

	project, err := projectIntf.Create(
		t.Context(),
		&v1alpha1.Project{
			ApiVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       v1alpha1.ProjectGK.Kind,
			ProjectMetadata: v1alpha1.ProjectMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	catalogIntf := k8s.TypedInterface[v1alpha1.Catalog](client, schema.GroupVersionResource{
		Group:    v1alpha1.CatalogGK.Group,
		Version:  "v1alpha1",
		Resource: "catalogs",
	})

	catalog, err := catalogIntf.Create(
		t.Context(),
		&v1alpha1.Catalog{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.GroupVersion.Identifier(),
				Kind:       v1alpha1.CatalogGK.Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "catalog",
			},
			Spec: v1alpha1.CatalogSpec{
				RepoURL:  "https://github.com/testing/catalog",
				Revision: "main",
				Charts: v1alpha1.CatalogCharts{
					Default: "default",
					Refs: map[string]helm.Chart{
						"default": {
							Name:     "alpha",
							RepoURL:  "file:///data/chart",
							Version:  "6.6.6",
							Mappings: map[string]any{},
						},
					},
				},
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	envIntf := k8s.TypedInterface[v1alpha1.Environment](client, schema.GroupVersionResource{
		Group:    v1alpha1.EnvironmentGK.Group,
		Version:  "v1alpha1",
		Resource: "environments",
	})

	env, err := envIntf.Create(
		t.Context(),
		&v1alpha1.Environment{
			ApiVersion:          v1alpha1.GroupVersion.Identifier(),
			Kind:                v1alpha1.EnvironmentGK.Kind,
			EnvironmentMetadata: v1alpha1.EnvironmentMetadata{ObjectMeta: metav1.ObjectMeta{Name: "staging"}},
			Spec: v1alpha1.EnvironmentSpec{
				Values: map[string]any{
					"env": "alpha",
				},
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	EventuallyNoErrorf(
		t,
		func() error {
			_, err := client.Clientset.CoreV1().Namespaces().Get(t.Context(), "staging", metav1.GetOptions{})
			return err
		},
		50*time.Millisecond,
		2*time.Second,
		"failed to get corresponding namespace for env",
	)

	releaseIntf := k8s.TypedInterface[v1alpha1.Release](client, schema.GroupVersionResource{
		Group:    v1alpha1.ReleaseGK.Group,
		Version:  "v1alpha1",
		Resource: "releases",
	})

	releaseIntf = releaseIntf.Namespace(env.Namespace)

	release, err := releaseIntf.Create(
		t.Context(),
		&v1alpha1.Release{
			Kind:       v1alpha1.ReleaseGK.Kind,
			ApiVersion: v1alpha1.GroupVersion.Identifier(),
			ReleaseMetadata: v1alpha1.ReleaseMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: env.Name,
				},
			},
			Spec: v1alpha1.ReleaseSpec{
				Project: project.Name,
				Version: "1.2.3",
				Values: map[string]any{
					"freehand": "hello",
					"env":      "{{ .Environment.Spec.Values.env }}",
				},
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	appsIntf := k8s.TypedInterface[argocd.Application](client, argocd.ApplicationGVR).Namespace("argocd")

	EventuallyNoErrorf(
		t,
		func() error {
			_, err := appsIntf.Get(t.Context(), "staging-test", metav1.GetOptions{})
			return err
		},
		50*time.Millisecond,
		2*time.Second,
		"failed to get staging-test application",
	)

	apps, err := appsIntf.List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, apps, 3)

	byName := make(map[string]*argocd.Application, len(apps))
	for _, app := range apps {
		byName[app.Name] = app
	}

	assertions := map[string]func(t *testing.T, app *argocd.Application){
		"catalog": func(t *testing.T, app *argocd.Application) {
			require.Equal(t, "argocd", app.Namespace)
			require.Equal(t, "default", app.Spec.Project)
			require.Equal(
				t,
				argocd.ApplicationSource{
					RepoURL:        "https://github.com/testing/catalog",
					TargetRevision: "main",
					Directory:      argocd.SourceDirectory{Include: "environments/*/env.yaml"},
				},
				app.Spec.Source,
			)
			require.Equal(t, argocd.ApplicationDestination{Server: "http://kubernetes.svc.local"}, app.Spec.Destination)
		},
		"staging": func(t *testing.T, app *argocd.Application) {
			require.Equal(t, "argocd", app.Namespace)
			require.Equal(t, "default", app.Spec.Project)
			require.Equal(
				t,
				argocd.ApplicationSource{
					RepoURL:        "https://github.com/testing/catalog",
					TargetRevision: "main",
					Directory: argocd.SourceDirectory{
						Recurse: true,
						Include: "environments/staging/releases",
					},
				},
				app.Spec.Source,
			)
			require.Equal(
				t,
				argocd.ApplicationDestination{
					Server:    "http://kubernetes.svc.local",
					Namespace: "staging",
				},
				app.Spec.Destination,
			)
		},
		"staging-test": func(t *testing.T, app *argocd.Application) {
			require.Equal(t, "argocd", app.Namespace)
			require.Equal(t, "default", app.Spec.Project)
			require.Equal(
				t,
				map[string]string{
					"nesto.ca/env":        "staging",
					"nesto.ca/project":    "test",
					"nesto.ca/release":    "true",
					"nesto.ca/repository": "",
					"nesto.ca/stream":     "lost",
					"nesto.ca/version":    "1.2.3",
				},
				app.Labels,
			)
			require.Equal(
				t,
				map[string]string{
					"nesto.ca/release-version":                                        "1.2.3",
					"notifications.argoproj.io/subscribe.on-production-release.slack": "notif-releases",
					"notifications.argoproj.io/subscribe.on-release.slack":            "",
				},
				app.Annotations,
			)
			require.Equal(
				t,
				argocd.ApplicationSource{
					RepoURL:        "file:///data/chart",
					TargetRevision: "6.6.6",
					Chart:          "alpha",
					Helm: argocd.SourceHelm{
						ReleaseName: "test",
						Values:      "chartname: alpha\nenv: alpha\nfreehand: hello\n",
					},
				},
				app.Spec.Source,
			)
			require.Equal(
				t,
				argocd.ApplicationDestination{
					Server:    "https://kubernetes.default.svc",
					Namespace: "default",
				},
				app.Spec.Destination,
			)
			require.Equal(t, []string{"CreateNamespace=true"}, app.Spec.SyncPolicy.SyncOptions)
		},
	}

	for name, assert := range assertions {
		app, ok := byName[name]
		require.Truef(t, ok, "expected application %q to exist", name)
		assert(t, app)
	}

	maps.Copy(release.Spec.Values, map[string]any{"freehand": "updated"})

	_, err = releaseIntf.Update(t.Context(), release, metav1.UpdateOptions{})
	require.NoError(t, err)

	EventuallyNoErrorf(
		t,
		func() error {
			app, err := appsIntf.Get(t.Context(), "staging-test", metav1.GetOptions{})
			if err != nil {
				return err
			}
			var values map[string]any
			if err := yaml.Unmarshal([]byte(app.Spec.Source.Helm.Values), &values); err != nil {
				return FatalError{err}
			}
			if values["freehand"] != "updated" {
				return fmt.Errorf("freehand property not updated")
			}
			return nil
		},
		50*time.Millisecond,
		2*time.Second,
		"failed to see freehand property updated",
	)

	env.Spec.Values["env"] = "updated"
	_, err = envIntf.Update(t.Context(), env, metav1.UpdateOptions{})
	require.NoError(t, err)

	EventuallyNoErrorf(
		t,
		func() error {
			app, err := appsIntf.Get(t.Context(), "staging-test", metav1.GetOptions{})
			if err != nil {
				return err
			}
			var values map[string]any
			if err := yaml.Unmarshal([]byte(app.Spec.Source.Helm.Values), &values); err != nil {
				return FatalError{err}
			}
			if values["env"] != "updated" {
				return fmt.Errorf("env property not updated")
			}
			return nil
		},
		50*time.Millisecond,
		2*time.Second,
		"failed to see env property updated",
	)

	defaultChartRef := catalog.Spec.Charts.Refs["default"]
	defaultChartRef.Name = "beta"

	catalog.Spec.Charts.Refs["default"] = defaultChartRef
	catalog.Spec.Revision = "HEAD"

	_, err = catalogIntf.Update(t.Context(), catalog, metav1.UpdateOptions{})
	require.NoError(t, err)

	EventuallyNoErrorf(
		t,
		func() error {
			app, err := appsIntf.Get(t.Context(), "staging-test", metav1.GetOptions{})
			if err != nil {
				return err
			}
			var values map[string]any
			if err := yaml.Unmarshal([]byte(app.Spec.Source.Helm.Values), &values); err != nil {
				return FatalError{err}
			}
			if values["chartname"] != "beta" {
				return fmt.Errorf("chartname property not updated")
			}
			return nil
		},
		50*time.Millisecond,
		2*time.Second,
		"failed to see chartname property updated",
	)

	// safe to make these assertions without an eventually wrapper since we are running them after the release application got updated.
	for _, name := range []string{"staging", "catalog"} {
		app, err := appsIntf.Get(t.Context(), name, metav1.GetOptions{})
		require.NoError(t, err, "failed to get app:", name)
		require.Equal(t, "HEAD", app.Spec.Source.TargetRevision, "unexpected target revision for app:", name)
	}

	// And finally, the operator is non-destructive.
	require.NoError(t, catalogIntf.Delete(t.Context(), "catalog", metav1.DeleteOptions{}))
	require.NoError(t, envIntf.Delete(t.Context(), "staging", metav1.DeleteOptions{}))
	require.NoError(t, releaseIntf.Namespace("staging").Delete(t.Context(), "test", metav1.DeleteOptions{}))

	EventuallyNoErrorf(
		t,
		func() error {
			if _, err := catalogIntf.Get(t.Context(), "catalog", metav1.GetOptions{}); !kerrors.IsNotFound(err) {
				return fmt.Errorf("catalog not removed from cluster: expected not found but got: %v", err)
			}
			if _, err := envIntf.Get(t.Context(), "staging", metav1.GetOptions{}); !kerrors.IsNotFound(err) {
				return fmt.Errorf("environment not removed from cluster: expected not found but got: %v", err)
			}
			if _, err := releaseIntf.Get(t.Context(), "test", metav1.GetOptions{}); !kerrors.IsNotFound(err) {
				return fmt.Errorf("release not removed from cluster: expected not found but got: %v", err)
			}
			return nil
		},
		100*time.Millisecond,
		15*time.Second,
		"failed to delete resources from cluster",
	)

	apps, err = appsIntf.List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, apps, 3)
}

func TestEnvironmentSourcePattern(t *testing.T) {
	client, err := getKubeClient()
	require.NoError(t, err)

	catalogIntf := k8s.TypedInterface[v1alpha1.Catalog](client, schema.GroupVersionResource{
		Group:    v1alpha1.CatalogGK.Group,
		Version:  "v1alpha1",
		Resource: "catalogs",
	})

	_, err = catalogIntf.Create(
		t.Context(),
		&v1alpha1.Catalog{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.GroupVersion.Identifier(),
				Kind:       v1alpha1.CatalogGK.Kind,
			},
			ObjectMeta: metav1.ObjectMeta{Name: "catalog"},
			Spec: v1alpha1.CatalogSpec{
				RepoURL:  "https://github.com/testing/catalog",
				Revision: "main",
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	appsIntf := k8s.TypedInterface[argocd.Application](client, argocd.ApplicationGVR).Namespace("argocd")

	EventuallyNoErrorf(
		t,
		func() error {
			_, err := appsIntf.Get(t.Context(), "catalog", metav1.GetOptions{})
			return err
		},
		50*time.Second,
		2*time.Second,
		"failed to get catalog app of apps",
	)

	app, err := appsIntf.Get(t.Context(), "catalog", metav1.GetOptions{})
	require.NoError(t, err)

	require.Equal(t, "environments/*/env.yaml", app.Spec.Source.Directory.Include)

	values := GetChartArgs()

	values.EnvironmentSourcePattern = "environments/staging/env.yaml"

	require.NoError(
		t,
		WithStandardInput(
			withStdio(exec.Command("helm", "upgrade", joyOperator, "../../chart", "-f", "-")),
			must2(json.Marshal(values)),
		).Run(),
	)

	EventuallyNoErrorf(
		t,
		func() error {
			app, err := appsIntf.Get(t.Context(), "catalog", metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}
			if app.Spec.Source.Directory.Include != values.EnvironmentSourcePattern {
				return fmt.Errorf(
					"expected catalog app of apps to include %q but got %q",
					values.EnvironmentSourcePattern,
					app.Spec.Source.Directory.Include,
				)
			}
			return nil
		},
		250*time.Millisecond,
		10*time.Second,
		"failed assert environment source pattern",
	)
}

func EventuallyNoErrorf(t *testing.T, fn func() error, tick time.Duration, timeout time.Duration, msg string, args ...any) {
	var (
		ticker   = time.NewTimer(0)
		deadline = time.Now().Add(timeout)
	)

	for range ticker.C {
		err := fn()
		if err == nil {
			return
		}
		if errors.Is(err, FatalError{}) || time.Now().After(deadline) {
			require.NoErrorf(t, err, msg, args...)
			return
		}
		ticker.Reset(tick)
	}
}

type FatalError struct {
	error
}

func Fatal(err error) FatalError {
	return FatalError{err}
}

func (FatalError) Is(err error) bool {
	_, ok := err.(FatalError)
	return ok
}

func withStdio(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func WithStandardInput(cmd *exec.Cmd, input []byte) *exec.Cmd {
	cmd.Stdin = bytes.NewReader(input)
	return cmd
}

func getKubeClient() (*k8s.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube/config"))
	if err != nil {
		return nil, fmt.Errorf("failed to construct kuberentes rest config: %w", err)
	}
	return k8s.NewClient(cfg)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func must2[T any](value T, err error) T {
	must(err)
	return value
}
