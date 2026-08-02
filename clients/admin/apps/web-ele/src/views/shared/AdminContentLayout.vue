<script setup lang="ts">
defineProps<{
  description?: string;
  title: string;
  total?: number;
}>();
</script>

<template>
  <section class="admin-content-page">
    <header class="admin-content-page__header">
      <div class="admin-content-page__heading">
        <div class="admin-content-page__title-row">
          <h1>{{ title }}</h1>
          <span
            v-if="typeof total === 'number'"
            class="admin-content-page__total"
          >
            {{ total }}
          </span>
        </div>
        <p v-if="description" class="admin-content-page__description">
          {{ description }}
        </p>
      </div>
      <div v-if="$slots.actions" class="admin-content-page__actions">
        <slot name="actions"></slot>
      </div>
    </header>

    <div v-if="$slots.toolbar" class="admin-content-page__toolbar">
      <slot name="toolbar"></slot>
    </div>

    <div class="admin-content-page__body">
      <slot></slot>
    </div>

    <footer v-if="$slots.pagination" class="admin-content-page__footer">
      <slot name="pagination"></slot>
    </footer>
  </section>
</template>

<style>
.admin-content-page {
  min-height: calc(100dvh - 104px);
  padding: 18px;
  color: var(--el-text-color-primary);
}

.admin-content-page__header,
.admin-content-page__toolbar,
.admin-content-page__body,
.admin-content-page__footer {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
}

.admin-content-page__header {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px;
  border-radius: 8px 8px 0 0;
}

.admin-content-page__heading {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
  min-width: 0;
}

.admin-content-page__title-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.admin-content-page__heading h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 650;
  line-height: 1.3;
}

.admin-content-page__description {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.admin-content-page__total {
  min-width: 28px;
  padding: 2px 8px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 20px;
  color: var(--el-text-color-secondary);
  text-align: center;
  background: var(--el-fill-color-light);
  border-radius: 999px;
}

.admin-content-page__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.admin-content-page__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: end;
  padding: 14px 20px;
  border-top: 0;
}

.admin-content-page__body {
  overflow: hidden;
  border-top: 0;
}

.admin-content-page__footer {
  display: flex;
  justify-content: flex-end;
  padding: 14px 20px;
  border-top: 0;
  border-radius: 0 0 8px 8px;
}

.admin-embedded-pagination {
  display: flex;
  justify-content: flex-end;
  padding: 14px 20px;
  border-top: 1px solid var(--el-border-color);
}

.admin-toolbar-control {
  width: 160px;
}

.admin-toolbar-control--wide {
  width: 200px;
}

.admin-load-error {
  margin: 12px 20px 0;
}

@media (max-width: 768px) {
  .admin-content-page {
    padding: 12px;
  }

  .admin-content-page__header,
  .admin-content-page__toolbar,
  .admin-content-page__footer {
    padding-inline: 14px;
  }

  .admin-toolbar-control {
    width: 100%;
  }

  .admin-toolbar-control--wide {
    width: 100%;
  }

  .admin-content-page__body {
    overflow: hidden;
  }
}
</style>
