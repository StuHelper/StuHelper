// clients/web/src/design-system/tokens.ts
// Single source of truth — mirrors CSS custom properties in tailwind.css

export const colors = {
  primary: '#3f5ccb',
  primaryLight: '#7e9aff',
  accent: '#e87aac',
  accentLight: '#f09ec5',
  secondary: '#5ab8cc',
  success: '#52c07a',
  warning: '#e8a840',
  danger: '#d86060',
} as const

export const ratingColors: Record<number, string> = {
  1: 'var(--color-rating-1)',
  2: 'var(--color-rating-2)',
  3: 'var(--color-rating-3)',
  4: 'var(--color-rating-4)',
  5: 'var(--color-rating-5)',
} as const

export const animations = {
  easing: {
    smooth: 'cubic-bezier(0.4, 0.0, 0.2, 1)',
    bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
    spring: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
  },
  duration: {
    fast: 120,
    base: 200,
    slow: 300,
    slower: 500,
  },
} as const

export const glassEffects = {
  card: {
    background: 'var(--color-bg-glass)',
    backdropFilter: 'blur(12px) saturate(180%)',
  },
  navbar: {
    background: 'var(--color-bg-glass)',
    backdropFilter: 'blur(20px) saturate(180%)',
  },
  modal: {
    background: 'var(--color-bg-glass-heavy)',
    backdropFilter: 'blur(30px) saturate(200%)',
  },
} as const
