import { User } from "../types"

const API = "http://localhost:8081"

async function register(username: string): Promise<User> {
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

async function getUsers(): Promise<User[]> {
	const res = await fetch(API + "/users")

	return res.json()
}

async function createDirectChat(
	userId: string,
	targetUserId: string
) {

	const res = await fetch(API + "/chats/direct", {
		method: "POST",
		headers: {
			"Content-Type": "application/json"
		},
		body: JSON.stringify({
			user_id: userId,
			target_user_id: targetUserId
		})
	})

	return res.json()
}

async function login(username: string): Promise<User> {
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

async function getMessages(chatId: string) {

	const res = await fetch(
		API + "/messages?chat_id=" + chatId
	)

	return res.json()

}

export default {
	register,
	getUsers,
    createDirectChat,
	login,
	getMessages
}