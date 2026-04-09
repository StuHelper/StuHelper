import { ref, computed, onUnmounted } from 'vue'
import { api } from '@/api'
import type { TeacherStats } from '@/types/course'

export function useTeacherSelect() {
  const teacherList = ref<TeacherStats[]>([])
  const selectedTeacherID = ref<number | null>(null)
  const teacherQuery = ref('')
  const teacherDropdownOpen = ref(false)
  const loadingTeachers = ref(false)

  const filteredTeachers = computed(() => {
    const q = teacherQuery.value.trim().toLowerCase()
    if (!q) return teacherList.value
    return teacherList.value.filter(t => t.teacherName.toLowerCase().includes(q))
  })

  async function loadTeachers(courseID: number) {
    loadingTeachers.value = true
    teacherList.value = []
    selectedTeacherID.value = null
    teacherQuery.value = ''
    try {
      const res = await api.rating.getCourseTeachers(courseID)
      teacherList.value = res.data?.data ?? []
    } catch { /* silent */ }
    finally { loadingTeachers.value = false }
  }

  function selectTeacher(teacher: TeacherStats) {
    selectedTeacherID.value = teacher.teacherID
    teacherQuery.value = teacher.teacherName
    teacherDropdownOpen.value = false
  }

  function clearTeacher() {
    selectedTeacherID.value = null
    teacherQuery.value = ''
  }

  function applyDraftSelection(teacherID?: number | null, teacherName?: string) {
    if (teacherID === undefined || teacherID === null) {
      selectedTeacherID.value = null
      teacherQuery.value = teacherName ?? ''
      teacherDropdownOpen.value = false
      return
    }

    selectedTeacherID.value = teacherID
    const matched = teacherList.value.find((teacher) => teacher.teacherID === teacherID)
    teacherQuery.value = matched?.teacherName ?? teacherName ?? ''
    teacherDropdownOpen.value = false
  }

  function reset() {
    teacherList.value = []
    selectedTeacherID.value = null
    teacherQuery.value = ''
    teacherDropdownOpen.value = false
    loadingTeachers.value = false
  }

  onUnmounted(() => {
    reset()
  })

  return {
    teacherList,
    selectedTeacherID,
    teacherQuery,
    teacherDropdownOpen,
    loadingTeachers,
    filteredTeachers,
    loadTeachers,
    selectTeacher,
    clearTeacher,
    applyDraftSelection,
    reset,
  }
}
