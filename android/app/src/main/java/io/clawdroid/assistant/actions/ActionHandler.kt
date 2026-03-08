package io.clawdroid.assistant.actions

import android.content.Context
import android.content.Intent
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonPrimitive

interface ActionHandler {
    val supportedActions: Set<String>
    suspend fun handle(request: ToolRequest, context: Context): ToolResponse
    fun requiredPermissions(action: String): List<PermissionRequirement> = emptyList()
}

fun ToolRequest.stringParam(name: String): String? =
    params?.get(name)?.jsonPrimitive?.contentOrNull

fun ToolRequest.intParam(name: String): Int? =
    params?.get(name)?.jsonPrimitive?.intOrNull

fun ToolRequest.boolParam(name: String): Boolean? =
    params?.get(name)?.jsonPrimitive?.booleanOrNull

fun ToolRequest.doubleParam(name: String): Double? =
    params?.get(name)?.jsonPrimitive?.doubleOrNull

fun launchActivity(request: ToolRequest, context: Context, intent: Intent, successMessage: String): ToolResponse {
    return try {
        context.startActivity(intent)
        ToolResponse(request.requestId, true, result = successMessage)
    } catch (e: Exception) {
        ToolResponse(request.requestId, false, error = e.message ?: "Unknown error")
    }
}
