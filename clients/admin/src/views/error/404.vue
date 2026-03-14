<template>
  <div class="min-h-[60vh] flex flex-col items-center justify-center">
    <h1 class="text-6xl font-bold text-gray-300">404</h1>
    <p class="mt-4 text-gray-500">页面不存在</p>
    <el-button type="primary" class="mt-6" @click="$router.push('/')">返回首页</el-button>
  </div>
</template>
