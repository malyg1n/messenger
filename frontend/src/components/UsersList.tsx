import { useEffect, useState } from "react"
import api from "../api/client"
import { Chat, User } from "../types"

type Props = {
	currentUser: User
	onSelect: (chat: Chat) => void
	selectedChatId?: string
	refreshToken?: number
}

export default function UsersList({ currentUser, onSelect, selectedChatId, refreshToken = 0 }: Props) {
	const [chats, setChats] = useState<Chat[]>([])
	const [users, setUsers] = useState<User[]>([])

	function formatChatTime(value: string) {
		if (!value) return ""
		const date = new Date(value)
		if (Number.isNaN(date.getTime())) return ""
		return date.toLocaleString("ru-RU", {
			day: "2-digit",
			month: "2-digit",
			hour: "2-digit",
			minute: "2-digit"
		})
	}

	useEffect(() => {
		api.getChats(currentUser.id).then(setChats)
		api.getUsers().then(setUsers)
	}, [currentUser.id, refreshToken])

	async function startChatWithUser(target: User) {
		const res = await api.createDirectChat(currentUser.id, target.id)
		const newChat: Chat = {
			chat_id: res.chat_id,
			title: target.username,
			last_message: "",
			last_message_at: ""
		}

		setChats(prev => {
			const exists = prev.some(chat => chat.chat_id === newChat.chat_id)
			if (exists) {
				return prev
			}
			return [newChat, ...prev]
		})

		onSelect(newChat)
	}

	const usersWithoutChats = users.filter(u => {
		if (u.id === currentUser.id) return false
		return !chats.some(chat => chat.title === u.username)
	})

	return (
		<div className="chat-sidebar">
			<div className="sidebar-title">Чаты</div>

			{chats.map(chat => (
					<div
						key={chat.chat_id}
						onClick={() => onSelect(chat)}
						className={`chat-item ${selectedChatId === chat.chat_id ? "chat-item-active" : ""}`}
					>
						<div className="chat-item-head">
							<div className="chat-item-name">{chat.title}</div>
							<div className="chat-item-time">{formatChatTime(chat.last_message_at)}</div>
						</div>
						<div className="chat-item-preview">{chat.last_message || "Нет сообщений"}</div>
					</div>
				))}

			<div className="sidebar-subtitle">Новый чат</div>
			{usersWithoutChats.map(target => (
				<div
					key={target.id}
					onClick={() => startChatWithUser(target)}
					className="chat-item chat-item-new"
				>
					<div className="chat-item-name">{target.username}</div>
					<div className="chat-item-preview">создать диалог</div>
				</div>
			))}
		</div>
	)
}