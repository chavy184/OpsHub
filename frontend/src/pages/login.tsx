import { FormEvent, useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Toaster, toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { login } from '@/api/auth'
import { isAuthenticated, saveAuth } from '@/lib/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)

  if (isAuthenticated()) {
    return <Navigate to="/services" replace />
  }

  const from = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname || '/services'

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (!username.trim() || !password) {
      toast.error('请输入用户名和密码')
      return
    }

    setSubmitting(true)
    try {
      const result = await login({ username: username.trim(), password })
      saveAuth(result.token, result.username, result.expires_at)
      toast.success('登录成功')
      navigate(from, { replace: true })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '登录失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center px-4">
      <Toaster position="top-center" richColors />
      <div className="w-full max-w-sm rounded-xl border bg-background p-6 shadow-sm">
        <div className="mb-6 flex flex-col items-center gap-2 text-center">
          <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground font-bold">
            O
          </div>
          <div>
            <h1 className="text-xl font-semibold">登录 OpsHub</h1>
            <p className="text-sm text-muted-foreground">使用超级管理员账号进入系统</p>
          </div>
        </div>

        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="username">用户名</Label>
            <Input
              id="username"
              value={username}
              autoComplete="username"
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="password">密码</Label>
            <Input
              id="password"
              type="password"
              value={password}
              autoComplete="current-password"
              placeholder="请输入密码"
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <Button type="submit" className="mt-2 w-full" disabled={submitting}>
            {submitting ? '登录中…' : '登录'}
          </Button>
        </form>
      </div>
    </div>
  )
}
