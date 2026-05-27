# AGENTS.md - barbican-operator

## Project overview

barbican-operator is a Kubernetes operator that manages
[OpenStack Barbican](https://docs.openstack.org/barbican/latest/) (the key
management service: secret storage, encryption keys, certificates, and
PKCS#11 HSM integration) on OpenShift/Kubernetes. It is part of the
[openstack-k8s-operators](https://github.com/openstack-k8s-operators) project.

Key Barbican domain concepts: **secret stores** (software-backed, PKCS#11 HSM),
**secrets** (symmetric keys, certificates, passphrases, opaque data),
**certificate authorities**, **encryption keys** (for Cinder/Nova/Glance
volume/disk/image encryption).
## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go (modules, multi-module workspace via `go.work`) |
| Scaffolding | [Kubebuilder v4](https://book.kubebuilder.io/) + [Operator SDK](https://sdk.operatorframework.io/) |
| CRD generation | controller-gen (DeepCopy, CRDs, RBAC, webhooks) |
| Config management | Kustomize |
| Packaging | OLM bundle |
| Testing | Ginkgo/Gomega + envtest (functional), KUTTL (integration) |
| Linting | golangci-lint (`.golangci.yaml`) |
| CI | Zuul (`zuul.d/`), Prow (`.ci-operator.yaml`), GitHub Actions |

## Custom Resources

| Kind | Purpose |
|------|---------|
| `Barbican` | Top-level CR. Owns the database, keystone service, transport URL, and spawns sub-CRs for each service component. |
| `BarbicanAPI` | Manages the Barbican API deployment (httpd/WSGI). |
| `BarbicanWorker` | Manages the worker service deployment (async secret operations). |
| `BarbicanKeystoneListener` | Manages the Keystone notification listener (reacts to Keystone events). |

The `Barbican` CR has defaulting and validating admission webhooks.
Sub-CRs are created and owned by the `Barbican` controller -- not intended to
be created directly by users.

## Directory structure

**Maintenance rule:** when directories are added, removed, or renamed, or when
their purpose changes, update this table to match.

| Directory | Contents |
|-----------|----------|
| `api/v1beta1/` | CRD types (`barbican_types.go`, `barbicanapi_types.go`, `barbicanworker_types.go`), conditions, webhook markers |
| `cmd/` | `main.go` entry point |
| `internal/controller/` | Reconcilers: `barbican_controller.go`, `barbicanapi_controller.go`, `barbicankeystonelistener_controller.go`, `barbicanworker_controller.go` |
| `internal/barbican/` | Barbican-level resource builders (db-sync, common helpers) |
| `internal/barbicanapi/` | BarbicanAPI resource builders |
| `internal/barbicankeystonelistener/` | BarbicanKeystoneListener resource builders |
| `internal/barbicanworker/` | BarbicanWorker resource builders |
| `internal/webhook/` | Webhook implementation |
| `templates/` | Config files and scripts mounted into pods via `OPERATOR_TEMPLATES` env var |
| `config/crd,rbac,manager,webhook/` | Generated Kubernetes manifests (CRDs, RBAC, deployment, webhooks) |
| `config/samples/` | Example CRs (Kustomize overlays). Includes PKCS#11 Luna HSM and TLS variants. |
| `test/functional/` | envtest-based Ginkgo/Gomega tests |
| `test/kuttl/` | KUTTL integration tests |
| `hack/` | Helper scripts (CRD schema checker, local webhook runner) |
| `docs/` | Documentation |

## Build commands

After modifying Go code, always run: `make generate manifests fmt vet`.

## Code style guidelines

- Follow standard openstack-k8s-operators conventions and lib-common patterns.
- Use `lib-common` modules for conditions, endpoints, TLS, storage, and other
  cross-cutting concerns rather than re-implementing them.
- CRD types go in `api/v1beta1/`. Controller logic goes in
  `internal/controller/`. Resource-building helpers go in `internal/barbican*`
  packages matching the CR they support.
- Config templates are plain files in `templates/` -- they are mounted at
  runtime via the `OPERATOR_TEMPLATES` environment variable.
- Webhook logic is split between the kubebuilder markers in `api/v1beta1/` and
  the implementation in `internal/webhook/`.

## Testing

- Functional tests use the envtest framework with Ginkgo/Gomega and live in
  `test/functional/`.
- KUTTL integration tests live in `test/kuttl/`.
- Run all functional tests: `make test`.
- When adding a new field or feature, add corresponding test cases in
  `test/functional/` and update fixture data accordingly.

## Key dependencies

- [lib-common](https://github.com/openstack-k8s-operators/lib-common): shared modules for conditions, endpoints, database, TLS, secrets, etc.
- [infra-operator](https://github.com/openstack-k8s-operators/infra-operator): RabbitMQ and topology APIs.
- [mariadb-operator](https://github.com/openstack-k8s-operators/mariadb-operator): database provisioning.
- [keystone-operator](https://github.com/openstack-k8s-operators/keystone-operator): identity service registration.
- [dev-docs/developer.md](https://github.com/openstack-k8s-operators/dev-docs/blob/main/developer.md): developer guide and coding conventions.
