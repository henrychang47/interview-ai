import type { AnchorHTMLAttributes, ButtonHTMLAttributes, HTMLAttributes, ReactNode } from 'react'

type IconProps = {
  name: string
  filled?: boolean
  className?: string
}

export function Icon({ name, filled = false, className = '' }: IconProps) {
  return (
    <span
      aria-hidden="true"
      className={`material-symbols-outlined ${filled ? 'material-symbols-filled' : ''} ${className}`}
    >
      {name}
    </span>
  )
}

type ButtonTone = 'primary' | 'secondary' | 'ghost' | 'danger'

const buttonToneClasses: Record<ButtonTone, string> = {
  primary:
    'bg-primary text-on-primary shadow-calm hover:bg-primary-container hover:text-on-primary-container disabled:bg-surface-container-high disabled:text-on-surface-variant',
  secondary:
    'border border-outline-variant bg-surface-container-lowest text-primary hover:bg-primary/5 disabled:text-on-surface-variant',
  ghost:
    'text-on-surface-variant hover:bg-surface-container-low hover:text-primary disabled:text-outline',
  danger:
    'border border-error/30 bg-error-container text-on-error-container hover:bg-error/15 disabled:text-on-surface-variant',
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: ButtonTone
  icon?: string
}

export function Button({ tone = 'primary', icon, className = '', children, ...props }: ButtonProps) {
  return (
    <button
      className={`inline-flex min-h-11 items-center justify-center gap-sm rounded-lg px-md py-sm text-label-md font-bold transition-all active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70 ${buttonToneClasses[tone]} ${className}`}
      {...props}
    >
      {icon ? <Icon name={icon} className="text-[18px]" /> : null}
      {children}
    </button>
  )
}

type LinkButtonProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  tone?: ButtonTone
  icon?: string
}

export function LinkButton({
  tone = 'primary',
  icon,
  className = '',
  children,
  ...props
}: LinkButtonProps) {
  return (
    <a
      className={`inline-flex min-h-11 items-center justify-center gap-sm rounded-lg px-md py-sm text-label-md font-bold transition-all active:scale-[0.98] ${buttonToneClasses[tone]} ${className}`}
      {...props}
    >
      {icon ? <Icon name={icon} className="text-[18px]" /> : null}
      {children}
    </a>
  )
}

type CardProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
}

export function Card({ className = '', children, ...props }: CardProps) {
  return (
    <div
      className={`rounded-xl border border-outline-variant bg-surface-container-lowest shadow-calm ${className}`}
      {...props}
    >
      {children}
    </div>
  )
}

export function TopBar({
  maxWidth = 'max-w-container-max',
  action,
}: {
  maxWidth?: string
  action?: {
    href: string
    label: string
    icon?: string
  }
}) {
  return (
    <header className="sticky top-0 z-50 border-b border-outline-variant bg-surface/90 backdrop-blur-md">
      <div
        className={`mx-auto flex h-16 w-full ${maxWidth} items-center justify-between gap-md px-margin-mobile md:px-margin-desktop`}
      >
        <a href="/" className="flex min-w-0 items-center gap-base font-headline text-headline-md font-bold text-primary">
          <span className="truncate">AI模擬面試</span>
        </a>
        {action ? (
          <a
            href={action.href}
            className="inline-flex shrink-0 items-center gap-xs rounded-lg px-sm py-xs text-label-md font-bold text-on-surface-variant hover:bg-surface-container-low hover:text-primary"
          >
            {action.icon ? <Icon name={action.icon} className="text-[18px]" /> : null}
            {action.label}
          </a>
        ) : null}
      </div>
    </header>
  )
}

type StatusTone = 'neutral' | 'primary' | 'success' | 'danger'

const statusToneClasses: Record<StatusTone, string> = {
  neutral: 'border-outline-variant bg-surface-container-low text-on-surface-variant',
  primary: 'border-primary/15 bg-primary/5 text-primary',
  success: 'border-green-200 bg-green-100 text-green-800',
  danger: 'border-error/20 bg-error-container text-on-error-container',
}

export function StatusBadge({
  children,
  tone = 'neutral',
  className = '',
}: {
  children: ReactNode
  tone?: StatusTone
  className?: string
}) {
  return (
    <span
      className={`inline-flex w-fit items-center gap-xs rounded-full border px-sm py-xs text-label-md font-bold ${statusToneClasses[tone]} ${className}`}
    >
      {children}
    </span>
  )
}

export function StepProgress({
  currentStep,
  steps,
}: {
  currentStep: number
  steps: [string, string]
}) {
  const percent = currentStep === 1 ? 50 : 100

  return (
    <div className="flex items-center gap-sm">
      {steps.map((label, index) => {
        const stepNumber = index + 1
        const isActive = currentStep >= stepNumber

        return (
          <div
            key={label}
            className={`flex min-w-0 flex-1 items-center gap-sm ${index === 1 ? 'justify-end' : ''
              } ${isActive ? '' : 'opacity-50'}`}
          >
            <div
              className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-label-md font-bold ${isActive
                ? 'bg-primary text-on-primary'
                : 'border border-outline text-on-surface-variant'
                }`}
            >
              {stepNumber}
            </div>
            <span
              className={`truncate text-label-md font-bold ${isActive ? 'text-primary' : 'text-on-surface-variant'
                }`}
            >
              {label}
            </span>
          </div>
        )
      })}
      <div className="absolute left-0 right-0 top-1/2 -z-10 mx-[34%] h-1.5 -translate-y-1/2 overflow-hidden rounded-full bg-surface-container-highest">
        <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${percent}%` }} />
      </div>
    </div>
  )
}

export function PageShell({
  children,
  maxWidth = 'max-w-readable',
  className = '',
}: {
  children: ReactNode
  maxWidth?: string
  className?: string
}) {
  return (
    <main className={`min-h-screen bg-background text-on-background ${className}`}>
      <div className={`mx-auto w-full ${maxWidth} px-margin-mobile py-lg md:px-margin-desktop md:py-xl`}>
        {children}
      </div>
    </main>
  )
}
