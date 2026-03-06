package io.clawdroid.receiver

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import androidx.core.app.NotificationCompat
import io.clawdroid.backend.api.R

object NotificationHelper {

    private const val CHANNEL_ID = "clawdroid_messages"
    private const val CHANNEL_NAME = "Agent Messages"
    private const val NOTIFICATION_ID = 1001

    const val ASSISTANT_CHANNEL_ID = "clawdroid_assistant"
    private const val ASSISTANT_CHANNEL_NAME = "Assistant"

    const val GATEWAY_CHANNEL_ID = "clawdroid_gateway"
    private const val GATEWAY_CHANNEL_NAME = "Gateway"

    fun createNotificationChannel(context: Context) {
        val manager = context.getSystemService(NotificationManager::class.java)

        val messageChannel = NotificationChannel(
            CHANNEL_ID,
            CHANNEL_NAME,
            NotificationManager.IMPORTANCE_DEFAULT
        ).apply {
            description = "Messages from ClawDroid agent"
        }
        manager.createNotificationChannel(messageChannel)

        val assistantChannel = NotificationChannel(
            ASSISTANT_CHANNEL_ID,
            ASSISTANT_CHANNEL_NAME,
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Assistant overlay service"
            setShowBadge(false)
        }
        manager.createNotificationChannel(assistantChannel)

        val gatewayChannel = NotificationChannel(
            GATEWAY_CHANNEL_ID,
            GATEWAY_CHANNEL_NAME,
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Gateway backend service"
            setShowBadge(false)
        }
        manager.createNotificationChannel(gatewayChannel)
    }

    fun showMessageNotification(context: Context, content: String) {
        val launchIntent = context.packageManager.getLaunchIntentForPackage(context.packageName)
        val pendingIntent = PendingIntent.getActivity(
            context, 0, launchIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle("ClawDroid")
            .setContentText(content.take(200))
            .setStyle(NotificationCompat.BigTextStyle().bigText(content.take(1000)))
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .build()

        val manager = context.getSystemService(NotificationManager::class.java)
        manager.notify(NOTIFICATION_ID, notification)
    }
}
