package io.clawdroid.assistant.actions

import android.app.SearchManager
import android.content.Context
import android.content.Intent
import android.media.AudioManager
import android.provider.MediaStore
import android.view.KeyEvent
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class MediaActionHandler : ActionHandler {
    override val supportedActions = setOf("media_play_pause", "media_next", "media_previous", "play_music_search")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "media_play_pause" -> dispatchMediaKey(context, request, KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE, "Play/Pause toggled")
            "media_next" -> dispatchMediaKey(context, request, KeyEvent.KEYCODE_MEDIA_NEXT, "Skipped to next track")
            "media_previous" -> dispatchMediaKey(context, request, KeyEvent.KEYCODE_MEDIA_PREVIOUS, "Skipped to previous track")
            "play_music_search" -> handlePlayMusicSearch(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown media action")
        }
    }

    private fun dispatchMediaKey(context: Context, request: ToolRequest, keyCode: Int, successMsg: String): ToolResponse {
        return try {
            val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
            val downEvent = KeyEvent(KeyEvent.ACTION_DOWN, keyCode)
            val upEvent = KeyEvent(KeyEvent.ACTION_UP, keyCode)
            audioManager.dispatchMediaKeyEvent(downEvent)
            audioManager.dispatchMediaKeyEvent(upEvent)
            ToolResponse(request.requestId, true, result = successMsg)
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to send media key: ${e.message}")
        }
    }

    private fun handlePlayMusicSearch(request: ToolRequest, context: Context): ToolResponse {
        val query = request.stringParam("query")
            ?: return ToolResponse(request.requestId, false, error = "query required")

        val intent = Intent(MediaStore.INTENT_ACTION_MEDIA_PLAY_FROM_SEARCH).apply {
            putExtra(SearchManager.QUERY, query)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Playing music: $query")
    }
}
