# Quest Mode

Monorepo: **Go/Gin** backend, **Vite/TypeScript** frontend, **Kubernetes** manifests for local [Minikube](https://minikube.sigs.k8s.io/), optional **Tekton** pipelines.

## Layout

- `backend/` — Gin HTTP API (`cmd/`, `internal/`, `migrations/`, `go.mod`, `Dockerfile`)
- `frontend/` — Vite + TypeScript static app (`Dockerfile` serves with nginx)
- `k8s/` — Namespaced manifests (`quest-mode`), including Postgres, Redis, app Deployments/Services, Tekton, and secrets templates
- `scripts/` — Helper scripts (`build-images.sh` for local Docker; `build-all.sh` for Minikube’s Docker daemon plus rollout restart)

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

   After this, build application images into Minikube and restart app Deployments:

   ```bash
   ./scripts/build-all.sh
   ```

   To build images only (without Minikube’s Docker), use `./scripts/build-images.sh` from the repo root instead.

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

**Smoke-test** — either use the frontend **NodePort** on the Minikube node IP (port **30080**):

```bash
minikube ip
# Then open http://<that-ip>:30080 — the UI loads and /api is proxied to the backend by nginx.
```

On Linux Docker driver environments (including WSL), direct access to `http://$(minikube ip):30080` may hang depending on host networking. In that case, use one of the local access methods below.

Or use port-forwards:

```bash
kubectl port-forward -n quest-mode svc/frontend 8081:80
kubectl port-forward -n quest-mode svc/backend 8082:8080
```

Then open `http://127.0.0.1:8081` (same-origin `/api/health` via nginx) and, if needed, `http://127.0.0.1:8082/api/health` (curl -X GET http://127.0.0.1:8082/api/health) directly against the backend Service.

Health checks through frontend proxy:

```bash
curl -X GET http://127.0.0.1:8081/api/health
```

Expected response:

```json
{"db":"ok","redis":"ok","status":"ok"}
```

Alternative local tunnel (keep command running in one terminal):

```bash
minikube service -n quest-mode frontend --url
```

Then curl the printed URL plus `/api/health` from another terminal.

## Tekton (optional)

Install Tekton Pipelines and Triggers into Minikube:

```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl wait --for=condition=Ready pods --all -n tekton-pipelines --timeout=120s
```

Apply all Tekton resources for this repository:

```bash
kubectl apply -f k8s/tekton/tasks/ -n quest-mode
kubectl apply -f k8s/tekton/pipeline.yaml -n quest-mode
kubectl apply -f k8s/tekton/workspace-pvc.yaml -n quest-mode
```

Create a new pipeline run (the manifest uses `generateName` for unique run names):

```bash
kubectl create -f k8s/tekton/pipeline-run.yaml -n quest-mode
```

`k8s/tekton/pipeline-run.yaml` passes pipeline params for source checkout (`repo-url` and `revision`). Update `revision` if you want to run CI against a different branch/tag/SHA.

Or use the helper script:

```bash
./scripts/run-ci.sh
```

The helper script deletes old runs labeled `app=quest-mode-ci`, creates a new run from `k8s/tekton/pipeline-run.yaml`, and streams logs with `tkn pipelinerun logs -n quest-mode --last -f`.

## Development (without Kubernetes)

- **Backend**: `cd backend && go run ./cmd` (requires `DATABASE_URL` and `REDIS_URL`, e.g. from a local `.env`)
- **Frontend**: `cd frontend && npm ci && npm run dev`

## Data flow

See [DATA_FLOW.md](./DATA_FLOW.md) for a diagram of traffic among the browser, frontend, backend, Postgres, Redis, and optional Anthropic usage.
