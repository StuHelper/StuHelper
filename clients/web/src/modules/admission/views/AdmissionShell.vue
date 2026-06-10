<template>
  <main class="admission-shell">
    <section class="admission-shell__frame">
      <header class="admission-shell__header">
        <div class="admission-shell__heading">
          <p class="admission-shell__eyebrow">StuHelper JOIN</p>
          <h1 class="admission-shell__title">入群身份认证</h1>
          <p class="admission-shell__description">
            请按当前步骤完成账号绑定和学生身份认证。认证通过后，机器人会继续处理群内状态。
          </p>
        </div>

        <dl class="admission-shell__facts" aria-label="认证上下文">
          <div class="admission-shell__fact">
            <dt>目标 QQ</dt>
            <dd>{{ displayQq ? `QQ：${displayQq}` : "等待读取" }}</dd>
          </div>
          <div class="admission-shell__fact">
            <dt>当前状态</dt>
            <dd>{{ statusLabel }}</dd>
          </div>
        </dl>
      </header>

      <section class="admission-shell__panel">
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

<style scoped>
.admission-shell {
  background:
    linear-gradient(180deg, #f8fafc 0%, #eef2f7 100%);
  color: #0f172a;
  min-height: 100dvh;
  padding: 24px 16px 32px;
}

.admission-shell__frame {
  display: grid;
  gap: 16px;
  margin: 0 auto;
  max-width: 760px;
  width: 100%;
}

.admission-shell__header,
.admission-shell__panel {
  background: #ffffff;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  box-shadow: 0 12px 28px rgb(15 23 42 / 0.06);
}

.admission-shell__header {
  display: grid;
  gap: 18px;
  padding: 22px;
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

.admission-shell__eyebrow {
  color: #0f766e;
  font-size: 13px;
  font-weight: 700;
  line-height: 20px;
}

.admission-shell__title {
  color: #0f172a;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 34px;
  margin: 0;
}

.admission-shell__description {
  color: #475569;
  font-size: 14px;
  line-height: 22px;
  max-width: 58ch;
}

.admission-shell__facts {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.admission-shell__fact {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 12px;
}

.admission-shell__fact dt {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
}

.admission-shell__fact dd {
  color: #0f172a;
  font-size: 16px;
  font-weight: 700;
  line-height: 24px;
  overflow-wrap: anywhere;
}

.admission-shell__panel {
  padding: 22px;
}

.admission-shell__content {
  display: grid;
  gap: 18px;
}

@media (max-width: 640px) {
  .admission-shell {
    padding: 16px 12px 24px;
  }

  .admission-shell__header,
  .admission-shell__panel {
    padding: 16px;
  }

  .admission-shell__title {
    font-size: 24px;
    line-height: 30px;
  }

  .admission-shell__facts {
    grid-template-columns: 1fr;
  }
}
</style>
