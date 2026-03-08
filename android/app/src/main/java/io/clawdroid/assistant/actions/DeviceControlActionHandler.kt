package io.clawdroid.assistant.actions

import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.hardware.camera2.CameraManager
import android.media.AudioManager
import android.net.Uri
import android.provider.Settings
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class DeviceControlActionHandler : ActionHandler {
    override val supportedActions = setOf("flashlight", "set_volume", "set_ringer_mode", "set_dnd", "set_brightness")

    override fun requiredPermissions(action: String): List<PermissionRequirement> = when (action) {
        "set_dnd", "set_ringer_mode" -> listOf(
            PermissionRequirement.Special(
                check = {
                    (it.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
                        .isNotificationPolicyAccessGranted
                },
                settingsIntent = Intent(Settings.ACTION_NOTIFICATION_POLICY_ACCESS_SETTINGS),
                description = "Do Not Disturb access"
            )
        )
        "set_brightness" -> listOf(
            PermissionRequirement.Special(
                check = { Settings.System.canWrite(it) },
                settingsIntent = Intent(
                    Settings.ACTION_MANAGE_WRITE_SETTINGS,
                    Uri.parse("package:io.clawdroid")
                ),
                description = "Modify system settings"
            )
        )
        else -> emptyList()
    }

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "flashlight" -> handleFlashlight(request, context)
            "set_volume" -> handleSetVolume(request, context)
            "set_ringer_mode" -> handleSetRingerMode(request, context)
            "set_dnd" -> handleSetDnd(request, context)
            "set_brightness" -> handleSetBrightness(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown device control action")
        }
    }

    private fun handleFlashlight(request: ToolRequest, context: Context): ToolResponse {
        val enabled = request.boolParam("enabled")
            ?: return ToolResponse(request.requestId, false, error = "enabled required")

        return try {
            val cameraManager = context.getSystemService(Context.CAMERA_SERVICE) as CameraManager
            val cameraId = cameraManager.cameraIdList.firstOrNull()
                ?: return ToolResponse(request.requestId, false, error = "No camera available for flashlight")
            cameraManager.setTorchMode(cameraId, enabled)
            val state = if (enabled) "on" else "off"
            ToolResponse(request.requestId, true, result = "Flashlight turned $state")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to toggle flashlight: ${e.message}")
        }
    }

    private fun handleSetVolume(request: ToolRequest, context: Context): ToolResponse {
        val stream = request.stringParam("stream")
            ?: return ToolResponse(request.requestId, false, error = "stream required")
        val level = request.intParam("level")
            ?: return ToolResponse(request.requestId, false, error = "level required")

        val streamType = when (stream) {
            "music" -> AudioManager.STREAM_MUSIC
            "ring" -> AudioManager.STREAM_RING
            "notification" -> AudioManager.STREAM_NOTIFICATION
            "alarm" -> AudioManager.STREAM_ALARM
            "system" -> AudioManager.STREAM_SYSTEM
            else -> return ToolResponse(request.requestId, false, error = "Invalid stream: $stream")
        }

        return try {
            val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
            val maxVolume = audioManager.getStreamMaxVolume(streamType)
            val clampedLevel = level.coerceIn(0, maxVolume)
            audioManager.setStreamVolume(streamType, clampedLevel, 0)
            ToolResponse(request.requestId, true, result = "Volume set: $stream = $clampedLevel/$maxVolume")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to set volume: ${e.message}")
        }
    }

    private fun handleSetRingerMode(request: ToolRequest, context: Context): ToolResponse {
        val mode = request.stringParam("mode")
            ?: return ToolResponse(request.requestId, false, error = "mode required")

        val ringerMode = when (mode) {
            "normal" -> AudioManager.RINGER_MODE_NORMAL
            "vibrate" -> AudioManager.RINGER_MODE_VIBRATE
            "silent" -> AudioManager.RINGER_MODE_SILENT
            else -> return ToolResponse(request.requestId, false, error = "Invalid mode: $mode")
        }

        return try {
            val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
            audioManager.ringerMode = ringerMode
            ToolResponse(request.requestId, true, result = "Ringer mode set to: $mode")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Cannot change ringer mode. DND access may be required.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to set ringer mode: ${e.message}")
        }
    }

    private fun handleSetDnd(request: ToolRequest, context: Context): ToolResponse {
        val enabled = request.boolParam("enabled")
            ?: return ToolResponse(request.requestId, false, error = "enabled required")

        return try {
            val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            if (!notificationManager.isNotificationPolicyAccessGranted) {
                return ToolResponse(request.requestId, false, error = "DND access not granted. Please enable in Settings > Do Not Disturb access.")
            }
            val filter = if (enabled) {
                NotificationManager.INTERRUPTION_FILTER_PRIORITY
            } else {
                NotificationManager.INTERRUPTION_FILTER_ALL
            }
            notificationManager.setInterruptionFilter(filter)
            val state = if (enabled) "enabled" else "disabled"
            ToolResponse(request.requestId, true, result = "Do Not Disturb $state")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to set DND: ${e.message}")
        }
    }

    private fun handleSetBrightness(request: ToolRequest, context: Context): ToolResponse {
        val level = request.intParam("level")
            ?: return ToolResponse(request.requestId, false, error = "level required")
        val auto = request.boolParam("auto")

        return try {
            if (!Settings.System.canWrite(context)) {
                return ToolResponse(request.requestId, false, error = "WRITE_SETTINGS permission not granted. Please enable in Settings > Apps > Special access.")
            }

            if (auto == true) {
                Settings.System.putInt(
                    context.contentResolver,
                    Settings.System.SCREEN_BRIGHTNESS_MODE,
                    Settings.System.SCREEN_BRIGHTNESS_MODE_AUTOMATIC
                )
                ToolResponse(request.requestId, true, result = "Auto-brightness enabled")
            } else {
                Settings.System.putInt(
                    context.contentResolver,
                    Settings.System.SCREEN_BRIGHTNESS_MODE,
                    Settings.System.SCREEN_BRIGHTNESS_MODE_MANUAL
                )
                val clampedLevel = level.coerceIn(0, 255)
                Settings.System.putInt(
                    context.contentResolver,
                    Settings.System.SCREEN_BRIGHTNESS,
                    clampedLevel
                )
                ToolResponse(request.requestId, true, result = "Brightness set to: $clampedLevel/255")
            }
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to set brightness: ${e.message}")
        }
    }
}
