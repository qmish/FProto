package com.fproto.sdk

import kotlin.math.min
import kotlin.math.pow
import kotlin.random.Random

data class ReconnectConfig(
    val initialDelayMs: Long = 1000,
    val maxDelayMs: Long = 30000,
    val multiplier: Double = 2.0,
    val jitter: Double = 0.2,
    val maxAttempts: Int = 0
)

/**
 * Compute reconnect delay for a given attempt with exponential backoff and jitter.
 */
fun computeDelay(attempt: Int, config: ReconnectConfig): Long {
    var delay = config.initialDelayMs * config.multiplier.pow(attempt.toDouble())
    delay = min(delay, config.maxDelayMs.toDouble())
    if (config.jitter > 0) {
        val delta = delay * config.jitter
        delay = delay - delta + Random.nextDouble() * 2 * delta
    }
    return delay.toLong()
}

/**
 * Connect with automatic reconnection using exponential backoff.
 */
fun FProtoClient.connectWithReconnect(config: ReconnectConfig) {
    var attempt = 0
    while (true) {
        try {
            connect()
            return
        } catch (e: Exception) {
            attempt++
            if (config.maxAttempts > 0 && attempt >= config.maxAttempts) {
                throw e
            }
            Thread.sleep(computeDelay(attempt - 1, config))
        }
    }
}
