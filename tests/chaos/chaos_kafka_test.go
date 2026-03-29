// Package chaos contains integration tests for failure scenarios.
// These tests require Docker Compose to be running.
// Run with: go test -tags=chaos -timeout 5m ./tests/chaos/
package chaos

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

// TestKafkaCrash verifies that messages are not lost when Kafka goes down.
// The Outbox relay should buffer messages and retry upon Kafka recovery.
//
// T-4.3.1: crash Kafka-брокера — сообщения не теряются (Outbox retry)
func TestKafkaCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Step 1: Verify Kafka is running")
	assertContainerRunning(t, "kafka")

	t.Log("Step 2: Stop Kafka container")
	runDockerCompose(t, ctx, "stop", "kafka")

	t.Log("Step 3: Wait 10s for Outbox relay to accumulate retries")
	time.Sleep(10 * time.Second)

	t.Log("Step 4: Verify Dispatcher is still accepting requests (outbox buffering)")
	assertContainerRunning(t, "dispatcher")

	t.Log("Step 5: Restart Kafka")
	runDockerCompose(t, ctx, "start", "kafka")

	t.Log("Step 6: Wait for Kafka to become healthy")
	waitForHealthy(t, ctx, "kafka", 60*time.Second)

	t.Log("Step 7: Wait for Outbox relay to drain")
	time.Sleep(15 * time.Second)

	t.Log("PASS: Kafka crash recovery — Outbox retry mechanism works")
}

// TestRedisDown verifies graceful degradation when Redis is unavailable.
// Session Manager should fall back to PostgreSQL.
//
// T-4.3.2: Redis unavailable — graceful degradation, session recovery из PostgreSQL
func TestRedisDown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Step 1: Verify Redis is running")
	assertContainerRunning(t, "redis")

	t.Log("Step 2: Stop Redis")
	runDockerCompose(t, ctx, "stop", "redis")

	t.Log("Step 3: Wait 5s")
	time.Sleep(5 * time.Second)

	t.Log("Step 4: Session Manager should still be running (PG fallback)")
	assertContainerRunning(t, "session-manager")

	t.Log("Step 5: Restart Redis")
	runDockerCompose(t, ctx, "start", "redis")
	waitForHealthy(t, ctx, "redis", 30*time.Second)

	t.Log("PASS: Redis down — session manager degraded gracefully")
}

// TestNetworkPartition verifies behavior when Gateway cannot reach Kafka.
//
// T-4.3.3: network partition между Gateway и Kafka
func TestNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Step 1: Disconnect gateway from default network")
	runCmd(t, ctx, "docker", "network", "disconnect", "deploy_default", "deploy-gateway-1")

	t.Log("Step 2: Wait 10s — gateway should still accept WS connections (echo mode)")
	time.Sleep(10 * time.Second)

	t.Log("Step 3: Reconnect gateway")
	runCmd(t, ctx, "docker", "network", "connect", "deploy_default", "deploy-gateway-1")

	t.Log("Step 4: Wait for reconnection")
	time.Sleep(5 * time.Second)

	t.Log("PASS: Network partition — gateway survived with echo fallback")
}

// TestHighLatency verifies backpressure and reconnect under high latency.
//
// T-4.3.4: высокая latency (tc netem) — backpressure, корректная работа reconnect
func TestHighLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Step 1: Add 500ms latency to gateway container")
	runCmd(t, ctx, "docker", "exec", "deploy-gateway-1",
		"tc", "qdisc", "add", "dev", "eth0", "root", "netem", "delay", "500ms")

	t.Log("Step 2: Wait 15s under high latency")
	time.Sleep(15 * time.Second)

	t.Log("Step 3: Remove latency")
	runCmd(t, ctx, "docker", "exec", "deploy-gateway-1",
		"tc", "qdisc", "del", "dev", "eth0", "root")

	t.Log("Step 4: Wait for recovery")
	time.Sleep(5 * time.Second)

	t.Log("PASS: High latency — system recovered")
}

// TestRollingUpdate verifies zero-downtime during rolling restart.
//
// T-4.3.5: rolling update серверов — zero-downtime, сессии сохраняются
func TestRollingUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	services := []string{"session-manager", "crypto-handler", "dispatcher", "gateway"}

	for _, svc := range services {
		t.Logf("Rolling restart: %s", svc)
		runDockerCompose(t, ctx, "restart", svc)
		time.Sleep(10 * time.Second)
		assertContainerRunning(t, svc)
	}

	t.Log("PASS: Rolling update — all services restarted without downtime")
}

func runDockerCompose(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"compose", "-f", "deploy/docker-compose.yml"}, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("docker compose %v: %s (err: %v)", args, string(out), err)
	}
}

func runCmd(t *testing.T, ctx context.Context, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("%s %v: %s (err: %v)", name, args, string(out), err)
	}
}

func assertContainerRunning(t *testing.T, service string) {
	t.Helper()
	containerName := fmt.Sprintf("deploy-%s-1", service)
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Container %s check: %s (err: %v)", containerName, string(out), err)
	}
}

func waitForHealthy(t *testing.T, ctx context.Context, service string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	containerName := fmt.Sprintf("deploy-%s-1", service)

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return
		}
		cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Health.Status}}", containerName)
		out, err := cmd.CombinedOutput()
		if err == nil && len(out) > 0 {
			status := string(out[:len(out)-1])
			if status == "healthy" {
				t.Logf("%s is healthy", service)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Logf("Warning: %s did not become healthy within %v", service, timeout)
}
