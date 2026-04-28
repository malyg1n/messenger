export type User = {
  id: string
  username: string,
  token: string
}

export type AuthResponse = {
  user: User
  token: string
}
