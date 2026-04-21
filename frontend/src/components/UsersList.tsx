import { useEffect, useState } from "react"
import api from "../api/client"
import { User } from "../types"

type Props = {
	currentUser: User
	onSelect: (user: User) => void
}

export default function UsersList({ currentUser, onSelect }: Props) {
	const [users, setUsers] = useState<User[]>([])

	useEffect(() => {
		api.getUsers().then(setUsers)
	}, [])

	return (
		<div>
			<h4>users</h4>

			{users
				.filter(u => u.id !== currentUser.id)
				.map(u => (
					<div
						key={u.id}
						onClick={() => onSelect(u)}
						style={{ cursor: "pointer", padding: 5 }}
					>
						{u.username}
					</div>
				))}
		</div>
	)
}