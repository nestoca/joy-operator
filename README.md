# joy-operator

Kubernetes operator for joy.

## Using Private Registries

If your charts live in a private OCI registry, the operator needs to log in to it before it can pull them.

On startup, the operator watches its own namespace for secrets of type `kubernetes.io/dockerconfigjson` with the label `joy.nesto.ca/dockerconfigjson: helm`. For each registry listed in the secret, it runs `helm registry login`. Updating the secret logs in again, and deleting it logs out.

The easiest way to create one is with `kubectl`:

```sh
kubectl create secret docker-registry my-registry \
  --namespace <operator-namespace> \
  --docker-server=us-docker.pkg.dev \
  --docker-username=_json_key \
  --docker-password="$(cat key.json)"

kubectl label secret my-registry \
  --namespace <operator-namespace> \
  joy.nesto.ca/dockerconfigjson=helm
```

Or as a manifest:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-registry
  labels:
    joy.nesto.ca/dockerconfigjson: helm
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64 encoded docker config>
```

A single secret can hold credentials for several registries, and you can have as many of these secrets as you need.

The secret must be in the same namespace as the operator. The namespace comes from the `SERVICE_NAMESPACE` environment variable, which the Helm chart sets for you.
