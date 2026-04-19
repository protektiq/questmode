# Quest Mode — data flow

High-level request and data paths for the local Minikube stack.

```mermaid
flowchart LR
  user[User]
  fe[frontend_Service]
  be[backend_Service]
  pg[(postgres)]
  rd[(redis)]
  anthropic[Anthropic_API]
  user --> fe
  fe --> be
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

- **User → frontend**: Browser hits the `frontend` Service (e.g. via `kubectl port-forward` or ingress you add later); nginx serves static assets from the Vite build.
- **Frontend → backend**: API calls target the `backend` Service DNS name `backend.quest-mode.svc` (or `http://backend:8080` from another pod in the cluster).
- **Backend → Postgres / Redis**: Connection strings come from the `quest-secrets` Secret (`DATABASE_URL`, `REDIS_URL`).
- **Postgres Pod**: The `postgres` Deployment mounts data on PVC `quest-postgres-pvc` and sets `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` from `quest-secrets`. Non-sensitive defaults for user and database name are also recorded in the `postgres-config` ConfigMap (`k8s/postgres/configmap.yaml`); keep them aligned with the Secret when you change credentials.
- **Backend → Anthropic**: When implemented, the backend uses `ANTHROPIC_API_KEY` from the same Secret for outbound calls to Anthropic’s API (not shown as in-cluster traffic).

Update this diagram when you add ingress, TLS, message queues, or external identity providers.
