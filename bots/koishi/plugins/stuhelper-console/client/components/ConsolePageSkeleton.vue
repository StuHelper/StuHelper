<template>
  <div class="sh-page-skeleton">
    <section class="sh-section">
      <div class="sh-page-skeleton__band">
        <el-skeleton-item
          v-for="index in 3"
          :key="`band-${index}`"
          variant="rect"
          class="sh-page-skeleton__band-item"
        />
      </div>
    </section>

    <div class="sh-page-skeleton__metrics">
      <el-skeleton-item
        v-for="index in 4"
        :key="`metric-${index}`"
        variant="rect"
        class="sh-page-skeleton__metric"
      />
    </div>

    <div class="sh-page-skeleton__grid">
      <section class="sh-section">
        <div class="sh-page-skeleton__card">
          <el-skeleton-item variant="h3" class="sh-page-skeleton__title" />
          <el-skeleton-item
            v-for="index in 5"
            :key="`todo-${index}`"
            variant="text"
            class="sh-page-skeleton__line"
          />
        </div>
      </section>
      <section class="sh-section">
        <div class="sh-page-skeleton__card">
          <el-skeleton-item variant="h3" class="sh-page-skeleton__title" />
          <el-skeleton-item
            v-for="index in 3"
            :key="`status-${index}`"
            variant="text"
            class="sh-page-skeleton__line"
          />
        </div>
      </section>
    </div>

    <section class="sh-section">
      <div class="sh-page-skeleton__card">
        <el-skeleton-item variant="h3" class="sh-page-skeleton__title" />
        <el-skeleton-item
          v-for="index in 4"
          :key="`shortcut-${index}`"
          variant="rect"
          class="sh-page-skeleton__shortcut"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.sh-page-skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-3);
}

.sh-page-skeleton__band,
.sh-page-skeleton__metrics,
.sh-page-skeleton__grid {
  display: grid;
  gap: var(--sh-s-3);
}

.sh-page-skeleton__band {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding: var(--sh-s-3) 14px;
}

.sh-page-skeleton__metrics {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.sh-page-skeleton__grid {
  grid-template-columns: minmax(0, 7fr) minmax(300px, 4fr);
}

.sh-page-skeleton__band-item,
.sh-page-skeleton__metric,
.sh-page-skeleton__shortcut {
  height: 72px;
  border-radius: var(--sh-r-2);
}

.sh-page-skeleton__metric {
  height: 112px;
}

.sh-page-skeleton__card {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-3);
  padding: var(--sh-s-4) 14px;
}

.sh-page-skeleton__title {
  width: 140px;
}

.sh-page-skeleton__line {
  height: 14px;
}

.sh-page-skeleton__shortcut {
  height: 60px;
}

@media (max-width: 1080px) {
  .sh-page-skeleton__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 720px) {
  .sh-page-skeleton__band,
  .sh-page-skeleton__metrics {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
