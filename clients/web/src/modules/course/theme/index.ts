/** Get CSS variable for rating color by value 1–5 */
export function getRatingColor(value: number): string {
  if (value <= 1) return 'var(--color-rating-1)'
  if (value <= 2) return 'var(--color-rating-2)'
  if (value <= 3) return 'var(--color-rating-3)'
  if (value <= 4) return 'var(--color-rating-4)'
  return 'var(--color-rating-5)'
}
