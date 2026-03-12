export interface TermOptionSource {
  id: string
  name: string
  isCurrent: boolean
}

export interface TermOption {
  id: string
  name: string
}

export function buildTermOptions(input: TermOptionSource[]): TermOption[] {
  return [...input]
    .sort((a, b) => {
      if (a.isCurrent !== b.isCurrent) {
        return a.isCurrent ? -1 : 1
      }
      return a.id < b.id ? 1 : -1
    })
    .map((item) => ({ id: item.id, name: item.name }))
}
