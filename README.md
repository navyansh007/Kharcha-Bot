# Kharcha-Bot

A Telegram bot, written in Go, for tracking personal spending and splitting
shared expenses without installing another app. Send `120 chai` in a private
chat and it is logged. Send `/split 900 dinner @rahul @aman` in a group and
everyone's balance updates.

The bot never moves money. For settling up it opens your UPI app with the amount
filled in, and you tell it once you have paid.

## Commands

| Where   | Command                               | What it does                                  |
| ------- | ------------------------------------- | --------------------------------------------- |
| Private | `120 chai`, `chai 120`, `900 x #tag`  | Log an expense                                |
| Private | `/today`, `/week`, `/month`           | Totals by category                            |
| Private | `/budget food 3000`                   | Monthly budget with warnings at 80% and 100%  |
| Private | `/summary on`                         | Daily total at 9 pm                           |
| Private | `/export`                             | CSV of your records                           |
| Private | `/deleteme`                           | Erase your personal data                      |
| Group   | `/join`                               | Take part in splits                           |
| Group   | `/split 900 dinner`                   | Split equally between everyone who has joined |
| Group   | `/split 900 dinner @a @b`             | Split equally between you, a and b            |
| Group   | `/split 900 dinner @a:300 @b:600`     | Exact shares                                  |
| Group   | `/balance`                            | Who owes whom, with settle buttons            |
| Group   | `/settle @a 300`                      | Record a payment you made                     |
| Both    | `/setupi name@okaxis`                 | Let others pay you by UPI                     |
| Both    | `/undo`                               | Undo your last entry, within 10 minutes       |

Any command sent in a group also joins you to it.

## Running it locally

You need Go (see `go.mod` for the version) and Docker.

