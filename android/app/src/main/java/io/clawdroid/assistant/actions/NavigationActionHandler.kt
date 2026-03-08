package io.clawdroid.assistant.actions

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.location.Geocoder
import android.location.LocationManager
import android.net.Uri
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse
import java.util.Locale

class NavigationActionHandler : ActionHandler {
    override val supportedActions = setOf("navigate", "search_nearby", "show_map", "get_current_location")

    override fun requiredPermissions(action: String): List<PermissionRequirement> = when (action) {
        "get_current_location" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.ACCESS_FINE_LOCATION, "Location access"))
        else -> emptyList()
    }

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "navigate" -> handleNavigate(request, context)
            "search_nearby" -> handleSearchNearby(request, context)
            "show_map" -> handleShowMap(request, context)
            "get_current_location" -> handleGetCurrentLocation(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown navigation action")
        }
    }

    private fun handleNavigate(request: ToolRequest, context: Context): ToolResponse {
        val destination = request.stringParam("destination")
            ?: return ToolResponse(request.requestId, false, error = "destination required")
        val mode = request.stringParam("mode")

        val modeParam = when (mode) {
            "driving" -> "&mode=d"
            "walking" -> "&mode=w"
            "bicycling" -> "&mode=b"
            "transit" -> "&mode=r"
            else -> ""
        }

        val uri = Uri.parse("google.navigation:q=${Uri.encode(destination)}$modeParam")
        val intent = Intent(Intent.ACTION_VIEW, uri).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Navigation started to: $destination")
    }

    private fun handleSearchNearby(request: ToolRequest, context: Context): ToolResponse {
        val query = request.stringParam("query")
            ?: return ToolResponse(request.requestId, false, error = "query required")

        val uri = Uri.parse("geo:0,0?q=${Uri.encode(query)}")
        val intent = Intent(Intent.ACTION_VIEW, uri).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Searching nearby: $query")
    }

    private fun handleShowMap(request: ToolRequest, context: Context): ToolResponse {
        val query = request.stringParam("query")
        val lat = request.doubleParam("latitude")
        val lng = request.doubleParam("longitude")

        val uriString = when {
            lat != null && lng != null && query != null -> "geo:$lat,$lng?q=${Uri.encode(query)}"
            lat != null && lng != null -> "geo:$lat,$lng"
            query != null -> "geo:0,0?q=${Uri.encode(query)}"
            else -> return ToolResponse(request.requestId, false, error = "query or latitude+longitude required")
        }

        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(uriString)).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Map opened")
    }

    @SuppressLint("MissingPermission")
    private fun handleGetCurrentLocation(request: ToolRequest, context: Context): ToolResponse {
        return try {
            val locationManager = context.getSystemService(Context.LOCATION_SERVICE) as LocationManager
            val location = locationManager.getLastKnownLocation(LocationManager.FUSED_PROVIDER)
                ?: locationManager.getLastKnownLocation(LocationManager.GPS_PROVIDER)
                ?: locationManager.getLastKnownLocation(LocationManager.NETWORK_PROVIDER)
                ?: return ToolResponse(request.requestId, false, error = "Could not determine location. Please ensure location is enabled.")

            val result = buildString {
                appendLine("Latitude: ${location.latitude}")
                appendLine("Longitude: ${location.longitude}")
                appendLine("Accuracy: ${location.accuracy}m")
                try {
                    @Suppress("DEPRECATION")
                    val addresses = Geocoder(context, Locale.getDefault()).getFromLocation(
                        location.latitude, location.longitude, 1
                    )
                    if (!addresses.isNullOrEmpty()) {
                        val addr = addresses[0]
                        val addressLine = (0..addr.maxAddressLineIndex).joinToString(", ") {
                            addr.getAddressLine(it)
                        }
                        appendLine("Address: $addressLine")
                    }
                } catch (_: Exception) {
                    // Geocoder may not be available
                }
            }
            ToolResponse(request.requestId, true, result = result)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Location permission not granted. Please grant ACCESS_FINE_LOCATION permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to get location: ${e.message}")
        }
    }
}
