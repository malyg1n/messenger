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

export default {
	register,
	getUsers
}