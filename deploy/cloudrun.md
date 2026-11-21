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

```sh
printf '%s' "$VALUE" | gcloud secrets create NAME --data-file=-
```

The Cloud Run service account needs the Secret Manager Secret Accessor role.

## 3. Deploy

```sh
gcloud run deploy kharcha-bot \
  --source . \
  --region asia-south1 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 1 \
  --memory 256Mi \
  --set-env-vars BOT_MODE=webhook \
  --set-secrets TELEGRAM_BOT_TOKEN=TELEGRAM_BOT_TOKEN:latest,DATABASE_URL=DATABASE_URL:latest,WEBHOOK_SECRET=WEBHOOK_SECRET:latest,WEBHOOK_PATH_TOKEN=WEBHOOK_PATH_TOKEN:latest
```

`--source` expects a Dockerfile at the repository root. Either copy
`deploy/Dockerfile` there for the deploy, or build and push the image yourself
and pass `--image`.

Set `PUBLIC_BASE_URL` to the service URL and deploy again.

## 4. Register the webhook

```sh
TELEGRAM_BOT_TOKEN=... PUBLIC_BASE_URL=... WEBHOOK_PATH_TOKEN=... WEBHOOK_SECRET=... \
  ./scripts/set-webhook.sh
```

## 5. Budget alert

Add a billing budget with a very low threshold to the project.
