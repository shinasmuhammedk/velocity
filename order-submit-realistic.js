// k6 load test for POST /api/orders — "realistic" profile.
//
// This profile deliberately stays within the API's own per-user
// rate limit (rate_limit.submit_rate / submit_burst in
// configs/config.<env>.yaml — 10 req/s sustained, burst 20 by
// default). All traffic here uses a single JWT, i.e. a single user,
// so the rate limiter's token bucket is shared across every VU.
//
// Use this profile to answer: "what does a real, well-behaved client
// experience against the API as it's actually configured today?"
// If you want to instead find the ceiling of the HTTP + engine
// pipeline itself, see order-submit-stress.js, which requires
// temporarily raising the rate limit config.
//
// Run:
//   k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=<jwt> order-submit-realistic.js
//
// TOKEN must be a valid JWT for an existing user (see how you obtained
// one earlier in this project — the identity service, external to
// this repo, issues these). Tokens observed so far expire in ~15
// minutes, so keep runs short or refresh the token for longer runs.

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN;
const SYMBOL = __ENV.SYMBOL || 'BTCUSDT';

if (!TOKEN) {
  throw new Error('Set -e TOKEN=<your JWT> — see the project chat for how one was obtained.');
}

// Deliberately conservative: 8 VUs, each pacing itself to roughly
// 1 request/sec via the sleep(1) below, for ~8 req/s aggregate —
// comfortably inside the 10 req/s + burst-20 bucket shared by this
// single user, so this measures real latency under normal load
// rather than mostly measuring 429 responses.
export const options = {
  vus: 8,
  duration: '30s',
  thresholds: {
    // Under this profile, virtually nothing should be rate-limited
    // or fail outright — if this threshold fails, something other
    // than the rate limiter is wrong.
    http_req_failed: ['rate<0.01'],
    submit_latency: ['p(95)<500', 'p(99)<1000'],
  },
};

const submitLatency = new Trend('submit_latency', true);
const rateLimited = new Counter('rate_limited_responses');

export default function () {
  // Random low price so orders rest without crossing, spreading
  // across several price levels rather than piling one level deep —
  // similar spirit to the in-process engine load test's non-crossing
  // workload.
  const price = 90 + Math.floor(Math.random() * 20);

  const payload = JSON.stringify({
    symbol: SYMBOL,
    side: 'BUY',
    type: 'LIMIT',
    time_in_force: 'GTC',
    price: price,
    stop_price: 0,
    quantity: 1,
  });

  const res = http.post(`${BASE_URL}/api/orders`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${TOKEN}`,
    },
  });

  submitLatency.add(res.timings.duration);

  if (res.status === 429) {
    rateLimited.add(1);
  }

  check(res, {
    'status is 201': (r) => r.status === 201,
    'body has success:true': (r) => {
      try {
        return JSON.parse(r.body).success === true;
      } catch (e) {
        return false;
      }
    },
  });

  sleep(1);
}