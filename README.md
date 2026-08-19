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

ExternalDNS serves as an add-on for Kubernetes designed to automate the management of Domain Name System (DNS) records for Kubernetes services by utilizing various DNS providers. While Kubernetes traditionally manages DNS records internally, ExternalDNS augments this functionality by transferring the responsibility of DNS records management to an external DNS provider such as STACKIT.

Consequently, the STACKIT webhook enables the management of your STACKIT domains within your Kubernetes cluster using [ExternalDNS](https://github.com/kubernetes-sigs/external-dns).

For utilizing ExternalDNS with STACKIT, it is mandatory to establish a STACKIT project, create credentials (either a Service Account Key or configure Workload Identity Federation), authorize the service account with the DNS Admin role, and establish a STACKIT zone.

## Kubernetes Deployment

The STACKIT webhook is provided as a standard Open Container Initiative (OCI) image available in the [GitHub container registry](https://github.com/stackitcloud/external-dns-stackit-webhook/pkgs/container/external-dns-stackit-webhook). It is deployed as a [sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods) within the ExternalDNS pod.

### Option A: Authenticating via Service Account Key

This method uses a standard STACKIT Service Account Key mounted into the container as a file.

```shell
# Create a Secret containing the STACKIT service-account key JSON (as a file).
# This Secret will be mounted into the webhook container; AUTH_KEY_PATH will point to that mounted file.
kubectl -n default create secret generic external-dns-stackit-webhook \
  --from-file=sa.json=/path/to/stackit-service-account-key.json
```

```yaml
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
      volumes:
        - name: stackit-sa-key
          secret:
            secretName: external-dns-stackit-webhook
            items:
              - key: sa.json
                path: sa.json
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
            - --policy=sync # set to upsert-only if you don't want it to delete records
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
          image: ghcr.io/stackitcloud/external-dns-stackit-webhook:v1.0.0
          imagePullPolicy: IfNotPresent
          args:
            - --project-id=c158c736-0300-4044-95c4-b7d404279b35 # your project id
            - --auth-key-path=/var/run/secrets/stackit/sa.json
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
          volumeMounts:
            - name: stackit-sa-key
              mountPath: /var/run/secrets/stackit
              readOnly: true
```

### Option B: Authenticating via Workload Identity Federation (WIF)

If your cluster supports Workload Identity Federation, you can avoid managing long-lived Service Account keys entirely by projecting a short-lived token into the webhook container.

For prerequisites and cluster setup, refer to the [STACKIT Workload Identity Federation documentation](https://docs.stackit.cloud/products/runtime/kubernetes-engine/how-tos/workload-identity/).

```yaml
        - name: webhook
          image: ghcr.io/stackitcloud/external-dns-stackit-webhook:v1.0.0
          args:
            - --project-id=c158c736-0300-4044-95c4-b7d404279b35
            - --auth-wif
          env:
            # The SDK natively looks for this environment variable to locate the projected token
            - name: STACKIT_FEDERATED_TOKEN_FILE
              value: /var/run/secrets/tokens/stackit-token
          volumeMounts:
            - name: stackit-token
              mountPath: /var/run/secrets/tokens
              readOnly: true
      volumes:
        - name: stackit-token
          projected:
            sources:
              - serviceAccountToken:
                  audience: [https://stackit.cloud](https://stackit.cloud)
                  expirationSeconds: 3600
                  path: stackit-token
```

## Configuration

The configuration of the STACKIT webhook is accomplished through command-line arguments or environment variables. The webhook utilizes explicit flags to enforce authentication intent, but it will fall back to automatic SDK discovery if no explicit auth flag is provided.

### Authentication Flags (Choose ONE)
- `--auth-key-path`/`AUTH_KEY_PATH`: Defines the file path of the Service Account key JSON.
- `--auth-wif`/`AUTH_WIF` (boolean): Explicitly enables Workload Identity Federation (WIF) authentication.
- `--auth-wif-token-path`/`AUTH_WIF_TOKEN_PATH` (optional): Defines a custom file path for the federated JWT token for WIF authentication. This is generally only needed if you are overriding standard Kubernetes volume projections.

*Note: If no explicit `--auth-*` flags are provided, the webhook delegates authentication to the STACKIT SDK, which will automatically search the environment for standard SDK variables (e.g., `STACKIT_FEDERATED_TOKEN_FILE`, `STACKIT_SERVICE_ACCOUNT_KEY_PATH`) or a local `~/.stackit/credentials.json` file.*

### General Configuration
- `--project-id`/`PROJECT_ID` (required): Specifies the project ID of the STACKIT project.
- `--token-url`/`TOKEN_URL` (optional): Specifies an alternative OAuth2 endpoint for trading the Service Account key for an access token (default: "https://service-account.api.stackit.cloud/token").
- `--worker`/`WORKER`  (optional): Specifies the number of concurrent workers to employ for querying the API. Given that we iterate over all zones and records, this is parallelized. Avoid setting this excessively high to prevent `429 Too Many Requests` responses from the API (default: 10).
- `--base-url`/`BASE_URL` (optional): Identifies the Base URL for utilizing the STACKIT DNS API (default: "https://dns.api.stackit.cloud").
- `--api-port`/`API_PORT` (optional): Specifies the port the webhook listens on (default: 8888).
- `--domain-filter`/`DOMAIN_FILTER` (optional): Establishes a filter for DNS zone names (default: []).
- `--dry-run`/`DRY_RUN` (optional): Specifies whether to perform a dry run without applying actual DNS changes (default: false).
- `--log-level`/`LOG_LEVEL` (optional): Defines the logging level. Possible values are: `debug`, `info`, `warn`, `error` (default: "info").

## FAQ

### 1. Issue with Creating Service using External DNS Annotation

If your zone is `example.runs.onstackit.cloud` and you're trying to create a service with the following external DNS annotation:

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

**Why isn't it working?**

**Answer**: ExternalDNS will try to create a TXT record named `a-example.runs.onstackit.cloud`, which will fail because you cannot establish a record outside the boundary of the zone. The solution is to use a name that resolves *within* the zone, such as `nginx.example.runs.onstackit.cloud`.

### 2. Issues with Creating Ingresses not in the Zone

For a project containing the zone `example.runs.onstackit.cloud`, suppose you've created these two ingresses:

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

**Why isn't it working?**

**Answer**: ExternalDNS will attempt to establish a record set for `test.example.stackit.rocks`. Because the zone `example.stackit.rocks` does not exist within the project, the operation will fail.

There are two potential fixes:
- Incorporate the zone `example.stackit.rocks` into the STACKIT project.
- Restrict ExternalDNS scoping by applying a domain filter flag `--domain-filter="example.runs.onstackit.cloud"`. This forces the webhook to ignore `test.example.stackit.rocks` and only synchronize records for `test.example.runs.onstackit.cloud`.

## Development

Run the app locally:
```bash
export BASE_URL="[https://dns.api.stackit.cloud](https://dns.api.stackit.cloud)"
export PROJECT_ID="c158c736-0300-4044-95c4-b7d404279b35"
export AUTH_KEY_PATH="/absolute/path/to/stackit-service-account-key.json"

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

### E2E Testing

End-to-end integration tests are orchestrated using [Kuttl](https://kuttl.dev/) and run against a dynamically generated [Kind](https://kind.sigs.k8s.io/) cluster. The test suite builds the webhook locally, deploys it alongside ExternalDNS, and directly verifies real DNS record propagation (A, AAAA, CNAME) against the STACKIT authoritative nameservers.

To run the E2E test suite locally, ensure you have Docker and `kind` installed, then execute:

```bash
make test-e2e-local \
  PROJECT_ID="your-project-id" \
  ZONE_NAME="your.test.zone.cloud" \
  AUTH_KEY_PATH="/absolute/path/to/your/sa.json"
```