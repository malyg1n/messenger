export type User = {
	id: string
	username: string
}

export type ChatMessage = {
	sender_id: string
	chat_id: string
	body: string
}

export type Chat = {
	chat_id: string
	title: string
	last_message: string
	last_message_at: string
}