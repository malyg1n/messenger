import { useEffect, useState } from "react"
import { ChatMessage, User } from "../types"
import UsersList from "./UsersList"

type Props = {
	user: User
}

export default function Chat({ user }: Props) {
	const [selectedUser, setSelectedUser] = useState<User | null>(null)
	const [text, setText] = useState("")
	const [messages, setMessages] = useState<string[]>([])
	const [socket, setSocket] = useState<WebSocket | null>(null)

	useEffect(() => {
		const ws = new WebSocket("ws://localhost:8080/ws")

		ws.onmessage = e => {
			setMessages(prev => [...prev, e.data])
		}

		setSocket(ws)

		return () => ws.close()
	}, [])

	function send() {
		if (!socket || !selectedUser) return

		const msg: ChatMessage = {
			sender_id: user.id,
			receiver_id: selectedUser.id,
			body: text
		}

		socket.send(JSON.stringify(msg))

		setMessages(prev => [
			...prev,
			"me: " + text
		])

		setText("")
	}

	return (
		<div style={{ display: "flex", gap: 20 }}>

			<UsersList
				currentUser={user}
				onSelect={setSelectedUser}
			/>

			<div style={{ flex: 1 }}>

				<h3>user: {user.username}</h3>

				<div>
					chat with:
					{selectedUser?.username}
				</div>

				<div className="messages">
					{messages.map((m, i) => (
						<div key={i}>{m}</div>
					))}
				</div>

				<input
					value={text}
					onChange={e => setText(e.target.value)}
				/>

				<button onClick={send}>
					send
				</button>

			</div>

		</div>
	)
}