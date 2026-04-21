import { useEffect, useState } from "react"
import { ChatMessage, User } from "../types"
import UsersList from "./UsersList"
import api from "../api/client"

type Props = {
	user: User
}

export default function Chat({ user }: Props) {

	const [chatId, setChatId] = useState<string | null>(null)
	const [selectedUser, setSelectedUser] = useState<User | null>(null)
	const [text, setText] = useState("")
	const [messages, setMessages] = useState<string[]>([])
	const [socket, setSocket] = useState<WebSocket | null>(null)

	useEffect(() => {
		const ws = new WebSocket(`ws://localhost:8080/ws?user_id=${user.id}`)
		ws.onmessage = e => {
			const m = JSON.parse(e.data)
			if (m.sender_id !== user.id) {
				setMessages(prev => [
					...prev,
					"them: " + m.body
				])
			}
		}

		setSocket(ws)
		return () => ws.close()
	}, [])

	function send() {

		if (!socket || !chatId) return

		const msg: ChatMessage = {

			chat_id: chatId,
			sender_id: user.id,
			body: text

		}

		socket.send(JSON.stringify(msg))

		setMessages(prev => [
			...prev,
			"me: " + text
		])

		setText("")

	}

	async function selectUser(u: User) {

		setSelectedUser(u)
	
		const res = await api.createDirectChat(
			user.id,
			u.id
		)
	
		setChatId(res.chat_id)
	
		const history = await api.getMessages(
			res.chat_id
		)
		setMessages([])
		if (!history || !history.length) {
			return
		}
	
		setMessages(
			history.map(
				(				m: { sender_id: string; body: string }) =>
					m.sender_id === user.id
						? "me: " + m.body
						: u.username + ": " + m.body
			)
		)
	
	}

	return (

		<div style={{ display: "flex", gap: 20 }}>

			<UsersList
				currentUser={user}
				onSelect={selectUser}
			/>

			<div style={{ flex: 1 }}>

				<h3>
					user: {user.username}
				</h3>

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