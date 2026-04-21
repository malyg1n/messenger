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
	id: string
}