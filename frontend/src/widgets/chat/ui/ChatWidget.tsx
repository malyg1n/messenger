import { useEffect, useRef, useState } from "react"
import type { Chat } from "@/entities/chat"
import type { ChatMessage, ViewMessage, LastMessageOverride } from "@/entities/message/model/types"
import { UsersList } from "@/entities/user"
import type { User } from "@/entities/user"
import api from "@/shared/api/client"
import { getAuthToken } from "@/shared/config/storage"
import { playNewMessageSound, warmupMessageSound } from "@/shared/lib/audio/newMessageSound"

type Props = {
  user: User
  onLogout: () => void
}

const PAGE_SIZE = 50
const DELIVERY_TIMEOUT_MS = 10000

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
  const deliveryTimeoutsRef = useRef<Record<string, number>>({})

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
  function toViewMessages(
    history: { sender_id: string; body: string; created_at?: string }[],
    chat: Chat
  ): ViewMessage[] {
    return history
      .map<ViewMessage>(m => ({
        body: m.body,
        text: m.sender_id === user.id ? "me: " + m.body : chat.title + ": " + m.body,
        isMe: m.sender_id === user.id,
        timestamp: formatTimestamp(m.created_at),
        createdAtMs: toCreatedAtMs(m.created_at),
        createdAt: m.created_at ?? new Date().toISOString(),
        status: "saved",
        clientMessageId: undefined
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
    const token = getAuthToken()
    const params = new URLSearchParams({
      user_id: user.id
    })
    if (token) {
      params.set("token", token)
    }
    return new WebSocket(`${wsUrl}?${params.toString()}`)
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
        const m = JSON.parse(e.data) as { sender_id: string; chat_id: string; body: string; created_at?: string, client_message_id?: string }
        const createdAt = m.created_at ?? new Date().toISOString()
        const isIncoming = m.sender_id !== user.id
        const isActiveChatMessage = m.chat_id === selectedChatIdRef.current
        if (!isIncoming && m.client_message_id) {
          const pendingTimeout = deliveryTimeoutsRef.current[m.client_message_id]
          if (pendingTimeout) {
            window.clearTimeout(pendingTimeout)
            delete deliveryTimeoutsRef.current[m.client_message_id]
          }
          setMessages(prev => {
            let isUpdated = false
            const next = prev.map(message => {
              if (isUpdated || !message.isMe || message.clientMessageId !== m.client_message_id) {
                return message
              }
              isUpdated = true
              return {
                ...message,
                status: "saved" as const,
                createdAt: createdAt,
                createdAtMs: toCreatedAtMs(createdAt),
                timestamp: formatTimestamp(createdAt)
              }
            })
            return isUpdated ? next : prev
          })
        }
        if (m.chat_id) {
          setLastMessageOverrides(prev => ({
            ...prev,
            [m.chat_id]: {
              last_message: m.body,
              last_message_at: createdAt
            }
          }))
        }
        if (isIncoming) {
          playNewMessageSound()
        }
        if (isIncoming && isActiveChatMessage) {
          // Если сообщение относится к открытому чату — сразу показываем его в ленте.
          setMessages(prev => [
            ...prev,
            {
              body: m.body,
              text: "them: " + m.body,
              isMe: false,
              timestamp: formatTimestamp(createdAt),
              createdAtMs: toCreatedAtMs(createdAt),
              createdAt: createdAt,
              status: "saved"
            }
          ])
          scrollToBottom()
        }
        if (isIncoming) {
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
      Object.values(deliveryTimeoutsRef.current).forEach(timeoutId => window.clearTimeout(timeoutId))
      deliveryTimeoutsRef.current = {}
      if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
        ws.close()
      }
      setSocket(null)
    }
  }, [user.id])

  useEffect(() => {
    // Браузеры требуют user gesture для старта звука.
    const enableAudio = () => {
      void warmupMessageSound()
    }

    window.addEventListener("pointerdown", enableAudio, { once: true })
    window.addEventListener("keydown", enableAudio, { once: true })

    return () => {
      window.removeEventListener("pointerdown", enableAudio)
      window.removeEventListener("keydown", enableAudio)
    }
  }, [])

  // send отправляет сообщение в websocket и сразу отражает его в интерфейсе.
  function scheduleDeliveryTimeout(clientMessageId: string) {
    deliveryTimeoutsRef.current[clientMessageId] = window.setTimeout(() => {
      setMessages(prev => {
        let isUpdated = false
        const next = prev.map(message => {
          if (isUpdated || message.clientMessageId !== clientMessageId || message.status !== "pending") {
            return message
          }
          isUpdated = true
          return {
            ...message,
            status: "failed" as const
          }
        })
        return isUpdated ? next : prev
      })
      delete deliveryTimeoutsRef.current[clientMessageId]
    }, DELIVERY_TIMEOUT_MS)
  }

  function send() {
    if (!socket || !chatId || !text.trim()) return

    const msg: ChatMessage = {
      chat_id: chatId,
      client_message_id: crypto.randomUUID(),
      body: text
    }

    socket.send(JSON.stringify(msg))
    scheduleDeliveryTimeout(msg.client_message_id)
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
        body: text,
        text: "me: " + text,
        isMe: true,
        timestamp: formatTimestamp(),
        createdAtMs: Date.now(),
        createdAt: new Date().toISOString(),
        status: "pending",
        clientMessageId: msg.client_message_id
      }
    ])

    setText("")
    scrollToBottom()
  }

  function retryMessage(messageIndex: number) {
    if (!socket || socket.readyState !== WebSocket.OPEN || !chatId) return
    const target = messages[messageIndex]
    if (!target || !target.isMe || target.status !== "failed") return

    const retryClientMessageId = target.clientMessageId ?? crypto.randomUUID()
    const pendingTimeout = deliveryTimeoutsRef.current[retryClientMessageId]
    if (pendingTimeout) {
      window.clearTimeout(pendingTimeout)
      delete deliveryTimeoutsRef.current[retryClientMessageId]
    }

    const retryCreatedAt = new Date().toISOString()
    const retryPayload: ChatMessage = {
      chat_id: chatId,
      client_message_id: retryClientMessageId,
      body: target.body
    }

    socket.send(JSON.stringify(retryPayload))
    scheduleDeliveryTimeout(retryClientMessageId)
    setChatsRefreshToken(prev => prev + 1)
    setLastMessageOverrides(prev => ({
      ...prev,
      [chatId]: {
        last_message: target.body,
        last_message_at: retryCreatedAt
      }
    }))
    setMessages(prev =>
      prev.map((message, index) =>
        index === messageIndex
          ? {
            ...message,
            status: "pending",
            clientMessageId: retryClientMessageId,
            createdAt: retryCreatedAt,
            createdAtMs: toCreatedAtMs(retryCreatedAt),
            timestamp: formatTimestamp(retryCreatedAt)
          }
          : message
      )
    )
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
                  <div
                    className={`message-bubble ${m.isMe ? "message-bubble-me" : "message-bubble-them"} ${m.status === "failed" ? "message-bubble-failed" : ""}`}
                  >
                    <div>{m.text}</div>
                    <div className="message-meta">{m.timestamp}</div>
                    {m.isMe && m.status === "failed" ? (
                      <button className="message-retry-btn" onClick={() => retryMessage(i)}>
                        Повторить
                      </button>
                    ) : null}
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
