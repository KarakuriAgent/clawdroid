package io.clawdroid.assistant.actions

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class ClipboardActionHandler : ActionHandler {
    override val supportedActions = setOf("clipboard_copy", "clipboard_read")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "clipboard_copy" -> handleCopy(request, context)
            "clipboard_read" -> handleRead(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown clipboard action")
        }
    }

    private fun handleCopy(request: ToolRequest, context: Context): ToolResponse {
        val text = request.stringParam("text")
            ?: return ToolResponse(request.requestId, false, error = "text required")

        return try {
            val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            clipboard.setPrimaryClip(ClipData.newPlainText("ClawDroid", text))
            ToolResponse(request.requestId, true, result = "Copied to clipboard: ${text.take(100)}")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to copy: ${e.message}")
        }
    }

    private fun handleRead(request: ToolRequest, context: Context): ToolResponse {
        return try {
            val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            if (!clipboard.hasPrimaryClip()) {
                return ToolResponse(request.requestId, true, result = "Clipboard is empty")
            }
            val clip = clipboard.primaryClip
            val text = clip?.getItemAt(0)?.text?.toString() ?: "(non-text content)"
            ToolResponse(request.requestId, true, result = text)
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to read clipboard: ${e.message}")
        }
    }
}
