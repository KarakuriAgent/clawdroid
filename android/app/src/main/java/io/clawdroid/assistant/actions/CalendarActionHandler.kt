package io.clawdroid.assistant.actions

import android.Manifest
import android.content.ContentUris
import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.provider.CalendarContract
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone

class CalendarActionHandler : ActionHandler {
    override val supportedActions = setOf(
        "create_event", "query_events", "update_event",
        "delete_event", "list_calendars", "add_reminder"
    )

    override fun requiredPermissions(action: String): List<PermissionRequirement> = when (action) {
        "query_events", "list_calendars" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.READ_CALENDAR, "Calendar read access"))
        "update_event", "delete_event", "add_reminder" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.WRITE_CALENDAR, "Calendar write access"))
        else -> emptyList()
    }

    private fun isoFormat(): SimpleDateFormat =
        SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.US).apply {
            timeZone = TimeZone.getDefault()
        }

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "create_event" -> handleCreateEvent(request, context)
            "query_events" -> handleQueryEvents(request, context)
            "update_event" -> handleUpdateEvent(request, context)
            "delete_event" -> handleDeleteEvent(request, context)
            "list_calendars" -> handleListCalendars(request, context)
            "add_reminder" -> handleAddReminder(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown calendar action")
        }
    }

    private fun handleCreateEvent(request: ToolRequest, context: Context): ToolResponse {
        val title = request.stringParam("title")
            ?: return ToolResponse(request.requestId, false, error = "title required")
        val startTimeStr = request.stringParam("start_time")
            ?: return ToolResponse(request.requestId, false, error = "start_time required")

        val fmt = isoFormat()
        val startMillis = try { fmt.parse(startTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid start_time format: ${e.message}")
        }

        val intent = Intent(Intent.ACTION_INSERT).apply {
            data = CalendarContract.Events.CONTENT_URI
            putExtra(CalendarContract.EXTRA_EVENT_BEGIN_TIME, startMillis)
            putExtra(CalendarContract.Events.TITLE, title)
            request.stringParam("end_time")?.let {
                try { fmt.parse(it)?.time } catch (_: Exception) { null }
            }?.let { putExtra(CalendarContract.EXTRA_EVENT_END_TIME, it) }
            request.stringParam("description")?.let {
                putExtra(CalendarContract.Events.DESCRIPTION, it)
            }
            request.stringParam("location")?.let {
                putExtra(CalendarContract.Events.EVENT_LOCATION, it)
            }
            request.boolParam("all_day")?.let {
                putExtra(CalendarContract.EXTRA_EVENT_ALL_DAY, it)
            }
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }

        return launchActivity(request, context, intent, "Calendar event creation opened: $title")
    }

    private fun handleQueryEvents(request: ToolRequest, context: Context): ToolResponse {
        val startTimeStr = request.stringParam("start_time")
            ?: return ToolResponse(request.requestId, false, error = "start_time required")
        val endTimeStr = request.stringParam("end_time")
            ?: return ToolResponse(request.requestId, false, error = "end_time required")

        val fmt = isoFormat()
        val startMillis = try { fmt.parse(startTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid start_time: ${e.message}")
        }
        val endMillis = try { fmt.parse(endTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid end_time: ${e.message}")
        }

        val projection = arrayOf(
            CalendarContract.Instances.EVENT_ID,
            CalendarContract.Instances.TITLE,
            CalendarContract.Instances.BEGIN,
            CalendarContract.Instances.END,
            CalendarContract.Instances.EVENT_LOCATION,
            CalendarContract.Instances.DESCRIPTION,
            CalendarContract.Instances.ALL_DAY
        )

        val uri = CalendarContract.Instances.CONTENT_URI.buildUpon()
            .appendPath(startMillis.toString())
            .appendPath(endMillis.toString())
            .build()

        val query = request.stringParam("query")
        val selection = if (query != null) "${CalendarContract.Instances.TITLE} LIKE ?" else null
        val selectionArgs = if (query != null) arrayOf("%$query%") else null

        return try {
            val cursor = context.contentResolver.query(uri, projection, selection, selectionArgs, "${CalendarContract.Instances.BEGIN} ASC")
            val events = buildString {
                appendLine("Events found: ${cursor?.count ?: 0}")
                cursor?.use {
                    while (it.moveToNext()) {
                        val eventId = it.getLong(0)
                        val title = it.getString(1) ?: "(no title)"
                        val begin = it.getLong(2)
                        val end = it.getLong(3)
                        val location = it.getString(4) ?: ""
                        val description = it.getString(5) ?: ""
                        val allDay = it.getInt(6) == 1

                        appendLine("---")
                        appendLine("ID: $eventId")
                        appendLine("Title: $title")
                        appendLine("Start: ${fmt.format(begin)}")
                        appendLine("End: ${fmt.format(end)}")
                        if (allDay) appendLine("All day: true")
                        if (location.isNotEmpty()) appendLine("Location: $location")
                        if (description.isNotEmpty()) appendLine("Description: $description")
                    }
                }
            }
            ToolResponse(request.requestId, true, result = events)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant READ_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to query events: ${e.message}")
        }
    }

    private fun handleUpdateEvent(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")

        val fmt = isoFormat()
        val values = ContentValues()
        request.stringParam("title")?.let {
            values.put(CalendarContract.Events.TITLE, it)
        }
        request.stringParam("description")?.let {
            values.put(CalendarContract.Events.DESCRIPTION, it)
        }
        request.stringParam("location")?.let {
            values.put(CalendarContract.Events.EVENT_LOCATION, it)
        }
        request.stringParam("start_time")?.let {
            try { fmt.parse(it)?.time } catch (_: Exception) { null }
        }?.let { values.put(CalendarContract.Events.DTSTART, it) }
        request.stringParam("end_time")?.let {
            try { fmt.parse(it)?.time } catch (_: Exception) { null }
        }?.let { values.put(CalendarContract.Events.DTEND, it) }

        if (values.size() == 0) {
            return ToolResponse(request.requestId, false, error = "No fields to update")
        }

        return try {
            val uri = ContentUris.withAppendedId(CalendarContract.Events.CONTENT_URI, eventId.toLong())
            val rows = context.contentResolver.update(uri, values, null, null)
            ToolResponse(request.requestId, true, result = "Updated $rows event(s)")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to update event: ${e.message}")
        }
    }

    private fun handleDeleteEvent(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")

        return try {
            val uri = ContentUris.withAppendedId(CalendarContract.Events.CONTENT_URI, eventId.toLong())
            val rows = context.contentResolver.delete(uri, null, null)
            ToolResponse(request.requestId, true, result = "Deleted $rows event(s)")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to delete event: ${e.message}")
        }
    }

    private fun handleListCalendars(request: ToolRequest, context: Context): ToolResponse {
        val projection = arrayOf(
            CalendarContract.Calendars._ID,
            CalendarContract.Calendars.CALENDAR_DISPLAY_NAME,
            CalendarContract.Calendars.ACCOUNT_NAME,
            CalendarContract.Calendars.ACCOUNT_TYPE,
            CalendarContract.Calendars.CALENDAR_COLOR
        )

        return try {
            val cursor = context.contentResolver.query(
                CalendarContract.Calendars.CONTENT_URI, projection, null, null, null
            )
            val calendars = buildString {
                appendLine("Calendars found: ${cursor?.count ?: 0}")
                cursor?.use {
                    while (it.moveToNext()) {
                        appendLine("---")
                        appendLine("ID: ${it.getLong(0)}")
                        appendLine("Name: ${it.getString(1)}")
                        appendLine("Account: ${it.getString(2)} (${it.getString(3)})")
                    }
                }
            }
            ToolResponse(request.requestId, true, result = calendars)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant READ_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to list calendars: ${e.message}")
        }
    }

    private fun handleAddReminder(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")
        val minutes = request.intParam("minutes")
            ?: return ToolResponse(request.requestId, false, error = "minutes required")

        val values = ContentValues().apply {
            put(CalendarContract.Reminders.EVENT_ID, eventId.toLong())
            put(CalendarContract.Reminders.MINUTES, minutes)
            put(CalendarContract.Reminders.METHOD, CalendarContract.Reminders.METHOD_ALERT)
        }

        return try {
            context.contentResolver.insert(CalendarContract.Reminders.CONTENT_URI, values)
            ToolResponse(request.requestId, true, result = "Reminder added: $minutes minutes before event")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to add reminder: ${e.message}")
        }
    }
}
