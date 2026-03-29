package com.fproto.sdk

import okhttp3.*
import okio.ByteString
import okio.ByteString.Companion.toByteString
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit

/**
 * FProto Kotlin SDK client with OkHttp WebSocket transport.
 */
class FProtoClient(
    private val config: ClientConfig
) {
    private var ws: WebSocket? = null
    private val messageQueue = LinkedBlockingQueue<ByteArray>()
    @Volatile
    var isConnected: Boolean = false
        private set

    fun connect() {
        val client = OkHttpClient.Builder()
            .readTimeout(30, TimeUnit.SECONDS)
            .build()

        val request = Request.Builder()
            .url(config.serverUrl)
            .build()

        val latch = java.util.concurrent.CountDownLatch(1)
        var connectError: Throwable? = null

        ws = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                isConnected = true
                latch.countDown()
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                messageQueue.offer(bytes.toByteArray())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                messageQueue.offer(text.toByteArray())
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                isConnected = false
                connectError = t
                latch.countDown()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                isConnected = false
            }
        })

        latch.await(10, TimeUnit.SECONDS)
        if (connectError != null) {
            throw connectError!!
        }
    }

    fun send(data: ByteArray) {
        val socket = ws ?: throw IllegalStateException("Not connected")
        socket.send(data.toByteString())
    }

    fun receive(timeoutMs: Long = 5000): ByteArray {
        return messageQueue.poll(timeoutMs, TimeUnit.MILLISECONDS)
            ?: throw RuntimeException("Receive timeout")
    }

    fun close() {
        ws?.close(1000, "close")
        ws = null
        isConnected = false
    }
}

data class ClientConfig(
    val serverUrl: String
)
