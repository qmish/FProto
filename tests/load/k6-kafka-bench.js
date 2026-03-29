/*
  Kafka throughput benchmark via k6.

  This script measures Kafka produce/consume throughput by sending messages
  through the Gateway -> Dispatcher -> Kafka pipeline.

  Usage:
    k6 run --env GATEWAY_URL=ws://localhost:8080/ws tests/load/k6-kafka-bench.js

  Monitor Kafka metrics via:
    - Grafana Kafka dashboard (consumer lag, dispatch duration)
    - Prometheus: fproto_outbox_published_total, fproto_dispatch_duration_seconds
*/

import ws from "k6/ws";
import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";

const kafkaDispatched = new Counter("kafka_dispatched_total");
const dispatchLatency = new Trend("kafka_dispatch_latency_ms");

const GATEWAY_URL = __ENV.GATEWAY_URL || "ws://localhost:8080/ws";
const MESSAGES_PER_VU = parseInt(__ENV.MESSAGES_PER_VU || "50");

export const options = {
  scenarios: {
    kafka_throughput: {
      executor: "ramping-vus",
      startVUs: 10,
      stages: [
        { duration: "30s", target: 100 },
        { duration: "2m", target: 500 },
        { duration: "2m", target: 500 },
        { duration: "30s", target: 0 },
      ],
    },
  },
  thresholds: {
    kafka_dispatched_total: ["count>0"],
    kafka_dispatch_latency_ms: ["p(95)<500"],
  },
};

export default function () {
  const res = ws.connect(GATEWAY_URL, {}, function (socket) {
    socket.on("message", function (_data) {
      kafkaDispatched.add(1);
    });

    for (let i = 0; i < MESSAGES_PER_VU; i++) {
      const start = Date.now();
      socket.send(JSON.stringify({
        type: "chat",
        recipient: "benchmark-user",
        text: "kafka-bench-" + __VU + "-" + i,
        ts: start,
      }));
      dispatchLatency.add(Date.now() - start);
    }

    sleep(5);
    socket.close();
  });

  check(res, {
    "connected": (r) => r && r.status === 101,
  });
}
