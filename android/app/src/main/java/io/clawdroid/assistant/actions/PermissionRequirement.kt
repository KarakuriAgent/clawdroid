package io.clawdroid.assistant.actions

import android.content.Context
import android.content.Intent

sealed class PermissionRequirement {
    abstract val description: String

    data class Runtime(
        val permission: String,
        override val description: String
    ) : PermissionRequirement()

    data class Special(
        val check: (Context) -> Boolean,
        val settingsIntent: Intent,
        override val description: String
    ) : PermissionRequirement()
}
