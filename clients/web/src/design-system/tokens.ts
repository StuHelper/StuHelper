// clients/web/src/design-system/tokens.ts
export const glassEffects = {
  card: {
    background: 'rgba(255, 255, 255, 0.1)',
    backdropFilter: 'blur(10px) saturate(180%)',
    border: '1px solid rgba(255, 255, 255, 0.2)',
    boxShadow: '0 8px 32px 0 rgba(31, 38, 135, 0.15)'
  },
  navbar: {
    background: 'rgba(255, 255, 255, 0.8)',
    backdropFilter: 'blur(20px) saturate(180%)'
  },
  modal: {
    background: 'rgba(255, 255, 255, 0.95)',
    backdropFilter: 'blur(30px) saturate(200%)'
  }
} as const

export const colors = {
  primary: '#ec4899',
  secondary: '#3b82f6'
} as const

export const animations = {
  easing: {
    smooth: 'cubic-bezier(0.4, 0.0, 0.2, 1)',
    bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)'
  },
  duration: {
    fast: 150,
    normal: 300,
    slow: 500,
    page: 600
  }
} as const
