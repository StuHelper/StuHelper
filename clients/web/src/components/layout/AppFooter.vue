<template>
  <footer class="footer">
    <div class="footer-inner">
      <div class="footer-brand">
        <span class="brand-name">StuHelper 评课社区</span>
        <span class="brand-desc">分享课程体验，帮助同学选课</span>
      </div>
      <div class="footer-links">
        <router-link to="/review/about">关于</router-link>
        <router-link to="/review/faq">FAQ</router-link>
        <a href="https://github.com/StuHelper" target="_blank">GitHub</a>
      </div>
      <div class="footer-copyright">
        © {{ currentYear }} StuHelper. All rights reserved.
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
const currentYear = new Date().getFullYear()
</script>

<style scoped>
.footer {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  padding: var(--space-12) var(--space-6);
  margin-top: auto;
}

.footer-inner {
  max-width: var(--max-width);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-6);
  text-align: center;
}

.footer-brand {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.brand-name {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
}

.brand-desc {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.footer-links {
  display: flex;
  gap: var(--space-6);
}

.footer-links a {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.footer-links a:hover {
  color: var(--accent);
}

.footer-copyright {
  color: var(--text-muted);
  font-size: var(--text-xs);
}

@media (min-width: 768px) {
  .footer-inner {
    flex-direction: row;
    justify-content: space-between;
    text-align: left;
  }
}
</style>
