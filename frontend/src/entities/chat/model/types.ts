export type Chat = {
  chat_id: string
  title: string
  other_user_id: string
  last_message: string
  last_message_at: string
}

export type IsOnlineResponse = {
  is_online: boolean
  user_id: string
}