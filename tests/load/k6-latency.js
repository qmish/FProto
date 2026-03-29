import ws from "k6/ws";
import { check, sleep } from "k6";
import { Trend, Counter } from "k6/metrics";

const e2eLatency = new Trend("e2e_latency_ms");
const totalMessages = new Counter("total_messages");

const GATEWAY_URL = __ENV.GATEWAY_URL || "ws://localhost:8080/ws";

export const options = {
  scenarios: {
    steady_state: {
      executor: "constant-arrival-rate",
      rate: 1000,
      timeUnit: "1s",
      duration: "5m",
      preAllocatedVUs: 500,
      maxVUs: 2000,
    },
  },
  thresholds: {
    e2e_latency_ms: ["p(50)<20", "p(95)<100", "p(99)<200"],
  },
};

export default function () {
  const res = ws.connect(GATEWAY_URL, {}, function (socket) {
    socket.on("message", function (data) {
      try {
        const msg = JSON.parse(data);
        if (msg.ts) {
          const latency = Date.now() - msg.ts;
          e2eLatency.add(latency);
        }
      } catch (_e) {
        // echo mode: measure from send
      }
    });

    const msg = JSON.stringify({ ts: Date.now(), seq: __VU });
    socket.send(msg);
    totalMessages.add(1);

    sleep(1);
    socket.close();
  });

  check(res, {
    "connected": (r) => r && r.status === 101,
  });
}
