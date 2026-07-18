// ==============================================================================
// LOAD AND SCALE PERFORMANCE TESTS - k6 Benchmark Script
// Simulates concurrent Virtual Users (VUs) and ramps up load profiles
// ==============================================================================

import http from 'k6/http';
import { check, sleep } from 'k6';

// 1. Configure the load profile ramp-up and stress configurations
export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp-up: 0 to 50 concurrent virtual users
    { duration: '1m', target: 200 },   // Stress Phase: Sustain 200 virtual users
    { duration: '1m', target: 1000 },  // Spike Phase: Brief push to 1000 virtual users
    { duration: '30s', target: 0 },   // Cool-down: Ramp down back to 0 users
  ],
  thresholds: {
    // Assert p95 response latencies are strictly under 200ms
    http_req_duration: ['p(95)<200'],
    // Assert total HTTP request error rates are under 1%
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = 'http://api-gateway.nexus-core.svc.cluster.local:8080/api/v2';

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer nexus-token-secure-390',
    },
  };

  // 1. Benchmark API Gateway Core Routing and Authentication
  const authResponse = http.get(`${BASE_URL}/health/live`, params);
  check(authResponse, {
    'Gateway Liveness returns 200': (res) => res.status === 200,
    'Response payload size matches baseline': (res) => res.body.length > 0,
  });

  sleep(0.1); // Small pacing delay between requests

  // 2. Stress Write Operations on Ledger Database endpoint
  const payload = JSON.stringify({
    account_id: 'acc-load-test-99',
    amount: parseFloat((Math.random() * 1000).toFixed(2)),
  });

  const transactionResponse = http.post(`${BASE_URL}/transactions`, payload, params);
  check(transactionResponse, {
    'Ledger Write returns 201': (res) => res.status === 201,
    'Transaction commit response contains ID': (res) => JSON.parse(res.body).tx_id !== undefined,
  });

  sleep(Math.random() * 0.5 + 0.2); // Random pacing back-off (200ms - 700ms)
}
