package io.clawdroid.assistant.actions

import android.Manifest
import android.content.Context
import android.content.Intent
import android.provider.ContactsContract
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class ContactsActionHandler : ActionHandler {
    override val supportedActions = setOf("search_contacts", "get_contact_detail", "add_contact")

    override fun requiredPermissions(action: String): List<PermissionRequirement> = when (action) {
        "search_contacts", "get_contact_detail" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.READ_CONTACTS, "Contacts read access"))
        else -> emptyList()
    }

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "search_contacts" -> handleSearchContacts(request, context)
            "get_contact_detail" -> handleGetContactDetail(request, context)
            "add_contact" -> handleAddContact(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown contacts action")
        }
    }

    private fun handleSearchContacts(request: ToolRequest, context: Context): ToolResponse {
        val query = request.stringParam("query")
            ?: return ToolResponse(request.requestId, false, error = "query required")

        val uri = ContactsContract.Contacts.CONTENT_URI
        val projection = arrayOf(
            ContactsContract.Contacts._ID,
            ContactsContract.Contacts.DISPLAY_NAME_PRIMARY,
            ContactsContract.Contacts.HAS_PHONE_NUMBER
        )
        val escapedQuery = query.replace("%", "\\%").replace("_", "\\_")
        val selection = "${ContactsContract.Contacts.DISPLAY_NAME_PRIMARY} LIKE ? ESCAPE '\\'"
        val selectionArgs = arrayOf("%$escapedQuery%")

        return try {
            val cursor = context.contentResolver.query(uri, projection, selection, selectionArgs, null)
            val results = buildString {
                appendLine("Contacts found: ${cursor?.count ?: 0}")
                cursor?.use {
                    var count = 0
                    while (it.moveToNext() && count < 20) {
                        val id = it.getLong(0)
                        val name = it.getString(1) ?: "(no name)"
                        val hasPhone = it.getInt(2) > 0
                        appendLine("---")
                        appendLine("ID: $id")
                        appendLine("Name: $name")
                        if (hasPhone) {
                            appendLine("Phone: ${getPhoneNumbers(context, id)}")
                        }
                        count++
                    }
                }
            }
            ToolResponse(request.requestId, true, result = results)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Contacts permission not granted. Please grant READ_CONTACTS permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to search contacts: ${e.message}")
        }
    }

    private fun getPhoneNumbers(context: Context, contactId: Long): String {
        val phones = mutableListOf<String>()
        val cursor = context.contentResolver.query(
            ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
            arrayOf(ContactsContract.CommonDataKinds.Phone.NUMBER),
            "${ContactsContract.CommonDataKinds.Phone.CONTACT_ID} = ?",
            arrayOf(contactId.toString()),
            null
        )
        cursor?.use {
            while (it.moveToNext()) {
                phones.add(it.getString(0) ?: "")
            }
        }
        return phones.joinToString(", ")
    }

    private fun handleGetContactDetail(request: ToolRequest, context: Context): ToolResponse {
        val contactId = request.stringParam("contact_id")
            ?: return ToolResponse(request.requestId, false, error = "contact_id required")

        return try {
            val result = buildString {
                val cursor = context.contentResolver.query(
                    ContactsContract.Contacts.CONTENT_URI,
                    arrayOf(ContactsContract.Contacts.DISPLAY_NAME_PRIMARY),
                    "${ContactsContract.Contacts._ID} = ?",
                    arrayOf(contactId),
                    null
                )
                cursor?.use {
                    if (it.moveToFirst()) {
                        appendLine("Name: ${it.getString(0)}")
                    }
                }

                appendLine("Phone numbers:")
                val phoneCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
                    arrayOf(
                        ContactsContract.CommonDataKinds.Phone.NUMBER,
                        ContactsContract.CommonDataKinds.Phone.TYPE
                    ),
                    "${ContactsContract.CommonDataKinds.Phone.CONTACT_ID} = ?",
                    arrayOf(contactId),
                    null
                )
                phoneCursor?.use {
                    while (it.moveToNext()) {
                        val number = it.getString(0)
                        val type = ContactsContract.CommonDataKinds.Phone.getTypeLabel(
                            context.resources, it.getInt(1), ""
                        )
                        appendLine("  $type: $number")
                    }
                }

                appendLine("Email addresses:")
                val emailCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Email.CONTENT_URI,
                    arrayOf(ContactsContract.CommonDataKinds.Email.ADDRESS),
                    "${ContactsContract.CommonDataKinds.Email.CONTACT_ID} = ?",
                    arrayOf(contactId),
                    null
                )
                emailCursor?.use {
                    while (it.moveToNext()) {
                        appendLine("  ${it.getString(0)}")
                    }
                }
            }
            ToolResponse(request.requestId, true, result = result)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Contacts permission not granted. Please grant READ_CONTACTS permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to get contact: ${e.message}")
        }
    }

    private fun handleAddContact(request: ToolRequest, context: Context): ToolResponse {
        val name = request.stringParam("name")
            ?: return ToolResponse(request.requestId, false, error = "name required")

        val intent = Intent(ContactsContract.Intents.Insert.ACTION).apply {
            type = ContactsContract.RawContacts.CONTENT_TYPE
            putExtra(ContactsContract.Intents.Insert.NAME, name)
            request.stringParam("phone")?.let {
                putExtra(ContactsContract.Intents.Insert.PHONE, it)
            }
            request.stringParam("email")?.let {
                putExtra(ContactsContract.Intents.Insert.EMAIL, it)
            }
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Add contact screen opened for: $name")
    }
}
