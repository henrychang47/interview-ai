const workflowItems = ['輸入職位資訊', '產生面試問題', '錄音回答', '查看結果']

export default function App() {
  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto flex min-h-screen w-full max-w-5xl flex-col justify-center px-6 py-12">
        <div className="max-w-3xl">
          <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
            Interview Practice MVP
          </p>
          <h1 className="mt-4 text-4xl font-bold leading-tight sm:text-5xl">模擬面試應用</h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-slate-700">
            建立面試、產生題目、錄音回答，逐步打通 MVP 主流程。
          </p>
        </div>

        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {workflowItems.map((item, index) => (
            <div key={item} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-teal-100 text-sm font-bold text-teal-800">
                {index + 1}
              </div>
              <p className="mt-4 font-medium text-slate-900">{item}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}
