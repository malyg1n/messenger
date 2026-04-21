export type User = {
	id: string
	username: string
}

export type ChatMessage = {
	sender_id: string
	receiver_id: string
	body: string
}