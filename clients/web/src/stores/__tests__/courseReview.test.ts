import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockGetDepartments = vi.fn()
const mockGetCourses = vi.fn()

vi.mock('@/api', () => ({
  api: {
    course: {
      getDepartments: mockGetDepartments,
      getCourses: mockGetCourses,
    },
  },
}))

vi.mock('@/api/errors', () => ({
  classifyApiError: (error: unknown, options: { unknownType: string, fallbackMessage: string }) => ({
    type: options.unknownType,
    message: error instanceof Error ? error.message : options.fallbackMessage,
  }),
}))

vi.mock('@/i18n', () => ({
  default: {
    global: { t: (key: string) => key },
  },
}))

const { useCourseStore } = await import('@/stores/courseReview')

describe('useCourseStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchDepartments', () => {
    it('loads departments from API', async () => {
      const depts = [{ id: 1, name: 'CS' }, { id: 2, name: 'Math' }]
      mockGetDepartments.mockResolvedValue({ data: { data: depts } })

      const store = useCourseStore()
      const result = await store.fetchDepartments()

      expect(result).toEqual(depts)
      expect(store.departments).toEqual(depts)
      expect(store.departmentsLoading).toBe(false)
      expect(store.departmentsError).toBeNull()
    })

    it('returns cached data on second call', async () => {
      const depts = [{ id: 1, name: 'CS' }]
      mockGetDepartments.mockResolvedValue({ data: { data: depts } })

      const store = useCourseStore()
      await store.fetchDepartments()

      mockGetDepartments.mockClear()
      const cached = await store.fetchDepartments()

      expect(mockGetDepartments).not.toHaveBeenCalled()
      expect(cached).toEqual(depts)
    })

    it('passes category filter to API', async () => {
      mockGetDepartments.mockResolvedValue({ data: { data: [] } })

      const store = useCourseStore()
      await store.fetchDepartments('engineering')

      expect(mockGetDepartments).toHaveBeenCalledWith({ category: 'engineering' })
    })

    it('sets error on API failure', async () => {
      mockGetDepartments.mockRejectedValue(new Error('network down'))

      const store = useCourseStore()
      await expect(store.fetchDepartments()).rejects.toThrow()

      expect(store.departmentsError).toBeTruthy()
      expect(store.departments).toEqual([])
      expect(store.departmentsLoading).toBe(false)
    })

    it('fails closed when departments response is malformed', async () => {
      mockGetDepartments.mockResolvedValue({ data: { data: null } })

      const store = useCourseStore()
      await expect(store.fetchDepartments()).rejects.toThrow(
        'Invalid departments response',
      )

      expect(store.departmentsError).toBeTruthy()
      expect(store.departments).toEqual([])
      expect(store.departmentsLoading).toBe(false)
    })
  })

  describe('fetchCourses', () => {
    it('loads courses for a department', async () => {
      const courses = [{ id: 10, name: 'Intro CS' }]
      mockGetCourses.mockResolvedValue({
        data: { data: { list: courses } },
      })

      const store = useCourseStore()
      const result = await store.fetchCourses(1)

      expect(result).toEqual(courses)
      expect(store.courses).toEqual(courses)
      expect(store.coursesLoading).toBe(false)
    })

    it('returns cached courses on second call', async () => {
      const courses = [{ id: 10, name: 'Intro CS' }]
      mockGetCourses.mockResolvedValue({
        data: { data: { list: courses } },
      })

      const store = useCourseStore()
      await store.fetchCourses(1)

      mockGetCourses.mockClear()
      const cached = await store.fetchCourses(1)

      expect(mockGetCourses).not.toHaveBeenCalled()
      expect(cached).toEqual(courses)
    })

    it('sets error on API failure', async () => {
      mockGetCourses.mockRejectedValue(new Error('fail'))

      const store = useCourseStore()
      await expect(store.fetchCourses(1)).rejects.toThrow()

      expect(store.coursesError).toBeTruthy()
      expect(store.courses).toEqual([])
    })

    it('fails closed when courses response is missing list data', async () => {
      mockGetCourses.mockResolvedValue({ data: { data: null } })

      const store = useCourseStore()
      await expect(store.fetchCourses(1)).rejects.toThrow(
        'Invalid courses response',
      )

      expect(store.coursesError).toBeTruthy()
      expect(store.courses).toEqual([])
      expect(store.coursesLoading).toBe(false)
    })
  })

  describe('loading computed', () => {
    it('reflects combined loading state', () => {
      const store = useCourseStore()
      expect(store.loading).toBe(false)
    })
  })

  describe('clearCache', () => {
    it('forces fresh API call after clearing', async () => {
      const depts = [{ id: 1, name: 'CS' }]
      mockGetDepartments.mockResolvedValue({ data: { data: depts } })

      const store = useCourseStore()
      await store.fetchDepartments()
      store.clearCache()

      mockGetDepartments.mockClear()
      mockGetDepartments.mockResolvedValue({ data: { data: depts } })
      await store.fetchDepartments()

      expect(mockGetDepartments).toHaveBeenCalled()
    })
  })

  describe('invalidateCourseCache', () => {
    it('invalidates specific department course cache', async () => {
      const courses = [{ id: 10, name: 'x' }]
      mockGetCourses.mockResolvedValue({ data: { data: { list: courses } } })

      const store = useCourseStore()
      await store.fetchCourses(1)
      store.invalidateCourseCache(1)

      mockGetCourses.mockClear()
      mockGetCourses.mockResolvedValue({ data: { data: { list: courses } } })
      await store.fetchCourses(1)

      expect(mockGetCourses).toHaveBeenCalled()
    })
  })

  describe('clearErrors', () => {
    it('clears all error states', async () => {
      mockGetDepartments.mockRejectedValue(new Error('fail'))
      mockGetCourses.mockRejectedValue(new Error('fail'))

      const store = useCourseStore()
      await store.fetchDepartments().catch(() => {})
      await store.fetchCourses(1).catch(() => {})

      expect(store.departmentsError).toBeTruthy()
      expect(store.coursesError).toBeTruthy()

      store.clearErrors()

      expect(store.departmentsError).toBeNull()
      expect(store.coursesError).toBeNull()
    })
  })

  describe('reset', () => {
    it('resets all state including caches', async () => {
      mockGetDepartments.mockResolvedValue({ data: { data: [{ id: 1, name: 'CS' }] } })
      mockGetCourses.mockResolvedValue({ data: { data: { list: [{ id: 10, name: 'x' }] } } })

      const store = useCourseStore()
      await store.fetchDepartments()
      await store.fetchCourses(1)

      store.reset()

      expect(store.departments).toEqual([])
      expect(store.courses).toEqual([])
      expect(store.departmentsLoading).toBe(false)
      expect(store.coursesLoading).toBe(false)
      expect(store.departmentsError).toBeNull()
      expect(store.coursesError).toBeNull()
    })
  })
})
