package io.clawdroid.assistant.actions

import android.content.Context
import android.content.Intent
import android.provider.AlarmClock
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class AlarmActionHandler : ActionHandler {
    override val supportedActions = setOf("set_alarm", "set_timer", "dismiss_alarm", "show_alarms")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "set_alarm" -> handleSetAlarm(request, context)
            "set_timer" -> handleSetTimer(request, context)
            "dismiss_alarm" -> handleDismissAlarm(request, context)
            "show_alarms" -> handleShowAlarms(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown alarm action: ${request.action}")
        }
    }

    private fun handleSetAlarm(request: ToolRequest, context: Context): ToolResponse {
        val hour = request.intParam("hour")
            ?: return ToolResponse(request.requestId, false, error = "hour required")
        val minute = request.intParam("minute")
            ?: return ToolResponse(request.requestId, false, error = "minute required")

        val intent = Intent(AlarmClock.ACTION_SET_ALARM).apply {
            putExtra(AlarmClock.EXTRA_HOUR, hour)
            putExtra(AlarmClock.EXTRA_MINUTES, minute)
            request.stringParam("message")?.let {
                putExtra(AlarmClock.EXTRA_MESSAGE, it)
            }
            val skipUi = request.boolParam("skip_ui") ?: true
            putExtra(AlarmClock.EXTRA_SKIP_UI, skipUi)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }

        return launchActivity(request, context, intent, "Alarm set for %02d:%02d".format(hour, minute))
    }

    private fun handleSetTimer(request: ToolRequest, context: Context): ToolResponse {
        val duration = request.intParam("duration_seconds")
            ?: return ToolResponse(request.requestId, false, error = "duration_seconds required")

        val intent = Intent(AlarmClock.ACTION_SET_TIMER).apply {
            putExtra(AlarmClock.EXTRA_LENGTH, duration)
            request.stringParam("message")?.let {
                putExtra(AlarmClock.EXTRA_MESSAGE, it)
            }
            val skipUi = request.boolParam("skip_ui") ?: true
            putExtra(AlarmClock.EXTRA_SKIP_UI, skipUi)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }

        return launchActivity(request, context, intent, "Timer set for $duration seconds")
    }

    private fun handleDismissAlarm(request: ToolRequest, context: Context): ToolResponse {
        val intent = Intent(AlarmClock.ACTION_DISMISS_ALARM).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Alarm dismissed")
    }

    private fun handleShowAlarms(request: ToolRequest, context: Context): ToolResponse {
        val intent = Intent(AlarmClock.ACTION_SHOW_ALARMS).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Showing alarms")
    }
}
