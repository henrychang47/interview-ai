import { FormEvent, useState } from 'react'

import { createInterview } from '../api/interviews'
import type { CreateInterviewRequest } from '../types/interview'

type NewInterviewPageProps = {
  onCreated: (interviewID: string) => void
}

const initialForm: CreateInterviewRequest = {
  job_title: '',
  job_description: '',
  user_profile: '',
  question_count: 3,
}

export default function NewInterviewPage({ onCreated }: NewInterviewPageProps) {
  const [form, setForm] = useState<CreateInterviewRequest>(initialForm)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setError(null)

    try {
      const response = await createInterview({
        job_title: form.job_title.trim(),
        job_description: form.job_description.trim(),
        user_profile: form.user_profile.trim(),
        question_count: form.question_count,
      })
      onCreated(response.id)
    } catch (error) {
      setError(error instanceof Error ? error.message : '建立面試失敗')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-3xl px-6 py-10">
        <a href="/" className="text-sm font-medium text-teal-700 hover:text-teal-800">
          返回首頁
        </a>

        <div className="mt-8">
          <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
            Interview Setup
          </p>
          <h1 className="mt-3 text-3xl font-bold leading-tight">建立模擬面試</h1>
        </div>

        <form
          onSubmit={handleSubmit}
          className="mt-8 space-y-6 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
        >
          {error ? (
            <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          <label className="block">
            <span className="text-sm font-medium text-slate-800">職位名稱</span>
            <input
              className="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.job_title}
              onChange={(event) =>
                setForm((current) => ({ ...current, job_title: event.target.value }))
              }
            />
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">職位要求及說明</span>
            <textarea
              className="mt-2 min-h-32 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.job_description}
              onChange={(event) =>
                setForm((current) => ({ ...current, job_description: event.target.value }))
              }
            />
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">個人資訊</span>
            <textarea
              className="mt-2 min-h-32 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.user_profile}
              onChange={(event) =>
                setForm((current) => ({ ...current, user_profile: event.target.value }))
              }
            />
          </label>

          <label className="block max-w-40">
            <span className="text-sm font-medium text-slate-800">題目數量</span>
            <input
              type="number"
              min={1}
              max={10}
              className="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.question_count}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  question_count: Number(event.target.value),
                }))
              }
            />
          </label>

          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {isSubmitting ? '建立中...' : '建立面試'}
          </button>
        </form>
      </section>
    </main>
  )
}
