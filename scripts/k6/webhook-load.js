import http from "k6/http";
import exec from "k6/execution";
import { check } from "k6";

const url = `${__ENV.BASE_URL || "http://localhost:8080"}/webhook/${__ENV.WEBHOOK_PATH_TOKEN}`;
const firstUpdateId = Date.now() * 1000;

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || "30s",
  summaryTrendStats: ["avg", "p(50)", "p(95)", "p(99)", "max"],
  thresholds: {
    http_req_failed: ["rate<0.01"],
  },
};

export function setup() {
  return { firstUpdateId };
}

export default function (data) {
  const userId = 1000 + exec.vu.idInTest;
  const updateId = data.firstUpdateId + exec.scenario.iterationInTest;
  const update = {
    update_id: updateId,
    message: {
      message_id: exec.scenario.iterationInTest + 1,
      from: { id: userId, is_bot: false, first_name: `load${userId}` },
      chat: { id: userId, type: "private" },
      text: "120 chai",
    },
  };
  const res = http.post(url, JSON.stringify(update), {
    headers: {
      "Content-Type": "application/json",
      "X-Telegram-Bot-Api-Secret-Token": __ENV.WEBHOOK_SECRET,
    },
  });
  check(res, { "status is 200": (r) => r.status === 200 });
}
