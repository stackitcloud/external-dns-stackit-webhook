# STACKIT Webhook - ExternalDNS

[![GoTemplate](https://img.shields.io/badge/go/template-black?logo=go)](https://github.com/golang-standards/project-layout)
[![CI](https://github.com/stackitcloud/external-dns-stackit-webhook/actions/workflows/main.yml/badge.svg)](https://github.com/stackitcloud/external-dns-stackit-webhook/actions/workflows/main.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/stackitcloud/external-dns-stackit-webhook)](https://goreportcard.com/report/github.com/stackitcloud/external-dns-stackit-webhook)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/stackitcloud/external-dns-stackit-webhook.svg)](https://github.com/stackitcloud/external-dns-stackit-webhook/releases)
[![Last Commit](https://img.shields.io/github/last-commit/stackitcloud/external-dns-stackit-webhook/main.svg)](https://github.com/stackitcloud/external-dns-stackit-webhook/commits/main)
[![GitHub issues](https://img.shields.io/github/issues/stackitcloud/external-dns-stackit-webhook.svg)](https://github.com/stackitcloud/external-dns-stackit-webhook/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/stackitcloud/external-dns-stackit-webhook.svg)](https://github.com/stackitcloud/external-dns-stackit-webhook/pulls)
[![GitHub stars](https://img.shields.io/github/stars/stackitcloud/external-dns-stackit-webhook.svg?style=social&label=Star&maxAge=2592000)](https://github.com/stackitcloud/external-dns-stackit-webhook/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/stackitcloud/external-dns-stackit-webhook.svg?style=social&label=Fork&maxAge=2592000)](https://github.com/stackitcloud/external-dns-stackit-webhook/network)

ExternalDNS serves as an add-on for Kubernetes designed to automate the management of Domain Name System (DNS)
records for Kubernetes services by utilizing various DNS providers. While Kubernetes traditionally manages DNS
records internally, ExternalDNS augments this functionality by transferring the responsibility of DNS records
management to an external DNS provider such as STACKIT. Consequently, the STACKIT webhook enables the management
of your STACKIT domains within your Kubernetes cluster using
[ExternalDNS](https://github.com/kubernetes-sigs/external-dns).

For utilizing ExternalDNS with STACKIT, it is mandatory to establish a STACKIT project, a service account
within the project, generate an authentication token for the service account, authorize the service account
to create and read dns zones, and finally, establish a STACKIT zone.

## Kubernetes Deployment

The STACKIT webhook is presented as a standard Open Container Initiative (OCI) image released in the
[GitHub container registry](https://github.com/stackitcloud/external-dns-stackit-webhook/pkgs/container/external-dns-stackit-webhook).
The deployment is compatible with all Kubernetes-supported methods. The subsequent example
demonstrates the deployment as a
[sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods)
within the ExternalDNS pod.

```shell
# We create a Secret from an auth token. Alternatively, you can also
# use keys to authenticate the webhook - see "Authentication" below.
kubectl create secret generic external-dns-stackit-webhook --from-literal=auth-token='<Your-Token>'
```

```shell
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-dns
  namespace: default
  labels:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: external-dns
  labels:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["list","watch"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get","watch","list"]
  - apiGroups: [""]
    resources: ["services","endpoints"]
    verbs: ["get","watch","list"]
  - apiGroups: ["extensions","networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get","watch","list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: external-dns-viewer
  labels:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: external-dns
subjects:
  - kind: ServiceAccount
    name: external-dns
    namespace: default
---
apiVersion: v1
kind: Service
metadata:
  name: external-dns
  namespace: default
  labels:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
  ports:
    - name: http
      port: 7979
      targetPort: http
      protocol: TCP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
  namespace: default
  labels:
    app.kubernetes.io/name: external-dns
    app.kubernetes.io/instance: external-dns
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: external-dns
      app.kubernetes.io/instance: external-dns
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: external-dns
        app.kubernetes.io/instance: external-dns
    spec:
      serviceAccountName: external-dns
      securityContext:
        fsGroup: 65534
      containers:
        - name: external-dns
          securityContext:
            capabilities:
              drop:
              - ALL
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 65534
          image: registry.k8s.io/external-dns/external-dns:v0.14.0
          imagePullPolicy: IfNotPresent
          args:
            - --log-level=info
            - --log-format=text
            - --interval=1m
            - --source=service
            - --source=ingress
            - --policy=sync # set it upsert-only if you don't want it to delete records
            - --provider=webhook
          ports:
            - name: http
              protocol: TCP
              containerPort: 7979
          livenessProbe:
            failureThreshold: 2
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          readinessProbe:
            failureThreshold: 6
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
        - name: webhook
          securityContext:
            capabilities:
              drop:
                - ALL
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 65534
          image: ghcr.io/stackitcloud/external-dns-stackit-webhook:v0.2.0
          imagePullPolicy: IfNotPresent
          args:
            - --project-id=c158c736-0300-4044-95c4-b7d404279b35 # your project id
          ports:
            - name: http
              protocol: TCP
              containerPort: 8888
          livenessProbe:
            failureThreshold: 2
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          readinessProbe:
            failureThreshold: 6
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          env:
            - name: STACKIT_SERVICE_ACCOUNT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: external-dns-stackit-webhook
                  key: auth-token
EOF
```

## Configuration

The configuration of the STACKIT webhook can be accomplished through command line arguments and environment variables.
Below are the options that are available.

- `--project-id`/`PROJECT_ID` (required): Specifies the project id of the STACKIT project.
- `--worker`/`WORKER`  (optional): Specifies the number of workers to employ for querying the API. Given that we
  need to iterate over all zones and records, it can be parallelized. However, it is important to avoid
  setting this number excessively high to prevent receiving 429 rate limiting from the API (default 10).
- `--base-url`/`BASE_URL` (optional): Identifies the Base URL for utilizing the API (
  default "https://dns.api.stackit.cloud").
- `--api-port`/`API_PORT` (optional): Specifies the port to listen on (default 8888).
- `--domain-filter`/`DOMAIN_FILER` (optional): Establishes a filter for DNS zone names (default []).
- `--dry-run`/`DRY_RUN` (optional): Specifies whether to perform a dry run (default false).
- `--log-level`/`LOG_LEVEL` (optional): Defines the log level (default "info"). Possible values are: debug, info, warn,
  error.

## Authentication

The STACKIT webhook uses the [STACKIT Go SDK](https://github.com/stackitcloud/stackit-sdk-go) and therefore inherits its
options for authentication: You can use either Token or Key authentication flows. The example above uses the Token flow
for authentication by providing the `STACKIT_SERVICE_ACCOUNT_TOKEN` environment variable in the `Deployment`. For more
information on how to provide e.g. a Service Account Key to be used by the SDK, see
[authentication options for the STACKIT Go SDK](https://github.com/stackitcloud/stackit-sdk-go?tab=readme-ov-file#authentication). 

## FAQ

### 1. Issue with Creating Service using External DNS Annotation

If your zone is `example.runs.onstackit.cloud` and you're trying to create a service with the following external DNS
annotation:

    ```yaml
    apiVersion: v1
    kind: Service
    metadata:
      annotations:
        external-dns.alpha.kubernetes.io/hostname: example.runs.onstackit.cloud
      labels:
        app.kubernetes.io/name: ingress-nginx
        app.kubernetes.io/instance: nginx
        app.kubernetes.io/part-of: ingress-nginx
        app.kubernetes.io/component: controller
      name: nginx-ingress-controller
      namespace: nginx-ingress-controller
    spec:
      type: LoadBalancer
      externalTrafficPolicy: Local
      ipFamilyPolicy: SingleStack
      ipFamilies:
        - IPv4
      ports:
        - name: http
          port: 80
          protocol: TCP
          targetPort: http
        - name: https
          port: 443
          protocol: TCP
          targetPort: https
      selector:
        app.kubernetes.io/component: controller
        app.kubernetes.io/instance: nginx
        app.kubernetes.io/name: ingress-nginx
    ```

<b>Why isn't it working?</b>

<b>Answer</b>: The External DNS will try to create a TXT record named `a-example.runs.onstackit.cloud`, which will fail
because you can't establish a record outside the zone. The solution is to use a name that's within the zone, such as
`nginx.example.runs.onstackit.cloud`.

### 2. Issues with Creating Ingresses not in the Zone

For a project containing the zone `example.runs.onstackit.cloud`, suppose you've created these two ingress:

    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        kubernetes.io/ingress.class: nginx
      name: example-ingress-external-dns
      namespace: default
    spec:
      rules:
        - host: test.example.runs.onstackit.cloud
          http:
            paths:
              - backend:
                  service:
                    name: example
                    port:
                      number: 80
                path: /
                pathType: Prefix
        - host: test.example.stackit.rocks
          http:
            paths:
              - backend:
                  service:
                    name: example
                    port:
                      number: 80
                path: /
                pathType: Prefix
    ```

<b>Why isn't it working?</b>

<b>Answer</b>: External DNS will attempt to establish a record set for `test.example.stackit.rocks`. As the zone
`example.stackit.rocks` isn't within the project, it'll fail. There are two potential fixes:

- Incorporate the zone `example.stackit.rocks` into the project.
- Adjust the domain filter to `example.runs.onstackit.cloud` by setting the domain filter
  flag `--domain-filter="example.runs.onstackit.cloud"`. This will exclude `test.example.stackit.rocks` and only
  generate
  the record set for `test.example.runs.onstackit.cloud`.

## Development

Run the app:

```bash
export BASE_URL="https://dns.api.stackit.cloud"
export PROJECT_ID="c158c736-0300-4044-95c4-b7d404279b35"
export AUTH_TOKEN="your-auth-token"

make run
```

Lint the code:

```bash
make lint
```

Test the code:

```bash
make test
```
