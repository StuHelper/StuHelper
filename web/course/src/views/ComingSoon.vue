<template>
  <div class="coming-soon">
    <div class="content">
      <div class="icon">🚧</div>
      <h1>功能开发中</h1>
      <p>该功能正在紧锣密鼓地开发中，敬请期待！</p>
      <el-button type="primary" @click="goBack">返回首页</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goBack = () => {
  router.push('/')
}
</script>

<style scoped>
.coming-soon {
  min-height: 80vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.content {
  text-align: center;
}

.icon {
  font-size: 5rem;
  margin-bottom: 20px;
}

h1 {
  font-size: 2rem;
  color: #303133;
  margin-bottom: 15px;
}

p {
  color: #909399;
  margin-bottom: 30px;
  font-size: 1.1rem;
}
</style>
