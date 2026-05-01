import type { Chat } from "@/entities/chat"
import type { User } from "@/entities/user"
import { AuthResponse } from "@/entities/user/model/types"
import { getAuthToken } from "@/shared/config/storage"

const API = import.meta.env.VITE_API_URL ?? "http://localhost:8081"

// getAuthHeaders читает токен текущего пользователя из localStorage.
function getAuthHeaders(): HeadersInit {
  const token = getAuthToken()
  if (!token) return {}
  return {
    "Authorization": `Bearer ${token}`
  }
}

// register регистрирует нового пользователя в api-service.
async function register(username: string): Promise<AuthResponse> {
  const res = await fetch(API + "/register", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ username })
  })

  if (!res.ok) {
    throw new Error("username taken")
  }

  return res.json()
}

// getUsers возвращает список пользователей для создания диалогов.
async function getUsers(): Promise<User[]> {
  const res = await fetch(API + "/users", {
    headers: {
      ...getAuthHeaders()
    }
  })
  return res.json()
}

// createDirectChat создает или возвращает существующий direct-чат.
async function createDirectChat(targetUserId: string) {
  const res = await fetch(API + "/chats/direct", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...getAuthHeaders()
    },
    body: JSON.stringify({
      target_user_id: targetUserId
    })
  })

  return res.json()
}

// login выполняет вход по username.
async function login(username: string): Promise<AuthResponse> {
  const res = await fetch(API + "/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      username
    })
  })

  if (!res.ok) {
    throw new Error("user not found")
  }
  return res.json()
}

// getMessages запрашивает историю сообщений чата с пагинацией.
async function getMessages(chatId: string, limit = 50, before?: string) {
  const res = await fetch(API + "/messages?chat_id=" + chatId + "&limit=" + limit + "&before=" + before, {
    headers: {
      ...getAuthHeaders()
    }
  })
  return res.json()
}

// getChats возвращает список чатов пользователя.
async function getChats(): Promise<Chat[]> {
  const res = await fetch(API + "/chats", {
    headers: {
      ...getAuthHeaders()
    }
  })
  return res.json()
}

export default {
  register,
  getUsers,
  createDirectChat,
  login,
  getMessages,
  getChats
}
