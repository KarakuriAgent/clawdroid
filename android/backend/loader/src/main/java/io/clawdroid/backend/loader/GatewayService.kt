package io.clawdroid.backend.loader

import android.app.Notification
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import androidx.core.app.NotificationCompat
import androidx.lifecycle.LifecycleService
import io.clawdroid.backend.api.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import org.koin.android.ext.android.inject

class GatewayService : LifecycleService() {

    private val processManager: GatewayProcessManager by inject()
    private lateinit var serviceScope: CoroutineScope

    override fun onCreate() {
        super.onCreate()
        serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    }

    @android.annotation.SuppressLint("InlinedApi")
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)

        when (intent?.action) {
            ACTION_PAUSE -> {
                serviceScope.launch {
                    processManager.stop()
                    updateNotification(running = false)
                }
            }
            ACTION_RESUME -> {
                serviceScope.launch {
                    processManager.start()
                    updateNotification(running = true)
                }
            }
            else -> {
                startForeground(
                    NOTIFICATION_ID,
                    buildNotification(running = true),
                    ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
                )
                serviceScope.launch { processManager.start() }
            }
        }

        return START_STICKY
    }

    override fun onDestroy() {
        runBlocking { processManager.stop() }
        serviceScope.cancel()
        super.onDestroy()
    }

    private fun updateNotification(running: Boolean) {
        val manager = getSystemService(NOTIFICATION_SERVICE) as android.app.NotificationManager
        manager.notify(NOTIFICATION_ID, buildNotification(running))
    }

    private fun buildNotification(running: Boolean): Notification {
        val actionIntent = Intent(this, GatewayService::class.java).apply {
            action = if (running) ACTION_PAUSE else ACTION_RESUME
        }
        val pendingIntent = PendingIntent.getService(
            this, 0, actionIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val actionLabel = if (running) "Pause" else "Resume"
        val actionIcon = if (running) android.R.drawable.ic_media_pause else android.R.drawable.ic_media_play
        val contentText = if (running) "Backend running" else "Backend paused"

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle("ClawDroid")
            .setContentText(contentText)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOngoing(true)
            .addAction(actionIcon, actionLabel, pendingIntent)
            .build()
    }

    companion object {
        private const val NOTIFICATION_ID = 2002
        private const val CHANNEL_ID = "clawdroid_gateway"
        const val ACTION_PAUSE = "io.clawdroid.backend.loader.ACTION_PAUSE"
        const val ACTION_RESUME = "io.clawdroid.backend.loader.ACTION_RESUME"
    }
}
