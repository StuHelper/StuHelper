<template>
  <div class="teaching-center">
    <header class="header">
      <h1>教学中心</h1>
      <p class="subtitle">StuHelper 教学服务平台</p>
    </header>

    <div class="modules">
      <div class="module-card" @click="goTo('/review')">
        <div class="icon">📝</div>
        <h2>评课社区</h2>
        <p>匿名课程评价平台，分享你的选课经验</p>
        <span class="status available">立即使用</span>
      </div>

      <div class="module-card disabled" @click="goTo('/resource')">
        <div class="icon">📚</div>
        <h2>资料共享</h2>
        <p>课程资料、往年题共享平台</p>
        <span class="status coming">开发中</span>
      </div>

      <div class="module-card disabled" @click="goTo('/spoc')">
        <div class="icon">🎬</div>
        <h2>SPOC</h2>
        <p>校内在线课程聚合平台</p>
        <span class="status coming">开发中</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goTo = (path: string) => {
  router.push(path)
}
</script>

<style scoped>
.teaching-center {
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 20px;
}

.header {
  text-align: center;
  margin-bottom: 60px;
}

.header h1 {
  font-size: 2.5rem;
  color: #303133;
  margin-bottom: 10px;
}

.subtitle {
  color: #909399;
  font-size: 1.1rem;
}

.modules {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 30px;
}

.module-card {
  background: #fff;
  border-radius: 12px;
  padding: 30px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: all 0.3s ease;
}

.module-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.module-card.disabled {
  opacity: 0.7;
}

.module-card.disabled:hover {
  transform: none;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.icon {
  font-size: 3rem;
  margin-bottom: 15px;
}

.module-card h2 {
  font-size: 1.5rem;
  color: #303133;
  margin-bottom: 10px;
}

.module-card p {
  color: #606266;
  margin-bottom: 20px;
  line-height: 1.6;
}

.status {
  display: inline-block;
  padding: 6px 16px;
  border-radius: 20px;
  font-size: 0.9rem;
}

.status.available {
  background: #e1f3d8;
  color: #67c23a;
}

.status.coming {
  background: #fdf6ec;
  color: #e6a23c;
}
</style>
