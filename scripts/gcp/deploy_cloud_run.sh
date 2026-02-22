#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[deploy] %s\n' "$1"
}

fail() {
  printf '[deploy] ERROR: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    fail "missing required environment variable: $name"
  fi
}

require_command gcloud
require_command docker
require_command curl

require_env GCP_PROJECT_ID
require_env GCP_REGION
require_env CLOUD_RUN_SERVICE
require_env IMAGE_REPO
require_env CLOUD_RUN_ENV_FILE

if [[ ! -f "${CLOUD_RUN_ENV_FILE}" ]]; then
  fail "CLOUD_RUN_ENV_FILE does not exist: ${CLOUD_RUN_ENV_FILE}"
fi

IMAGE_TAG="${IMAGE_TAG:-$(date +%Y%m%d%H%M%S)}"
ARTIFACT_HOST="${GCP_REGION}-docker.pkg.dev"
IMAGE_URI="${ARTIFACT_HOST}/${GCP_PROJECT_ID}/${IMAGE_REPO}/${CLOUD_RUN_SERVICE}:${IMAGE_TAG}"

log "project=${GCP_PROJECT_ID} region=${GCP_REGION} service=${CLOUD_RUN_SERVICE}"
log "image=${IMAGE_URI}"

log "configuring docker auth for Artifact Registry"
gcloud auth configure-docker "${ARTIFACT_HOST}" --quiet

log "building container image"
docker build -f backend/Dockerfile -t "${IMAGE_URI}" .

log "pushing container image"
docker push "${IMAGE_URI}"

deploy_args=(
  run deploy "${CLOUD_RUN_SERVICE}"
  --project "${GCP_PROJECT_ID}"
  --region "${GCP_REGION}"
  --platform managed
  --image "${IMAGE_URI}"
  --port 8080
  --set-env-vars-file "${CLOUD_RUN_ENV_FILE}"
  --allow-unauthenticated
)

if [[ -n "${CLOUD_RUN_SERVICE_ACCOUNT:-}" ]]; then
  deploy_args+=(--service-account "${CLOUD_RUN_SERVICE_ACCOUNT}")
fi
if [[ -n "${CLOUD_RUN_MEMORY:-}" ]]; then
  deploy_args+=(--memory "${CLOUD_RUN_MEMORY}")
fi
if [[ -n "${CLOUD_RUN_CPU:-}" ]]; then
  deploy_args+=(--cpu "${CLOUD_RUN_CPU}")
fi
if [[ -n "${CLOUD_RUN_MIN_INSTANCES:-}" ]]; then
  deploy_args+=(--min-instances "${CLOUD_RUN_MIN_INSTANCES}")
fi
if [[ -n "${CLOUD_RUN_MAX_INSTANCES:-}" ]]; then
  deploy_args+=(--max-instances "${CLOUD_RUN_MAX_INSTANCES}")
fi

log "deploying to Cloud Run"
gcloud "${deploy_args[@]}"

SERVICE_URL="$(
  gcloud run services describe "${CLOUD_RUN_SERVICE}" \
    --project "${GCP_PROJECT_ID}" \
    --region "${GCP_REGION}" \
    --format='value(status.url)'
)"
[[ -n "${SERVICE_URL}" ]] || fail "failed to resolve Cloud Run service URL"

log "deployment complete: ${SERVICE_URL}"

log "running smoke checks"
curl -fsS "${SERVICE_URL}/api/health"
printf '\n'
curl -fsS "${SERVICE_URL}/api/capabilities"
printf '\n'

log "success"
