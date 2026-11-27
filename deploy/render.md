# Deploying to Render with Neon

Both have free plans that work without a payment card. Do not add one: without
it, a service that exceeds its limits is suspended instead of billed.

## Database

Create a Neon project in Singapore. The bot uses the **pooled** connection
string and migrations use the direct one.

In the GitHub repository, add the secret `DATABASE_URL` (direct) to the
`production` environment and set the variable `MIGRATE_ENABLED` to `true`. CI
then migrates the database on every push to `main`.

## Service

Create a Blueprint from this repository, which reads `render.yaml`. Render
deploys a commit after its CI checks pass. Set:

- `TELEGRAM_BOT_TOKEN`
- `DATABASE_URL` (pooled)
- `PUBLIC_BASE_URL` (the service's `onrender.com` address)
- `WEBHOOK_SECRET`, `WEBHOOK_PATH_TOKEN`, `CRON_SECRET`, `UPI_LINK_SECRET`,
  each from `openssl rand -hex 32`

Then run `scripts/set-webhook.sh` with the same values.

## Scheduled jobs

Free services sleep after 15 minutes without traffic and take about a minute to
wake, longer than a scheduler waits. Call `GET /healthz` two minutes before each
job to wake the service.

Create these in cron-job.org with the time zone Asia/Kolkata. Jobs are `POST`
requests with the header `Authorization: Bearer $CRON_SECRET`.

| Job                     | When                  | Wake up call          |
| ----------------------- | --------------------- | --------------------- |
| `/cron/daily-summary`   | 21:00 daily           | 20:58 daily           |
| `/cron/weekly-reminder` | Sunday 11:00          | Sunday 10:58          |
| `/cron/drain-outbox`    | hourly, at minute 30  | hourly, at minute 28  |
| `/cron/cleanup`         | 03:00 daily           | 02:58 daily           |

Draining hourly rather than every few minutes lets the database suspend, which
keeps it inside Neon's monthly compute allowance.
