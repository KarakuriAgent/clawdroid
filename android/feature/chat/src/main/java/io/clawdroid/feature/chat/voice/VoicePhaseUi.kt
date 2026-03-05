package io.clawdroid.feature.chat.voice

import androidx.compose.ui.graphics.Color
import io.clawdroid.core.domain.model.VoicePhase

val VoicePhase.isInterruptable: Boolean
    get() = this == VoicePhase.SENDING ||
        this == VoicePhase.THINKING ||
        this == VoicePhase.SPEAKING

private val ListeningColor = Color(0xFF00D4FF)
private val SendingColor = Color(0xFFFF8C42)
private val ThinkingColor = Color(0xFFA855F7)
private val SpeakingColor = Color(0xFF22C55E)
private val ErrorColor = Color(0xFFEF4444)
private val PausedColor = Color(0xFFFBBF24)
private val IdleColor = Color(0xFF4A5568)

fun phaseColor(phase: VoicePhase): Color = when (phase) {
    VoicePhase.LISTENING -> ListeningColor
    VoicePhase.PAUSED -> PausedColor
    VoicePhase.SENDING -> SendingColor
    VoicePhase.THINKING -> ThinkingColor
    VoicePhase.SPEAKING -> SpeakingColor
    VoicePhase.ERROR -> ErrorColor
    VoicePhase.IDLE -> IdleColor
}
