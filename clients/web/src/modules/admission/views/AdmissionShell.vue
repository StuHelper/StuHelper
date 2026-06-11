<template>
  <main class="admission-shell join-surface">
    <section class="admission-shell__frame">
      <header class="admission-shell__header join-glass-heavy">
        <div class="admission-shell__heading">
          <p class="admission-shell__eyebrow join-eyebrow">StuHelper JOIN</p>
          <h1 class="admission-shell__title">入群身份认证</h1>
          <p class="admission-shell__description">
            请按当前步骤完成账号绑定和学生身份认证。认证通过后，机器人会继续处理群内状态。
          </p>
        </div>

        <dl class="admission-shell__facts" aria-label="认证上下文">
          <div class="admission-shell__fact join-chip">
            <dt>目标 QQ</dt>
            <dd class="admission-shell__fact-qq">
              {{ displayQq ? `QQ：${displayQq}` : "等待读取" }}
            </dd>
          </div>
          <div class="admission-shell__fact join-chip">
            <dt>当前状态</dt>
            <dd>{{ statusLabel }}</dd>
          </div>
        </dl>
      </header>

      <section class="admission-shell__panel join-glass">
        <slot name="progress" />
        <div class="admission-shell__content">
          <slot />
        </div>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
defineProps<{
  displayQq: string
  statusLabel: string
}>()
</script>

<style src="./join-theme.css"></style>

<style scoped>
.admission-shell {
  min-height: 100dvh;
  padding: 28px 16px max(36px, env(safe-area-inset-bottom));
}

.admission-shell__frame {
  display: grid;
  gap: 18px;
  margin: 0 auto;
  max-width: 760px;
  width: 100%;
}

.admission-shell__header {
  display: grid;
  gap: 20px;
  padding: 26px;
}

.admission-shell__heading {
  display: grid;
  gap: 8px;
}

.admission-shell__eyebrow,
.admission-shell__description,
.admission-shell__facts,
.admission-shell__fact dt,
.admission-shell__fact dd {
  margin: 0;
}

.admission-shell__title {
  color: var(--join-ink);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 38px;
  margin: 0;
}

.admission-shell__description {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
  max-width: 58ch;
}

.admission-shell__facts {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.admission-shell__fact {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 12px 14px;
}

.admission-shell__fact dt {
  color: var(--join-ink-muted);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 18px;
}

.admission-shell__fact dd {
  color: var(--join-ink);
  font-size: 16px;
  font-weight: 700;
  line-height: 24px;
  overflow-wrap: anywhere;
}

.admission-shell__fact-qq {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.admission-shell__panel {
  padding: 26px;
}

.admission-shell__content {
  display: grid;
  gap: 18px;
}

@media (max-width: 640px) {
  .admission-shell {
    padding: 18px 12px max(28px, env(safe-area-inset-bottom));
  }

  .admission-shell__header,
  .admission-shell__panel {
    padding: 18px;
  }

  .admission-shell__title {
    font-size: 25px;
    line-height: 32px;
  }

  .admission-shell__facts {
    grid-template-columns: 1fr;
  }
}
</style>
