<template>
  <div class="teacher-stats-card">
    <div class="teacher-header">
      <div class="avatar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
      </div>
      <div class="teacher-info">
        <h3 class="teacher-name">{{ stats.teacherName }}</h3>
        <p class="department">{{ stats.departmentName }}</p>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-item">
        <span class="stat-value">{{ stats.avgRating?.toFixed(1) || '-' }}</span>
        <span class="stat-label">平均评分</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ stats.courseCount }}</span>
        <span class="stat-label">授课数</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ stats.reviewCount }}</span>
        <span class="stat-label">评价数</span>
      </div>
    </div>

    <div v-if="stats.tags?.length" class="teacher-tags">
      <span v-for="tag in stats.tags" :key="tag" class="tag">{{ tag }}</span>
    </div>

    <router-link :to="`/review/teachers/${stats.teacherID}`" class="view-detail">
      查看详情
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 18l6-6-6-6"/>
      </svg>
    </router-link>
  </div>
</template>

<script setup lang="ts">
export interface TeacherStats {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating: number | null
  courseCount: number
  reviewCount: number
  tags?: string[]
}

defineProps<{
  stats: TeacherStats
}>()
</script>

<style scoped>
.teacher-stats-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: var(--space-5);
  transition: all var(--duration-base) var(--ease-out);
}

.teacher-stats-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.teacher-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.avatar {
  width: 44px;
  height: 44px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.avatar svg {
  width: 28px;
  height: 28px;
}

.teacher-info {
  flex: 1;
}

.teacher-name {
  font-family: var(--font-serif);
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--space-1) 0;
}

.department {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-3);
  padding: var(--space-4) 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.stat-value {
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--accent);
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--space-1);
}

.teacher-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-top: var(--space-4);
}

.tag {
  padding: var(--space-1) var(--space-2);
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  color: var(--text-secondary);
}

.view-detail {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  margin-top: var(--space-4);
  padding: var(--space-2);
  color: var(--accent);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
  transition: all var(--duration-fast);
}

.view-detail:hover {
  gap: var(--space-2);
}

.view-detail svg {
  width: 16px;
  height: 16px;
  transition: transform var(--duration-fast);
}

.view-detail:hover svg {
  transform: translateX(2px);
}
</style>
