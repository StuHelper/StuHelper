import { ref, watch, onUnmounted } from 'vue'
import { api } from '@/api'
import type { Course } from '@/types/course'

const SEARCH_DEBOUNCE_MS = 300

export function useCourseSearch() {
  const courseQuery = ref('')
  const courseResults = ref<Course[]>([])
  const highlightedIndex = ref(-1)

  let searchTimer: ReturnType<typeof setTimeout> | null = null
  let searchController: AbortController | undefined

  watch(courseQuery, (val) => {
    if (searchTimer) clearTimeout(searchTimer)
    highlightedIndex.value = -1
    const q = val.trim()
    if (!q) {
      if (searchController) { searchController.abort(); searchController = undefined }
      courseResults.value = []
      return
    }

    searchTimer = setTimeout(async () => {
      if (searchController) searchController.abort()
      const controller = new AbortController()
      searchController = controller

      try {
        const res = await api.course.searchCourses(q, 8, { signal: controller.signal })
        if (controller.signal.aborted) return
        const list = res.data?.data?.list || []
        const seen = new Set<number>()
        courseResults.value = list.filter((c: Course) => {
          if (seen.has(c.id)) return false
          seen.add(c.id)
          return true
        })
        highlightedIndex.value = -1
      } catch (err) {
        if (controller.signal.aborted) return
        if (import.meta.env.DEV) { console.warn('[useCourseSearch] Course search failed:', err) }
        courseResults.value = []
      }
    }, SEARCH_DEBOUNCE_MS)
  })

  function handleSearchKeydown(e: KeyboardEvent, onSelect: (course: Course) => void) {
    const len = courseResults.value.length
    if (len === 0) return

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        highlightedIndex.value = (highlightedIndex.value + 1) % len
        break
      case 'ArrowUp':
        e.preventDefault()
        highlightedIndex.value = (highlightedIndex.value - 1 + len) % len
        break
      case 'Enter':
        e.preventDefault()
        if (highlightedIndex.value >= 0 && highlightedIndex.value < len) {
          onSelect(courseResults.value[highlightedIndex.value])
        }
        break
    }
  }

  function reset() {
    courseQuery.value = ''
    courseResults.value = []
    highlightedIndex.value = -1
  }

  function cleanup() {
    if (searchTimer) clearTimeout(searchTimer)
    if (searchController) { searchController.abort(); searchController = undefined }
  }

  onUnmounted(cleanup)

  return {
    courseQuery,
    courseResults,
    highlightedIndex,
    handleSearchKeydown,
    reset,
    cleanup,
  }
}
