import { useEffect, useRef, useState } from "react"
import { Chat as ChatItem, ChatMessage, User } from "../types"
import UsersList from "./UsersList"
import api from "../api/client"

type Props = {
	user: User
	onLogout: () => void
}

type ViewMessage = {
	text: string
	isMe: boolean
	timestamp: string
	createdAtMs: number
}

type LastMessageOverride = {
	last_message: string
	last_message_at: string
}

const PAGE_SIZE = 50

export default function Chat({ user, onLogout }: Props) {

	const [chatId, setChatId] = useState<string | null>(null)
	const [selectedChat, setSelectedChat] = useState<ChatItem | null>(null)
	const [text, setText] = useState("")
	const [messages, setMessages] = useState<ViewMessage[]>([])
	const [socket, setSocket] = useState<WebSocket | null>(null)
	const [chatsRefreshToken, setChatsRefreshToken] = useState(0)
	const [messagesLimit, setMessagesLimit] = useState(PAGE_SIZE)
	const [isLoadingOlder, setIsLoadingOlder] = useState(false)
	const [hasMoreOlder, setHasMoreOlder] = useState(true)
	const [lastMessageOverrides, setLastMessageOverrides] = useState<Record<string, LastMessageOverride>>({})
	const messagesRef = useRef<HTMLDivElement | null>(null)
	const selectedChatRef = useRef<ChatItem | null>(null)
	const selectedChatIdRef = useRef<string | null>(null)

	function formatTimestamp(value?: string) {
		const date = value ? new Date(value) : new Date()
		if (Number.isNaN(date.getTime())) {
			return new Date().toLocaleString("ru-RU", {
				day: "2-digit",
				month: "2-digit",
				hour: "2-digit",
				minute: "2-digit"
			})
		}

		return date.toLocaleString("ru-RU", {
			day: "2-digit",
			month: "2-digit",
			hour: "2-digit",
			minute: "2-digit"
		})
	}

	function toCreatedAtMs(value?: string) {
		const parsed = value ? new Date(value).getTime() : Date.now()
		return Number.isNaN(parsed) ? Date.now() : parsed
	}

	function toViewMessages(history: { sender_id: string; body: string; created_at?: string }[], chat: ChatItem) {
		return history
			.map(m => ({
				text: m.sender_id === user.id ? "me: " + m.body : chat.title + ": " + m.body,
				isMe: m.sender_id === user.id,
				timestamp: formatTimestamp(m.created_at),
				createdAtMs: toCreatedAtMs(m.created_at)
			}))
			.sort((a, b) => a.createdAtMs - b.createdAtMs)
	}

	function scrollToBottom() {
		requestAnimationFrame(() => {
			requestAnimationFrame(() => {
				const container = messagesRef.current
				if (!container) return
				container.scrollTop = container.scrollHeight
			})
		})
	}

	async function loadHistory(targetChatId: string, chat: ChatItem, limit: number, preserveScroll = false) {
		const container = messagesRef.current
		const prevScrollTop = container?.scrollTop ?? 0
		const prevScrollHeight = container?.scrollHeight ?? 0
		const history = await api.getMessages(targetChatId, limit)
		setMessages(toViewMessages(history ?? [], chat))
		setHasMoreOlder((history?.length ?? 0) >= limit)

		requestAnimationFrame(() => {
			const current = messagesRef.current
			if (!current) return
			if (preserveScroll) {
				const newScrollHeight = current.scrollHeight
				current.scrollTop = prevScrollTop + (newScrollHeight - prevScrollHeight)
				return
			}
			scrollToBottom()
		})
	}

	useEffect(() => {
		const ws = new WebSocket(`ws://localhost:8080/ws?user_id=${user.id}`)
		ws.onmessage = e => {
			const m = JSON.parse(e.data) as { sender_id: string; chat_id: string; body: string; created_at?: string }
			const createdAt = m.created_at ?? new Date().toISOString()
			if (m.chat_id) {
				setLastMessageOverrides(prev => ({
					...prev,
					[m.chat_id]: {
						last_message: m.body,
						last_message_at: createdAt
					}
				}))
			}
			if (m.sender_id !== user.id && m.chat_id === selectedChatIdRef.current) {
				setMessages(prev => [
					...prev,
					{
						text: "them: " + m.body,
						isMe: false,
						timestamp: formatTimestamp(createdAt),
						createdAtMs: toCreatedAtMs(createdAt)
					}
				])
				scrollToBottom()
			}
			if (m.sender_id !== user.id) {
				setChatsRefreshToken(prev => prev + 1)
			}
		}

		setSocket(ws)
		return () => ws.close()
	}, [])

	function send() {

		if (!socket || !chatId || !text.trim()) return

		const msg: ChatMessage = {

			chat_id: chatId,
			sender_id: user.id,
			body: text

		}

		socket.send(JSON.stringify(msg))
		setChatsRefreshToken(prev => prev + 1)
		setLastMessageOverrides(prev => ({
			...prev,
			[chatId]: {
				last_message: text,
				last_message_at: new Date().toISOString()
			}
		}))

		setMessages(prev => [
			...prev,
			{
				text: "me: " + text,
				isMe: true,
				timestamp: formatTimestamp(),
				createdAtMs: Date.now()
			}
		])

		setText("")
		scrollToBottom()

	}

	async function selectChat(chat: ChatItem) {

		setSelectedChat(chat)
		selectedChatRef.current = chat
		setMessages([])
		setMessagesLimit(PAGE_SIZE)
		setHasMoreOlder(true)

		setChatId(chat.chat_id)
		selectedChatIdRef.current = chat.chat_id
		await loadHistory(chat.chat_id, chat, PAGE_SIZE)
		scrollToBottom()
	
	}

	async function handleMessagesScroll() {
		const container = messagesRef.current
		if (!container || !chatId || !selectedChatRef.current) return
		if (isLoadingOlder || !hasMoreOlder) return
		if (container.scrollTop > 40) return

		setIsLoadingOlder(true)
		const nextLimit = messagesLimit + PAGE_SIZE
		setMessagesLimit(nextLimit)

		try {
			await loadHistory(chatId, selectedChatRef.current, nextLimit, true)
		} finally {
			setIsLoadingOlder(false)
		}
	}

	return (
		<div className="chat-shell">
			<UsersList
				currentUser={user}
				onSelect={selectChat}
				selectedChatId={selectedChat?.chat_id}
				refreshToken={chatsRefreshToken}
				lastMessageOverrides={lastMessageOverrides}
			/>

			<div className="chat-main">
				<div className="chat-topbar">
					<div className="chat-topbar-header">
						<div className="chat-topbar-title">
							{selectedChat?.title ?? "Выберите чат"}
						</div>
						<button className="btn btn-secondary chat-logout-btn" onClick={onLogout}>
							Выйти
						</button>
					</div>
					<div className="chat-topbar-subtitle">Вы: {user.username}</div>
				</div>

				{selectedChat ? (
					<>
						<div className="messages" ref={messagesRef} onScroll={handleMessagesScroll}>
							{messages.map((m, i) => (
								<div
									key={i}
									className={`message-row ${m.isMe ? "message-row-me" : "message-row-them"}`}
								>
									<div className={`message-bubble ${m.isMe ? "message-bubble-me" : "message-bubble-them"}`}>
										<div>{m.text}</div>
										<div className="message-meta">{m.timestamp}</div>
									</div>
								</div>
							))}
							{isLoadingOlder ? <div className="messages-status">Загружаем более старые сообщения...</div> : null}
						</div>

						<div className="chat-input-row">
							<input
								className="chat-input"
								value={text}
								onChange={e => setText(e.target.value)}
								onKeyDown={e => {
									if (e.key === "Enter") {
										send()
									}
								}}
								placeholder="Введите сообщение..."
							/>

							<button className="btn btn-primary" onClick={send}>
								Отправить
							</button>
						</div>
					</>
				) : (
					<div className="chat-empty-state">
						Выберите чат в списке слева, чтобы открыть переписку
					</div>
				)}
			</div>
		</div>
	)

}