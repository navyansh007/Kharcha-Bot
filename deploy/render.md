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

## Sleeping

Free services sleep after 15 minutes without traffic and take about a minute to
wake. No code runs while asleep, so the bot cannot say that it is waking. It
adds "I was asleep" to the first reply instead, for messages that were sent
before the process started. Button presses carry no send time and get no notice.

## Scheduled jobs

Create these in cron-job.org with the time zone Asia/Kolkata. All but the wake up
call are `POST` requests with the header `Authorization: Bearer $CRON_SECRET`.

| Request                 | When                          |
| ----------------------- | ----------------------------- |
| `GET /healthz`          | minutes 58 and 59 of each hour |
| `/cron/drain-outbox`    | hourly, on the hour           |
| `/cron/daily-summary`   | 21:00 daily                   |
| `/cron/weekly-reminder` | Sunday 11:00                  |
| `/cron/cleanup`         | 03:00 daily                   |

The wake up call exists because a sleeping service takes longer to answer than
the scheduler waits. The call at minute 58 may time out; the one at minute 59
succeeds, which also stops the scheduler from disabling the job for repeated
failures.

Draining hourly rather than every few minutes lets the database suspend, which
keeps it inside Neon's monthly compute allowance.
