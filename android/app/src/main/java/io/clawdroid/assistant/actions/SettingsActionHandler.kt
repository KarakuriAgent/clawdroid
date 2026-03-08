package io.clawdroid.assistant.actions

import android.content.Context
import android.content.Intent
import android.provider.Settings
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class SettingsActionHandler : ActionHandler {
    override val supportedActions = setOf("open_settings")

    private val settingsMap = mapOf(
        "main" to Settings.ACTION_SETTINGS,
        "wifi" to Settings.ACTION_WIFI_SETTINGS,
        "bluetooth" to Settings.ACTION_BLUETOOTH_SETTINGS,
        "airplane" to Settings.ACTION_AIRPLANE_MODE_SETTINGS,
        "display" to Settings.ACTION_DISPLAY_SETTINGS,
        "sound" to Settings.ACTION_SOUND_SETTINGS,
        "battery" to Settings.ACTION_BATTERY_SAVER_SETTINGS,
        "apps" to Settings.ACTION_APPLICATION_SETTINGS,
        "location" to Settings.ACTION_LOCATION_SOURCE_SETTINGS,
        "security" to Settings.ACTION_SECURITY_SETTINGS,
        "accessibility" to Settings.ACTION_ACCESSIBILITY_SETTINGS,
        "date_time" to Settings.ACTION_DATE_SETTINGS,
        "language" to Settings.ACTION_LOCALE_SETTINGS,
        "developer" to Settings.ACTION_APPLICATION_DEVELOPMENT_SETTINGS,
        "about" to Settings.ACTION_DEVICE_INFO_SETTINGS,
        "notification" to Settings.ACTION_APP_NOTIFICATION_SETTINGS,
        "mobile_data" to Settings.ACTION_DATA_ROAMING_SETTINGS,
        "nfc" to Settings.ACTION_NFC_SETTINGS,
        "privacy" to Settings.ACTION_PRIVACY_SETTINGS,
    )

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        val section = request.stringParam("section") ?: "main"
        val action = settingsMap[section] ?: Settings.ACTION_SETTINGS

        val intent = Intent(action).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }

        return try {
            context.startActivity(intent)
            ToolResponse(request.requestId, true, result = "Settings opened: $section")
        } catch (e: Exception) {
            // Fallback to main settings if specific section fails
            try {
                context.startActivity(Intent(Settings.ACTION_SETTINGS).apply {
                    addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                })
                ToolResponse(request.requestId, true, result = "Settings opened (fallback to main)")
            } catch (e2: Exception) {
                ToolResponse(request.requestId, false, error = "Failed to open settings: ${e2.message}")
            }
        }
    }
}
