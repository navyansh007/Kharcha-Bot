# Deploying to Cloud Run with Neon

Free tier limits change, so check them before you start.

## 1. Database

Create a Neon project, copy the **pooled** connection string and run the
migrations once from your machine:

```sh
DATABASE_URL='postgres://...' make migrate
```

## 2. Secrets

Store each value in Secret Manager. Generate the random ones with
`openssl rand -hex 32`.

- `TELEGRAM_BOT_TOKEN`
- `DATABASE_URL`
- `WEBHOOK_SECRET`
- `WEBHOOK_PATH_TOKEN`
- `CRON_SECRET`
- `UPI_LINK_SECRET`

```sh
printf '%s' "$VALUE" | gcloud secrets create NAME --data-file=-
```

The Cloud Run service account needs the Secret Manager Secret Accessor role.

## 3. Build and deploy

```sh
IMAGE=asia-south1-docker.pkg.dev/PROJECT/kharcha/bot:first
gcloud artifacts repositories create kharcha --repository-format=docker --location=asia-south1
gcloud auth configure-docker asia-south1-docker.pkg.dev
docker build --platform linux/amd64 -f deploy/Dockerfile -t "$IMAGE" .
docker push "$IMAGE"

SECRETS=$(printf '%s=%s:latest,' \
  TELEGRAM_BOT_TOKEN TELEGRAM_BOT_TOKEN \
  DATABASE_URL DATABASE_URL \
  WEBHOOK_SECRET WEBHOOK_SECRET \
  WEBHOOK_PATH_TOKEN WEBHOOK_PATH_TOKEN \
  CRON_SECRET CRON_SECRET \
  UPI_LINK_SECRET UPI_LINK_SECRET)

gcloud run deploy kharcha-bot \
  --image "$IMAGE" \
  --region asia-south1 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 1 \
  --memory 256Mi \
  --set-env-vars BOT_MODE=webhook \
  --set-secrets "${SECRETS%,}"
```

Then set `PUBLIC_BASE_URL` to the service URL, which turns on the UPI buttons:

```sh
gcloud run services update kharcha-bot --region asia-south1 \
  --update-env-vars PUBLIC_BASE_URL=https://...
```

## 4. Register the webhook

```sh
TELEGRAM_BOT_TOKEN=... PUBLIC_BASE_URL=... WEBHOOK_PATH_TOKEN=... WEBHOOK_SECRET=... \
  ./scripts/set-webhook.sh
```

## 5. Scheduled jobs

Create four jobs in cron-job.org (or Cloud Scheduler). Each sends a `POST` with
the header `Authorization: Bearer $CRON_SECRET`. Times are IST.

| URL                    | When          |
| ---------------------- | ------------- |
| `/cron/daily-summary`  | 21:00 daily   |
| `/cron/weekly-reminder`| Sunday 11:00  |
| `/cron/drain-outbox`   | every 5 min   |
| `/cron/cleanup`        | 03:00 daily   |

A job can spend up to 25 seconds sending messages, so give the scheduler a
request timeout of at least 30 seconds.

## 6. Deploy on merge

The `deploy` job in CI runs only when the repository variable `DEPLOY_ENABLED`
is `true`. It also needs:

- variables `GCP_REGION` and `GCP_PROJECT`
- secrets `GCP_WORKLOAD_IDENTITY_PROVIDER`, `GCP_SERVICE_ACCOUNT`,
  `DATABASE_URL` and `TELEGRAM_BOT_TOKEN`

## 7. Budget alert

Add a billing budget with a very low threshold to the project.
