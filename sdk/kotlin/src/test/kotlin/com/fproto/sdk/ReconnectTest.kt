package com.fproto.sdk

import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class ReconnectTest {

    @Test
    fun `default config values`() {
        val cfg = ReconnectConfig()
        assertEquals(1000, cfg.initialDelayMs)
        assertEquals(30000, cfg.maxDelayMs)
        assertEquals(2.0, cfg.multiplier)
    }

    @Test
    fun `compute delay attempt 0`() {
        val cfg = ReconnectConfig()
        val delay = computeDelay(0, cfg)
        assertTrue(delay in 800..1200, "Expected ~1000ms, got $delay")
    }

    @Test
    fun `compute delay capped at max`() {
        val cfg = ReconnectConfig()
        val delay = computeDelay(10, cfg)
        assertTrue(delay <= 36000, "Expected <= 36000ms (max+jitter), got $delay")
    }

    @Test
    fun `compute delay increases`() {
        val cfg = ReconnectConfig(jitter = 0.0)
        val d0 = computeDelay(0, cfg)
        val d1 = computeDelay(1, cfg)
        val d2 = computeDelay(2, cfg)
        assertTrue(d1 > d0, "Expected d1 > d0")
        assertTrue(d2 > d1, "Expected d2 > d1")
    }
}
