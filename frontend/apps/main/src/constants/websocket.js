export const WS_EVENT = {
    NEW_MESSAGE: 'new_message',
    MESSAGE_UPDATE: 'message_update',
    CONVERSATION_UPDATE: 'conversation_update',
    CONTACT_UPDATE: 'contact_update',
    CONVERSATION_SUBSCRIBE: 'conversation_subscribe',
    CONVERSATION_SUBSCRIBED: 'conversation_subscribed',
    TYPING: 'typing',
}

// Message types that should not be queued because they become stale quickly
export const WS_EPHEMERAL_TYPES = [
    WS_EVENT.TYPING,
]