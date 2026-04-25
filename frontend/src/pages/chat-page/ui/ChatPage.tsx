import type { User } from "@/entities/user"
import { ChatWidget } from "@/widgets/chat"

type Props = {
  user: User
  onLogout: () => void
}

// ChatPage — страница-обертка над виджетом чата.
export default function ChatPage({ user, onLogout }: Props) {
  return <ChatWidget user={user} onLogout={onLogout} />
}
