import ws from "k6/ws";
import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";

const wsConnections = new Counter("ws_connections_total");
const wsHandshakeDuration = new Trend("ws_handshake_duration_ms");

const GATEWAY_URL = __ENV.GATEWAY_URL || "ws://localhost:8080/ws";

export const options = {
  scenarios: {
    ramp_10k: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 1000 },
        { duration: "1m", target: 10000 },
        { duration: "2m", target: 10000 },
        { duration: "30s", target: 0 },
      ],
      tags: { scenario: "10k" },
    },
    ramp_50k: {
      executor: "ramping-vus",
      startVUs: 0,
      startTime: "5m",
      stages: [
        { duration: "1m", target: 10000 },
        { duration: "2m", target: 50000 },
        { duration: "3m", target: 50000 },
        { duration: "1m", target: 0 },
      ],
      tags: { scenario: "50k" },
    },
    ramp_100k: {
      executor: "ramping-vus",
      startVUs: 0,
      startTime: "13m",
      stages: [
        { duration: "2m", target: 50000 },
        { duration: "3m", target: 100000 },
        { duration: "5m", target: 100000 },
        { duration: "2m", target: 0 },
      ],
      tags: { scenario: "100k" },
    },
  },
  thresholds: {
    ws_handshake_duration_ms: ["p(95)<500"],
    ws_connections_total: ["count>0"],
  },
};

export default function () {
  const startTime = Date.now();

  const res = ws.connect(GATEWAY_URL, {}, function (socket) {
    const handshakeMs = Date.now() - startTime;
    wsHandshakeDuration.add(handshakeMs);
    wsConnections.add(1);

    socket.on("message", function (_data) {});

    socket.on("error", function (e) {
      console.error("WS error: " + e.error());
    });

    // Hold connection open
    sleep(5 + Math.random() * 10);
    socket.close();
  });

  check(res, {
    "WS status is 101": (r) => r && r.status === 101,
  });
}
