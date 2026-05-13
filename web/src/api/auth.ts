import axios from 'axios'

export async function loginAPI(username: string, password: string): Promise<string> {
  const res = await axios.post('/api/v1/auth/login', { username, password })
  return res.data.token as string
}
