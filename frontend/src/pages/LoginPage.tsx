import { zodResolver } from '@hookform/resolvers/zod'
import { motion } from 'framer-motion'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router-dom'
import { z } from 'zod'
import { login, persistAuthTokens } from '../api/auth'
import { toApiError } from '../api/client'
import { AuthInput } from '../components/AuthInput'
import { AuthLayout } from '../components/AuthLayout'

const loginSchema = z.object({
  email: z.email('Введите корректный email'),
  password: z.string().min(6, 'Минимум 6 символов'),
})

type LoginFormValues = z.infer<typeof loginSchema>

export function LoginPage() {
  const navigate = useNavigate()
  const [serverMessage, setServerMessage] = useState<string>('')
  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
    },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const tokens = await login(values)
      persistAuthTokens(tokens)
      navigate('/', { replace: true })
    } catch (error) {
      setServerMessage(toApiError(error).message)
    }
  })

  return (
    <AuthLayout title="Welcome to GoDrive" subtitle="Sign in to access your cloud workspace.">
      <form onSubmit={onSubmit} className="space-y-4">
        <AuthInput
          label="Email"
          type="email"
          autoComplete="email"
          placeholder="you@godrive.io"
          error={errors.email?.message}
          {...registerField('email')}
        />

        <AuthInput
          label="Password"
          type="password"
          autoComplete="current-password"
          placeholder="Enter your secure password"
          error={errors.password?.message}
          {...registerField('password')}
        />

        <motion.button
          whileHover={{ scale: 1.01 }}
          whileTap={{ scale: 0.99 }}
          disabled={isSubmitting}
          className="mt-2 w-full rounded-xl bg-slate-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
          type="submit"
        >
          {isSubmitting ? 'Signing in...' : 'Sign In'}
        </motion.button>
      </form>

      {serverMessage ? <p className="mt-4 text-sm text-slate-600">{serverMessage}</p> : null}

      <p className="mt-6 text-sm text-slate-500">
        No account yet?{' '}
        <Link className="font-semibold text-blue-600 transition hover:text-blue-500" to="/register">
          Create one now
        </Link>
      </p>
    </AuthLayout>
  )
}
