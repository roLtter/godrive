import { zodResolver } from '@hookform/resolvers/zod'
import { motion } from 'framer-motion'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { register } from '../api/auth'
import { toApiError } from '../api/client'
import { AuthInput } from '../components/AuthInput'
import { AuthLayout } from '../components/AuthLayout'

const registerSchema = z
  .object({
    email: z.email('Введите корректный email'),
    password: z
      .string()
      .min(8, 'Минимум 8 символов')
      .regex(/[A-Z]/, 'Добавьте хотя бы одну заглавную букву')
      .regex(/[0-9]/, 'Добавьте хотя бы одну цифру'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: 'Пароли не совпадают',
    path: ['confirmPassword'],
  })

type RegisterFormValues = z.infer<typeof registerSchema>

export function RegisterPage() {
  const [serverMessage, setServerMessage] = useState<string>('')
  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      email: '',
      password: '',
      confirmPassword: '',
    },
  })

  const onSubmit = handleSubmit(async ({ email, password }) => {
    try {
      await register({ email, password })
      setServerMessage('Регистрация успешна. Теперь можно войти в аккаунт.')
    } catch (error) {
      setServerMessage(toApiError(error).message)
    }
  })

  return (
    <AuthLayout title="Create Your GoDrive" subtitle="Build your private cloud in under a minute.">
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
          autoComplete="new-password"
          placeholder="8+ symbols, uppercase and number"
          error={errors.password?.message}
          {...registerField('password')}
        />

        <AuthInput
          label="Confirm password"
          type="password"
          autoComplete="new-password"
          placeholder="Repeat your password"
          error={errors.confirmPassword?.message}
          {...registerField('confirmPassword')}
        />

        <motion.button
          whileHover={{ scale: 1.01 }}
          whileTap={{ scale: 0.99 }}
          disabled={isSubmitting}
          className="mt-2 w-full rounded-xl bg-slate-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
          type="submit"
        >
          {isSubmitting ? 'Creating account...' : 'Create Account'}
        </motion.button>
      </form>

      {serverMessage ? <p className="mt-4 text-sm text-slate-600">{serverMessage}</p> : null}

      <p className="mt-6 text-sm text-slate-500">
        Already have an account?{' '}
        <Link className="font-semibold text-blue-600 transition hover:text-blue-500" to="/login">
          Sign in
        </Link>
      </p>
    </AuthLayout>
  )
}
