export type ChatMessage = {
  chat_id: string
  client_message_id: string
  body: string
}

export type MessageStatus = "pending" | "saved" | "failed"

export type ViewMessage = {
  body: string
  text: string
  isMe: boolean
  timestamp: string
  createdAtMs: number,
  createdAt: string
  status: MessageStatus
  clientMessageId?: string
}

export type LastMessageOverride = {
  last_message: string
  last_message_at: string
}