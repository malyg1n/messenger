import { useEffect, useRef, useState } from "react"
import type { Chat } from "@/entities/chat"
import type { ChatMessage } from "@/entities/message"
import { UsersList } from "@/entities/user"
import type { User } from "@/entities/user"
import api from "@/shared/api/client"

type Props = {
  user: User
  onLogout: () => void
}

type ViewMessage = {
  text: string
  isMe: boolean
  timestamp: string
  createdAtMs: number,
  createdAt: string
}

type LastMessageOverride = {
  last_message: string
  last_message_at: string
}

const PAGE_SIZE = 50

// ChatWidget — основной UI чата: список диалогов, история и отправка сообщений.
export default function ChatWidget({ user, onLogout }: Props) {
  // Локальное состояние виджета: выбранный чат, сообщения, сокет и служебные флаги UI.
  const [chatId, setChatId] = useState<string | null>(null)
  const [selectedChat, setSelectedChat] = useState<Chat | null>(null)
  const [text, setText] = useState("")
  const [messages, setMessages] = useState<ViewMessage[]>([])
  const [socket, setSocket] = useState<WebSocket | null>(null)
  const [chatsRefreshToken, setChatsRefreshToken] = useState(0)
  const [messagesLimit, setMessagesLimit] = useState(PAGE_SIZE)
  const [isLoadingOlder, setIsLoadingOlder] = useState(false)
  const [hasMoreOlder, setHasMoreOlder] = useState(true)
  const [lastMessageOverrides, setLastMessageOverrides] = useState<Record<string, LastMessageOverride>>({})
  const messagesRef = useRef<HTMLDivElement | null>(null)
  const selectedChatRef = useRef<Chat | null>(null)
  const selectedChatIdRef = useRef<string | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected">("connecting")
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectAttemptRef = useRef(0)

  const connectionStatusLabel: Record<"connecting" | "connected" | "disconnected", string> = {
    connecting: "Подключение",
    connected: "В сети",
    disconnected: "Нет соединения"
  }

  // formatTimestamp приводит серверный timestamp к читабельному виду.
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

  // toCreatedAtMs переводит дату в миллисекунды для сортировки.
  function toCreatedAtMs(value?: string) {
    const parsed = value ? new Date(value).getTime() : Date.now()
    return Number.isNaN(parsed) ? Date.now() : parsed
  }

  // toViewMessages конвертирует API-модель в формат отображения.
  function toViewMessages(history: { sender_id: string; body: string; created_at?: string }[], chat: Chat) {
    return history
      .map(m => ({
        text: m.sender_id === user.id ? "me: " + m.body : chat.title + ": " + m.body,
        isMe: m.sender_id === user.id,
        timestamp: formatTimestamp(m.created_at),
        createdAtMs: toCreatedAtMs(m.created_at),
        createdAt: m.created_at ?? new Date().toISOString()
      }))
      .sort((a, b) => a.createdAtMs - b.createdAtMs)
  }

  // scrollToBottom прокручивает контейнер сообщений к последнему элементу.
  function scrollToBottom() {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const container = messagesRef.current
        if (!container) return
        container.scrollTop = container.scrollHeight
      })
    })
  }

  // loadHistory загружает историю выбранного чата и обновляет состояние UI.
  async function loadHistory(targetChatId: string, chat: Chat, limit: number, preserveScroll = false) {
    const container = messagesRef.current
    const prevScrollTop = container?.scrollTop ?? 0
    const prevScrollHeight = container?.scrollHeight ?? 0
    const before = messages?.[messages.length - 1]?.createdAt ?? ''
    // История загружается с API, а затем приводится к формату для рендера в списке сообщений.
    const history = await api.getMessages(targetChatId, limit, before)
    setMessages(toViewMessages(history ?? [], chat))
    setHasMoreOlder((history?.length ?? 0) >= limit)

    requestAnimationFrame(() => {
      const current = messagesRef.current
      if (!current) return
      if (preserveScroll) {
        // При догрузке старых сообщений сохраняем позицию, чтобы экран не «прыгал».
        const newScrollHeight = current.scrollHeight
        current.scrollTop = prevScrollTop + (newScrollHeight - prevScrollHeight)
        return
      }
      scrollToBottom()
    })
  }

  // createSocket создает websocket-подключение для текущего пользователя.
  function createSocket() {
    const wsUrl = import.meta.env.VITE_WS_URL ?? "ws://localhost:8080/ws"
    return new WebSocket(`${wsUrl}?user_id=${user.id}`)
  }

  useEffect(() => {
    // Держим одно websocket-подключение на пользователя и переподключаемся при обрыве.
    let isUnmounted = false
    let ws: WebSocket | null = null

    const clearReconnectTimeout = () => {
      if (reconnectTimeoutRef.current !== null) {
        window.clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
    }

    const connect = () => {
      if (isUnmounted) return
      setConnectionStatus("connecting")
      ws = createSocket()
      setSocket(ws)

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
          // Если сообщение относится к открытому чату — сразу показываем его в ленте.
          setMessages(prev => [
            ...prev,
            {
              text: "them: " + m.body,
              isMe: false,
              timestamp: formatTimestamp(createdAt),
              createdAtMs: toCreatedAtMs(createdAt),
              createdAt: createdAt
            }
          ])
          scrollToBottom()
        }
        if (m.sender_id !== user.id) {
          setChatsRefreshToken(prev => prev + 1)
        }
      }

      ws.onopen = () => {
        reconnectAttemptRef.current = 0
        clearReconnectTimeout()
        setConnectionStatus("connected")
      }
      ws.onclose = () => {
        setConnectionStatus("disconnected")
        setSocket(prev => (prev === ws ? null : prev))
        clearReconnectTimeout()

        // Exponential backoff with cap protects from reconnect storms.
        const attempt = reconnectAttemptRef.current
        const delayMs = Math.min(30000, 1000 * Math.pow(2, attempt))
        reconnectAttemptRef.current = Math.min(attempt + 1, 10)
        reconnectTimeoutRef.current = window.setTimeout(connect, delayMs)
      }
      ws.onerror = () => {
        setConnectionStatus("disconnected")
      }
    }

    connect()

    return () => {
      isUnmounted = true
      reconnectAttemptRef.current = 0
      clearReconnectTimeout()
      if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
        ws.close()
      }
      setSocket(null)
    }
  }, [user.id])

  // send отправляет сообщение в websocket и сразу отражает его в интерфейсе.
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

    // Оптимистично отображаем сообщение: не ждём round-trip через сервер и Kafka.
    setMessages(prev => [
      ...prev,
      {
        text: "me: " + text,
        isMe: true,
        timestamp: formatTimestamp(),
        createdAtMs: Date.now(),
        createdAt: new Date().toISOString()
      }
    ])

    setText("")
    scrollToBottom()
  }

  // selectChat переключает активный чат и загружает его историю.
  async function selectChat(chat: Chat) {
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

  // handleMessagesScroll подгружает старые сообщения при прокрутке вверх.
  async function handleMessagesScroll() {
    const container = messagesRef.current
    if (!container || !chatId || !selectedChatRef.current) return
    if (isLoadingOlder || !hasMoreOlder) return
    if (container.scrollTop > 40) return

    // Бесконечная прокрутка вверх: увеличиваем лимит и перезапрашиваем историю.
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
            <div className="chat-topbar-title">{selectedChat?.title ?? "Выберите чат"}</div>
            <button className="btn btn-secondary chat-logout-btn" onClick={onLogout}>
              Выйти
            </button>
          </div>
          <div className="chat-topbar-subtitle">
            Вы: {user.username}
            <span className="chat-connection-status">
              <span
                className={`chat-connection-dot chat-connection-dot-${connectionStatus}`}
                aria-hidden="true"
              />
              {connectionStatusLabel[connectionStatus]}
            </span>
          </div>
        </div>

        {selectedChat ? (
          <>
            <div className="messages" ref={messagesRef} onScroll={handleMessagesScroll}>
              {messages.map((m, i) => (
                <div key={i} className={`message-row ${m.isMe ? "message-row-me" : "message-row-them"}`}>
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
          <div className="chat-empty-state">Выберите чат в списке слева, чтобы открыть переписку</div>
        )}
      </div>
    </div>
  )
}
