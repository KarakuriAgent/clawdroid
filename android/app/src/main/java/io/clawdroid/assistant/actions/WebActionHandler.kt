package io.clawdroid.assistant.actions

import android.app.SearchManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class WebActionHandler : ActionHandler {
    override val supportedActions = setOf("open_url", "web_search")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "open_url" -> handleOpenUrl(request, context)
            "web_search" -> handleWebSearch(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown web action")
        }
    }

    private fun handleOpenUrl(request: ToolRequest, context: Context): ToolResponse {
        val url = request.stringParam("url")
            ?: return ToolResponse(request.requestId, false, error = "url required")

        val intent = Intent(Intent.ACTION_VIEW).apply {
            data = Uri.parse(url)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "URL opened: $url")
    }

    private fun handleWebSearch(request: ToolRequest, context: Context): ToolResponse {
        val query = request.stringParam("query")
            ?: return ToolResponse(request.requestId, false, error = "query required")

        val intent = Intent(Intent.ACTION_WEB_SEARCH).apply {
            putExtra(SearchManager.QUERY, query)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Web search: $query")
    }
}
