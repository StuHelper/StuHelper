<script setup lang="ts">
import { useRouter } from "vue-router";
import HeroSection from "@/components/business/HeroSection.vue";
import FeatureCard from "@/components/business/FeatureCard.vue";
import CountUp from "@/components/animated/CountUp.vue";
import {
    AcademicCapIcon,
    UserGroupIcon,
    WrenchScrewdriverIcon,
    ChatBubbleLeftRightIcon,
} from "@heroicons/vue/24/outline";

const router = useRouter();

const features = [
    {
        icon: AcademicCapIcon,
        title: "评课中心",
        description: "查看真实的课程评价，帮助你选到心仪的课程",
        color: "#06b6d4",
        stats: "10000+ 条评价",
    },
    {
        icon: UserGroupIcon,
        title: "教师主页",
        description: "了解教师的教学风格和学生评价",
        color: "#4f46e5",
        stats: "500+ 位教师",
    },
    {
        icon: WrenchScrewdriverIcon,
        title: "工具箱",
        description: "实用的校园工具，让学习生活更便捷",
        color: "#0891b2",
        stats: "即将上线",
    },
    {
        icon: ChatBubbleLeftRightIcon,
        title: "社群",
        description: "与同学交流分享，共同成长进步",
        color: "#6366f1",
        stats: "即将上线",
    },
];

const onExplore = () => {
    router.push("/course");
};
</script>

<template>
    <div class="home-page">
        <HeroSection
            title="StuHelper 评课社区"
            subtitle="真实的课程评价，帮助你做出更好的选择"
            @explore="onExplore"
        />

        <section class="features-section">
            <div class="features-grid">
                <FeatureCard
                    v-for="feature in features"
                    :key="feature.title"
                    :icon="feature.icon"
                    :title="feature.title"
                    :description="feature.description"
                    :color="feature.color"
                    :stats="feature.stats"
                />
            </div>
        </section>

        <section class="stats-section">
            <div class="stats-container">
                <div class="stat-item">
                    <CountUp
                        :target="1200"
                        :duration="2000"
                        class="stat-number"
                    />
                    <p class="stat-label">课程数</p>
                </div>
                <div class="stat-item">
                    <CountUp
                        :target="10000"
                        :duration="2000"
                        class="stat-number"
                    />
                    <p class="stat-label">评价数</p>
                </div>
                <div class="stat-item">
                    <CountUp
                        :target="5000"
                        :duration="2000"
                        class="stat-number"
                    />
                    <p class="stat-label">用户数</p>
                </div>
            </div>
        </section>

        <footer class="footer">
            <div class="footer-content">
                <p>&copy; 2026 StuHelper. All rights reserved.</p>
                <div class="footer-links">
                    <RouterLink to="/about">关于我们</RouterLink>
                    <RouterLink to="/privacy">隐私政策</RouterLink>
                    <RouterLink to="/terms">服务条款</RouterLink>
                </div>
            </div>
        </footer>
    </div>
</template>

<style scoped>
.home-page {
    min-height: 100vh;
    background: linear-gradient(to bottom, #f9fafb, #ffffff);
}

.dark .home-page {
    background: linear-gradient(to bottom, #0f172a, #1e293b);
}

.features-section {
    padding: 4rem 2rem;
    max-width: 1200px;
    margin: 0 auto;
}

.features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 2rem;
}

.stats-section {
    padding: 4rem 2rem;
    background: rgba(255, 255, 255, 0.5);
    backdrop-filter: blur(12px);
}

.dark .stats-section {
    background: rgba(30, 30, 30, 0.5);
}

.stats-container {
    max-width: 1200px;
    margin: 0 auto;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 3rem;
    text-align: center;
}

.stat-item {
    padding: 2rem;
}

.stat-number {
    font-size: 3rem;
    font-weight: 800;
    background: linear-gradient(135deg, #06b6d4, #4f46e5);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

.stat-label {
    margin-top: 0.5rem;
    font-size: 1rem;
    color: #666;
}

.dark .stat-label {
    color: #a0a0a0;
}

.footer {
    padding: 2rem;
    background: rgba(255, 255, 255, 0.8);
    backdrop-filter: blur(12px);
    border-top: 1px solid rgba(0, 0, 0, 0.1);
}

.dark .footer {
    background: rgba(30, 30, 30, 0.8);
    border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.footer-content {
    max-width: 1200px;
    margin: 0 auto;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
    color: #666;
    font-size: 0.875rem;
}

.dark .footer-content {
    color: #a0a0a0;
}

.footer-links {
    display: flex;
    gap: 2rem;
}

.footer-links a {
    color: inherit;
    text-decoration: none;
    transition: color 0.2s;
}

.footer-links a:hover {
    color: #06b6d4;
}

@media (max-width: 768px) {
    .features-grid {
        grid-template-columns: 1fr;
    }

    .stats-container {
        grid-template-columns: 1fr;
        gap: 2rem;
    }

    .footer-content {
        flex-direction: column;
        text-align: center;
    }
}
</style>
