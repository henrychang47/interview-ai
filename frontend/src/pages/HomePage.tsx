import { Card, Icon, LinkButton, TopBar } from '../components/ui'

const workflowItems = [
  {
    icon: 'draft',
    title: '1. 建立面試',
    description: '輸入目標職位、職缺描述與個人背景，建立專屬練習情境。',
  },
  {
    icon: 'auto_awesome',
    title: '2. 產生題目',
    description: '系統根據您的設定生成面試題，支援 mock mode 與已設定的 AI 服務。',
  },
  {
    icon: 'record_voice_over',
    title: '3. 錄音回答',
    description: '逐題聆聽題目並使用瀏覽器錄音，保留真實面試的節奏感。',
  },
  {
    icon: 'query_stats',
    title: '4. 查看結果',
    description: '回顧每一題題目與回答音檔，確認這次練習的完整紀錄。',
  },
]

export default function HomePage() {
  return (
    <>
      <TopBar />
      <main className="min-h-screen overflow-x-hidden bg-background text-on-background">
        <section className="mx-auto w-full max-w-container-max px-margin-mobile pb-xl pt-xl md:px-margin-desktop">
          <div className="flex flex-col items-center text-center">
            <div className="mb-md inline-flex rounded-full border border-primary/10 bg-primary/5 px-md py-xs">
              <span className="text-label-md font-bold uppercase text-primary">AI-Powered Readiness</span>
            </div>
            <h1 className="max-w-4xl font-headline text-headline-xl-mobile font-bold text-on-surface md:text-headline-xl">
              提升您的面試表現，隨時隨地與 AI 導師練習
            </h1>
            <p className="mt-md max-w-2xl text-body-lg text-on-surface-variant">
              四個步驟，完成從題目生成到錄音回顧的模擬面試流程。
            </p>
            <LinkButton href="/interviews/new" icon="play_arrow" className="mt-xl w-full sm:w-auto">
              建立新的模擬面試
            </LinkButton>
          </div>

          <section className="py-xl">
            <div className="mb-xl text-center">
              <h2 className="font-headline text-headline-lg text-on-surface">核心流程</h2>
              <div className="mx-auto mt-sm h-1 w-12 rounded-full bg-primary/60" />
            </div>
            <div className="grid grid-cols-1 gap-md sm:grid-cols-2 lg:grid-cols-4">
              {workflowItems.map((item) => (
                <Card key={item.title} className="calm-card-hover flex h-full flex-col items-start p-lg">
                  <div className="mb-md flex h-14 w-14 items-center justify-center rounded-xl border border-primary/10 bg-primary/5 text-primary">
                    <Icon name={item.icon} className="text-[28px]" />
                  </div>
                  <h3 className="font-headline text-headline-sm text-on-surface">{item.title}</h3>
                  <p className="mt-sm text-body-sm leading-relaxed text-on-surface-variant">
                    {item.description}
                  </p>
                </Card>
              ))}
            </div>
          </section>
        </section>
      </main>
    </>
  )
}
