package io.clawdroid.backend.config

import io.clawdroid.backend.api.GatewaySettingsStore
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.put
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import java.io.Closeable
import java.io.IOException

@Serializable
data class ConfigSchema(val sections: List<SchemaSection>)

@Serializable
data class SchemaSection(val key: String, val label: String, val fields: List<SchemaField>)

@Serializable
data class SchemaField(
    val key: String,
    val label: String,
    val group: String = "",
    val depth: Int = 0,
    val type: String,
    val secret: Boolean = false,
    val default: JsonElement = Json.parseToJsonElement("null"),
)

@Serializable
data class SaveConfigResult(
    val status: String? = null,
    val restart: Boolean = false,
    val error: String? = null,
)

class AuthException(message: String) : IOException(message)

class ConfigApiClient(private val settingsStore: GatewaySettingsStore) : Closeable {
    private val baseUrl: String
        get() = settingsStore.settings.value.httpBaseUrl

    private val apiKey: String
        get() = settingsStore.settings.value.apiKey

    private val client = HttpClient(OkHttp) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true; isLenient = true })
        }
    }

    suspend fun getSchema(): ConfigSchema {
        return client.get("$baseUrl/api/config/schema") {
            if (apiKey.isNotEmpty()) header("Authorization", "Bearer $apiKey")
        }.ensureSuccess().body()
    }

    suspend fun getConfig(): JsonObject {
        return client.get("$baseUrl/api/config") {
            if (apiKey.isNotEmpty()) header("Authorization", "Bearer $apiKey")
        }.ensureSuccess().body()
    }

    suspend fun saveConfig(config: JsonObject): SaveConfigResult {
        return client.put("$baseUrl/api/config") {
            contentType(ContentType.Application.Json)
            setBody(config)
            if (apiKey.isNotEmpty()) header("Authorization", "Bearer $apiKey")
        }.ensureSuccess().body()
    }

    override fun close() {
        client.close()
    }

    private suspend fun HttpResponse.ensureSuccess(): HttpResponse {
        if (!status.isSuccess()) {
            val error = runCatching { body<SaveConfigResult>().error }.getOrNull()
            val message = "HTTP ${status.value}: ${error ?: "request failed"}"
            if (status.value == 403) throw AuthException(message)
            throw IOException(message)
        }
        return this
    }
}
