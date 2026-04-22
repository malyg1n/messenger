import { useState } from "react"
import api from "../api/client"
import { User } from "../types"

type Props = {
	onAuth: (user: User) => void
}

export default function Auth({ onAuth }: Props) {

	const [username, setUsername] = useState("")

	const [error, setError] = useState("")

	async function handleLogin() {

		try {

			const user = await api.login(username)

			localStorage.setItem(
				"user",
				JSON.stringify(user)
			)

			onAuth(user)

		} catch {

			setError("user not found")

		}

	}

	async function handleRegister() {

		try {

			const user = await api.register(username)

			localStorage.setItem(
				"user",
				JSON.stringify(user)
			)

			onAuth(user)

		} catch {

			setError("username taken")

		}

	}

	return (
		<div className="auth-shell">
			<div className="auth-card">
				<h1 className="auth-title">Messenger</h1>
				<p className="auth-subtitle">Вход или регистрация</p>

				<input
					className="auth-input"
					placeholder="username"
					value={username}
					onChange={e => setUsername(e.target.value)}
				/>

				<div className="auth-actions">
					<button className="btn btn-primary" onClick={handleLogin}>
						Войти
					</button>

					<button className="btn btn-secondary" onClick={handleRegister}>
						Регистрация
					</button>
				</div>

				{error ? <div className="auth-error">{error}</div> : null}
			</div>
		</div>

	)

}