1. Create a development bot with [@BotFather](https://t.me/BotFather).
2. Copy the example settings and put the token in `TELEGRAM_BOT_TOKEN`.

   ```sh
   cp .env.example .env
   ```

3. Start Postgres, apply the migrations and run the bot.

   ```sh
   make db-up
   make migrate
   make dev
   ```

In `polling` mode the bot fetches updates itself, so no public URL is needed. A
bot that already has a webhook cannot poll, which is one more reason to keep
separate development and production bots.

UPI pay buttons need `PUBLIC_BASE_URL` (https) and `UPI_LINK_SECRET`. Locally you
can get a URL with `cloudflared tunnel --url http://localhost:8080`.

## Settings

| Variable             | Default        | Notes                                              |
| -------------------- | -------------- | -------------------------------------------------- |
| `BOT_MODE`           | `polling`      | `polling` or `webhook`                             |
| `TELEGRAM_BOT_TOKEN` |                | Required unless dry run                            |
| `TELEGRAM_DRY_RUN`   | `false`        | Send nothing to Telegram, for load tests           |
| `DATABASE_URL`       |                | Required                                           |
| `DB_MAX_CONNS`       | `5`            |                                                    |
| `WEBHOOK_SECRET`     |                | Webhook mode, 32+ of `A-Z a-z 0-9 _ -`             |
| `WEBHOOK_PATH_TOKEN` |                | Webhook mode, 8+ of the same characters            |
| `PUBLIC_BASE_URL`    |                | Used for UPI payment links                         |
| `UPI_LINK_SECRET`    |                | 32+ characters, signs payment links                |
| `CRON_SECRET`        |                | 32+ characters, enables `/cron/*`                  |
| `PORT`               | `8080`         |                                                    |
| `LOG_LEVEL`          | `info`         |                                                    |
| `TZ_DISPLAY`         | `Asia/Kolkata` | Decides where days and months begin                |
| `MAX_EXPENSE_PAISE`  | `10000000`     | Largest single amount (₹1,00,000)                  |
| `METRICS_ENABLED`    | `false`        | Serves `/metrics` without authentication           |

## Architecture

```
Telegram ──POST /webhook/{token}──▶ httpserver ──▶ bot ──▶ ledger, budget ──▶ store ──▶ Postgres
scheduler ──POST /cron/{job}──────▶ httpserver ──▶ cron ──▶ outbox ──▶ telegram
browser ──GET /pay/{token}────────▶ httpserver (UPI redirect page)
```

| Package               | Responsibility                                  |
| --------------------- | ----------------------------------------------- |
| `cmd/bot`             | Wiring and graceful shutdown                    |
| `cmd/migrate`         | Applies the goose migrations                    |
| `internal/bot`        | Routes updates to handlers and sends replies    |
| `internal/parser`     | Text to command, no I/O                         |
| `internal/money`      | `Paise` type, formatting and equal splits       |
| `internal/ledger`     | Expenses, splits, balances, settlements, undo   |
| `internal/budget`     | Budgets and warnings                            |
| `internal/outbox`     | Queued notifications with rate limiting         |
| `internal/cron`       | Scheduled jobs                                  |
| `internal/store`      | Migrations, SQL queries and sqlc generated code |
| `internal/telegram`   | Small Bot API client, fake and dry run          |
| `internal/httpserver` | Routes and middleware                           |

## Design decisions

**Money is an integer.** Every amount is `money.Paise`, an `int64`. Parsing never
goes through a float. When an amount does not divide evenly, the spare paise go to
the participants with the lowest user IDs, so the result is reproducible.

**Updates take effect once.** Telegram delivers an update at least once. Each one
is handled in a single transaction that first inserts its `update_id` into
`processed_updates`. The marker and the effects commit together, and a repeat
delivery finds the marker and does nothing. If the transaction fails, the webhook
returns 500 and Telegram tries again.

Replies are sent after the commit. A reply that fails is logged and not retried,
because redelivering the update would be treated as a duplicate.

**The group ledger is append only.** There is no balance column to update, so two
splits at the same moment cannot overwrite each other. Balances are sums over
entries. Undo adds a reversal with negated entries, and a unique constraint on
`reverses_id` means a transaction can only be reversed once. Entries of a
transaction must sum to zero, which is checked in Go and again by a deferred
constraint trigger.

**Debt simplification is greedy.** The largest creditor is matched with the
largest debtor until everything is settled. That needs at most `n-1` payments for
`n` people. Finding the true minimum is NP-hard and not worth it at this size.

**A settle button works once.** Pressing it records the payment only if the same
payment is still suggested by the current balances, checked while holding a lock
on the chat.

**Webhook plus an external scheduler.** A scale to zero host only gives the
process CPU during a request, so nothing runs in the background. Updates are
handled inside the webhook request, and scheduled work is triggered over HTTP.
Notifications go to an `outbox` table first and are drained with a rate limiter.
Delivery there is at least once.

## Testing

```sh
make test              # unit tests
make test-integration  # needs Docker, runs against a real Postgres
make fuzz              # fuzzes the parser
make lint
```

Integration tests cover duplicate deliveries, 200 concurrent splits in one group,
double presses on buttons and concurrent outbox drains. Reply texts are compared
with golden files; refresh them with
`go test -tags=integration ./internal/bot -update`.

After changing SQL under `internal/store`, run `make sqlc` and commit the result.

## Load testing

Run the bot against a disposable database with `BOT_MODE=webhook` and
`TELEGRAM_DRY_RUN=true`, then:

```sh
k6 run -e WEBHOOK_PATH_TOKEN=... -e WEBHOOK_SECRET=... scripts/k6/webhook-load.js
```

`BASE_URL`, `VUS` and `DURATION` can be set the same way. The script writes real
rows, so never point it at production data.

A local run on an Apple M2, with the bot on the host, Postgres 16 and k6 in
Docker, `DB_MAX_CONNS=5`, dry run, 20 virtual users for 30 seconds:

| Requests | Req/s | p50     | p95     | p99     | Failed | Rows written |
| -------- | ----- | ------- | ------- | ------- | ------ | ------------ |
| 42,598   | 1,419 | 13.7 ms | 16.9 ms | 20.8 ms | 0      | 42,598       |

Replies were not sent to Telegram, so these numbers say nothing about a deployed
instance.

## Deploying

Production runs on Render's free plan with Neon, which needs no payment card.
See [deploy/render.md](deploy/render.md). To use Cloud Run instead, which does
need a billing account, see [deploy/cloudrun.md](deploy/cloudrun.md).

## Known limitations

- When Telegram upgrades a group to a supergroup the chat gets a new ID, and the
  bot starts a fresh ledger for it.
- In groups the bot reads commands only.
- A bot cannot look up people by username, so someone must have sent a command in
  the group before they can be mentioned in a split.
- Amounts are in rupees only.
