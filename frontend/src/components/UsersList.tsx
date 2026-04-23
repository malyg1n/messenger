import { useEffect, useState } from "react"
import api from "../api/client"
import { Chat, User } from "../types"

type Props = {
	currentUser: User
	onSelect: (chat: Chat) => void
	selectedChatId?: string
	refreshToken?: number
	lastMessageOverrides?: Record<string, { last_message: string; last_message_at: string }>
}

export default function UsersList({
	currentUser,
	onSelect,
	selectedChatId,
	refreshToken = 0,
	lastMessageOverrides = {}
}: Props) {
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

	function toTimeMs(value?: string) {
		if (!value) return 0
		const parsed = new Date(value).getTime()
		return Number.isNaN(parsed) ? 0 : parsed
	}

	useEffect(() => {
		let isActive = true

		api.getChats(currentUser.id).then(nextChats => {
			if (!isActive) return
			setChats(nextChats)
		})

		api.getUsers().then(nextUsers => {
			if (!isActive) return
			setUsers(nextUsers)
		})

		return () => {
			isActive = false
		}
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

	const chatsForRender = chats
		.map(chat => {
			const override = lastMessageOverrides[chat.chat_id]
			if (!override) return chat
			const overrideTime = toTimeMs(override.last_message_at)
			const chatTime = toTimeMs(chat.last_message_at)
			if (overrideTime < chatTime) return chat
			return {
				...chat,
				last_message: override.last_message,
				last_message_at: override.last_message_at
			}
		})
		.sort((a, b) => toTimeMs(b.last_message_at) - toTimeMs(a.last_message_at))

	return (
		<div className="chat-sidebar">
			<div className="sidebar-title">Чаты</div>

			{chatsForRender.map(chat => (
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