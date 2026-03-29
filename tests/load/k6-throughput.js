import ws from "k6/ws";
import { check } from "k6";
import { Counter, Trend } from "k6/metrics";

const messagesSent = new Counter("messages_sent_total");
const messagesReceived = new Counter("messages_received_total");
const echoRtt = new Trend("echo_rtt_ms");

const GATEWAY_URL = __ENV.GATEWAY_URL || "ws://localhost:8080/ws";
const PAYLOAD_SIZE = parseInt(__ENV.PAYLOAD_SIZE || "100");
const MESSAGES_PER_VU = parseInt(__ENV.MESSAGES_PER_VU || "100");

export const options = {
  scenarios: {
    payload_100b: {
      executor: "constant-vus",
      vus: 100,
      duration: "2m",
      env: { PAYLOAD_SIZE: "100" },
      tags: { payload: "100B" },
    },
    payload_1kb: {
      executor: "constant-vus",
      vus: 100,
      duration: "2m",
      startTime: "3m",
      env: { PAYLOAD_SIZE: "1024" },
      tags: { payload: "1KB" },
    },
    payload_10kb: {
      executor: "constant-vus",
      vus: 50,
      duration: "2m",
      startTime: "6m",
      env: { PAYLOAD_SIZE: "10240" },
      tags: { payload: "10KB" },
    },
  },
  thresholds: {
    echo_rtt_ms: ["p(95)<200", "p(99)<500"],
    messages_sent_total: ["count>0"],
  },
};

function generatePayload(size) {
  let payload = "";
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  for (let i = 0; i < size; i++) {
    payload += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return payload;
}

export default function () {
  const payload = generatePayload(PAYLOAD_SIZE);

  const res = ws.connect(GATEWAY_URL, {}, function (socket) {
    let received = 0;

    socket.on("message", function (_data) {
      received++;
      messagesReceived.add(1);
    });

    for (let i = 0; i < MESSAGES_PER_VU; i++) {
      const sendTime = Date.now();
      socket.send(payload);
      messagesSent.add(1);

      socket.on("message", function () {
        echoRtt.add(Date.now() - sendTime);
      });
    }

    // Wait for responses
    socket.setTimeout(function () {
      socket.close();
    }, 10000);
  });

  check(res, {
    "WS connected": (r) => r && r.status === 101,
  });
}
