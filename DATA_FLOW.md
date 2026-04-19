# Quest Mode — data flow

High-level request and data paths for the local Minikube stack.

```mermaid
flowchart LR
  user[User]
  np[NodePort_30080]
  fe[frontend_Service]
  be[backend_Service]
  storyEngine[story_StateManager]
  pg[(postgres)]
  rd[(redis)]
  anthropic[Anthropic_API]
  user --> np
  np --> fe
  fe --> be
  be --> storyEngine
  storyEngine -->|save_load_state| rd
  storyEngine -->|flush_progress_metrics| pg
  be --> pg
  be --> rd
  be -. optional .-> anthropic
```

Postgres bootstrap credentials (in-cluster):

```mermaid
flowchart LR
  pg_cm[postgres_ConfigMap]
  quest_sec[quest_secrets_Secret]
  pg_pod[postgres_Pod]
  pg_cm -.->|non_secret_defaults| pg_pod
  quest_sec -->|POSTGRES_user_password_db| pg_pod
```

- **User → frontend**: Browser hits the `frontend` Service. On Minikube, the Service is a **NodePort** (`30080`); open `http://$(minikube ip):30080` on the host. You can also use `kubectl port-forward -n quest-mode svc/frontend 8081:80`. Nginx serves the Vite build from `/usr/share/nginx/html` and uses SPA fallback (`try_files` → `index.html`).
- **Frontend → backend**: In the cluster, nginx proxies `/api/*` to `http://backend:8080/api/*`. The browser can call same-origin `/api/...` (no CORS needed). The backend Service DNS name is `backend.quest-mode.svc` (short name `backend` within the namespace).
- **Backend → Postgres / Redis**: Connection strings come from the `quest-secrets` Secret (`DATABASE_URL`, `REDIS_URL`). On startup the backend applies SQL migrations from `backend/migrations/` (embedded in the binary) and records them in `schema_migrations`; Kubernetes readiness uses `GET /api/health` on port 8080.
- **Story state machine**: Story runtime state is cached in Redis using `story:state:{learnerID}` with 24-hour TTL and flushed to `quest_sessions` in Postgres (`tasks_completed`, `engagement_seconds`) by `session_id`.
- **Postgres Pod**: The `postgres` Deployment mounts data on PVC `quest-postgres-pvc` and sets `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` from `quest-secrets`. Non-sensitive defaults for user and database name are also recorded in the `postgres-config` ConfigMap (`k8s/postgres/configmap.yaml`); keep them aligned with the Secret when you change credentials.
- **Backend → Anthropic**: When implemented, the backend uses `ANTHROPIC_API_KEY` from the same Secret for outbound calls to Anthropic’s API (not shown as in-cluster traffic).

Update this diagram when you add ingress, TLS, message queues, or external identity providers.

## Tekton CI flow (Minikube)

```mermaid
flowchart LR
  sourceRepo[source_repo] --> fetchTask[fetch_source]
  sourcePvc[tekton_source_pvc] --> fetchTask
  fetchTask --> lintTask[go_lint]
  lintTask --> testTask[go_test]
  lintTask --> feTask[frontend_build]
```

- **Source fetch**: `fetch-source` validates `repo-url` and `revision`, then checks out the requested ref into the shared workspace.
- **Shared workspace**: `tekton-source-pvc` is mounted as the `source` workspace for all CI tasks.
- **Quality gates**: `go-lint` runs `gofmt` and `go vet`; `go-test` and `frontend-build` run in parallel after lint passes.
- **Deploy flow**: image build and deployment restart are handled separately by local script `./scripts/build-all.sh` against Minikube's Docker daemon.
