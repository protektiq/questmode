# Quest Mode

Monorepo: **Go/Gin** backend, **Vite/TypeScript** frontend, **Kubernetes** manifests for local [Minikube](https://minikube.sigs.k8s.io/), optional **Tekton** pipelines.

## Layout

- `backend/` — Gin HTTP API (`src/`, `go.mod`, `Dockerfile`)
- `frontend/` — Vite + TypeScript static app (`Dockerfile` serves with nginx)
- `k8s/` — Namespaced manifests (`quest-mode`), including Postgres, Redis, app Deployments/Services, Tekton, and secrets templates
- `scripts/` — Helper scripts (e.g. image builds)

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (for building images and for the Minikube driver below)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

Optional:

- [Tekton Pipelines](https://tekton.dev/docs/installation/) on the cluster if you want to apply `k8s/tekton/` manifests

## Local setup (Minikube)

Run these from the repository root.

1. **Start Minikube** (Docker driver):

   ```bash
   minikube start --driver=docker
   ```

2. **Use Minikube’s Docker daemon** (build images into it so Pods can use `imagePullPolicy: Never` without a registry):

   ```bash
   eval $(minikube docker-env)
   ```

   After this, build application images:

   ```bash
   ./scripts/build-images.sh
   ```

3. **Create namespace**:

   ```bash
   kubectl apply -f k8s/namespace.yaml
   ```

4. **Secrets** — copy the template, fill in real values, then apply:

   ```bash
   cp k8s/secrets/secrets.template.yaml k8s/secrets/secrets.yaml
   # Edit k8s/secrets/secrets.yaml: set ANTHROPIC_API_KEY, ADMIN_KEY, POSTGRES_PASSWORD, and DATABASE_URL.
   # Use the same DB password in POSTGRES_PASSWORD and in DATABASE_URL (same user/db as in k8s/postgres/configmap.yaml).
   kubectl apply -f k8s/secrets/
   ```

   `k8s/secrets/secrets.yaml` is gitignored; only the template is committed.

   The Postgres Deployment reads `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` from this Secret. Non-sensitive defaults (`POSTGRES_USER`, `POSTGRES_DB`) are duplicated in `k8s/postgres/configmap.yaml` for reference; keep those values in sync if you change them.

   If you previously used the old PVC name `postgres-pvc`, switch to `quest-postgres-pvc` by deleting the old Postgres Deployment/PVC in Minikube (this wipes local DB data) or migrating the volume claim manually.

5. **Apply all Kubernetes manifests** under `k8s/` (recursive — includes `postgres/`, `redis/`, `backend/`, `frontend/`, `tekton/`, etc.):

   ```bash
   kubectl apply -R -f k8s/
   ```

   `kubectl apply -f k8s/` without `-R` only applies YAML files **directly** in `k8s/`, not subdirectories. This repository expects **`-R`** so nested folders are included.

   If this fails because Tekton CRDs are not installed, either [install Tekton Pipelines](https://tekton.dev/docs/installation/) first, or apply without the Tekton folder (for example omit `k8s/tekton` from that run by applying only `k8s/postgres`, `k8s/redis`, `k8s/backend`, and `k8s/frontend` recursively).

**Next:** wait for workloads (first Postgres start can take a minute):

```bash
kubectl get pods -n quest-mode -w
```

**Smoke-test** with port-forwards:

```bash
kubectl port-forward -n quest-mode svc/frontend 8081:80
kubectl port-forward -n quest-mode svc/backend 8082:8080
```

Then open `http://127.0.0.1:8081` and `http://127.0.0.1:8082/health`.

## Tekton (optional)

Install Tekton Pipelines on your cluster first (see [Tekton installation](https://tekton.dev/docs/installation/)). Then ensure `k8s/tekton/` resources are applied (included in step 5 with `kubectl apply -R -f k8s/`).

The sample `PipelineRun` is named `quest-mode-pipeline-run-sample`. If you need to re-apply it after a change, delete it first:

```bash
kubectl delete pipelinerun quest-mode-pipeline-run-sample -n quest-mode --ignore-not-found
kubectl apply -f k8s/tekton/pipeline-run.yaml
```

## Development (without Kubernetes)

- **Backend**: `cd backend && go run ./src`
- **Frontend**: `cd frontend && npm ci && npm run dev`

## Data flow

See [DATA_FLOW.md](./DATA_FLOW.md) for a diagram of traffic among the browser, frontend, backend, Postgres, Redis, and optional Anthropic usage.
