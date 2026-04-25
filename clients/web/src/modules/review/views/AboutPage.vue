<template>
  <div class="max-w-3xl mx-auto py-8 px-4">
    <div class="bg-bg-card rounded-2xl shadow-md p-6 md:p-8">
      <!-- Title -->
      <h1 class="text-3xl font-bold gradient-text mb-6">
        {{ t('review.about.title') }}
      </h1>

      <!-- Preface section -->
      <section class="mb-8">
        <h2 class="text-xl font-bold text-primary mb-4">
          {{ t('review.about.preface') }}
        </h2>
        <p class="mb-4 text-text-secondary leading-relaxed">
          {{ t('review.about.prefaceContent1') }}
        </p>
        <p class="mb-4 text-text-secondary leading-relaxed">
          {{ t('review.about.prefaceContent2') }}
        </p>
        <p class="mb-4 text-text-secondary leading-relaxed">
          {{ t('review.about.prefaceContent3pre') }}<a
            :href="t('review.about.siteUrl')"
            target="_blank"
            rel="noopener noreferrer"
            class="text-primary hover:text-primary-light transition-opacity"
          >{{ t('review.about.siteUrl') }}</a>{{ t('review.about.prefaceContent3post') }}
        </p>
        <p class="mb-4 text-text-secondary leading-relaxed">
          {{ t('review.about.prefaceContent4') }}
        </p>
        <p class="mb-4 text-text-secondary leading-relaxed">
          {{ t('review.about.prefaceContent5') }}
        </p>
      </section>

      <hr class="border-border my-8" />

      <!-- FAQ section -->
      <section class="mb-8">
        <h2 class="text-xl font-bold text-text-primary mb-6">
          {{ t('review.about.faq') }}
        </h2>

        <!-- FAQ 1 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(0)"
          >
            <span>{{ t('review.about.faq1q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[0] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[0]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p class="mb-4">
                {{ t('review.about.faq1a1') }}
              </p>
              <p class="mb-4">
                {{ t('review.about.faq1a2') }}
              </p>
              <p class="mb-4 pl-4">
                {{ t('review.about.faq1a3') }}
              </p>
              <p class="mb-4 pl-4">
                {{ t('review.about.faq1a4') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 2 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(1)"
          >
            <span>{{ t('review.about.faq2q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[1] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[1]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p>
                {{ t('review.about.faq2a') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 3 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(2)"
          >
            <span>{{ t('review.about.faq3q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[2] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[2]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p class="mb-4">
                {{ t('review.about.faq3a1') }}
              </p>
              <p>
                {{ t('review.about.faq3a2') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 4 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(3)"
          >
            <span>{{ t('review.about.faq4q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[3] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[3]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p>
                {{ t('review.about.faq4a') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 5 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(4)"
          >
            <span>{{ t('review.about.faq5q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[4] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[4]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p class="mb-4">
                {{ t('review.about.faq5a1') }}
              </p>
              <p>
                {{ t('review.about.faq5a2') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 6 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(5)"
          >
            <span>{{ t('review.about.faq6q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[5] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[5]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p class="mb-4">
                {{ t('review.about.faq6a1') }}
              </p>
              <p class="mb-4">
                {{ t('review.about.faq6a2') }}
              </p>
              <p class="mb-4 pl-4">
                {{ t('review.about.faq6a3') }}
              </p>
              <p class="mb-4 pl-4">
                {{ t('review.about.faq6a4') }}
              </p>
              <p>
                {{ t('review.about.faq6a5') }}
              </p>
            </div>
          </transition>
        </div>

        <!-- FAQ 7 -->
        <div class="bg-bg-elevated/50 rounded-xl overflow-hidden mb-2">
          <button
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-bg-hover font-semibold text-text-primary cursor-pointer transition-colors"
            @click="toggle(6)"
          >
            <span>{{ t('review.about.faq7q') }}</span>
            <ChevronDown
              :size="20"
              class="text-text-secondary shrink-0 transition-transform duration-300"
              :class="{ 'rotate-180': expanded[6] }"
            />
          </button>
          <transition
            enter-active-class="accordion-enter-active"
            leave-active-class="accordion-leave-active"
            @enter="onEnter"
            @after-enter="onAfterEnter"
            @leave="onLeave"
          >
            <div v-show="expanded[6]" class="px-5 py-4 text-text-secondary leading-relaxed">
              <p>
                {{ t('review.about.faq7a') }}
              </p>
            </div>
          </transition>
        </div>
      </section>

      <hr class="border-border my-8" />

      <!-- Contact section -->
      <section class="mb-8">
        <h2 class="text-xl font-bold text-text-primary mb-4">
          {{ t('review.about.contactTitle') }}
        </h2>
        <div class="flex items-center mb-4">
          <Mail :size="20" class="mr-2 text-primary" />
          <a
            :href="`mailto:${t('review.about.contactEmail')}`"
            class="text-primary hover:text-primary-light no-underline transition-opacity"
          >
            {{ t('review.about.contactEmail') }}
          </a>
        </div>
        <p class="text-text-secondary leading-relaxed">
          {{ t('review.about.contactDesc') }}
        </p>
      </section>

      <hr class="border-border my-8" />

      <!-- Disclaimer section -->
      <section>
        <h2 class="text-xl font-bold text-text-primary mb-4">
          {{ t('review.about.disclaimerTitle') }}
        </h2>
        <div class="flex flex-col gap-4">
          <p class="text-text-muted text-sm leading-relaxed">
            {{ t('review.about.disclaimer1') }}
          </p>
          <p class="text-text-muted text-sm leading-relaxed">
            {{ t('review.about.disclaimer2') }}
          </p>
          <p class="text-text-muted text-sm leading-relaxed">
            {{ t('review.about.disclaimer3') }}
          </p>
          <p class="text-text-muted text-sm leading-relaxed">
            {{ t('review.about.disclaimer4') }}
          </p>
          <p class="text-text-muted text-sm leading-relaxed">
            {{ t('review.about.disclaimer5') }}
          </p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { Mail, ChevronDown } from 'lucide-vue-next'

const { t } = useI18n()

const FAQ_COUNT = 7
const expanded = reactive<boolean[]>(Array.from({ length: FAQ_COUNT }, () => false))

function toggle(index: number): void {
  expanded[index] = !expanded[index]
}

/* ── 手风琴展开与收起过渡钩子 ── */
function onEnter(el: Element): void {
  const htmlEl = el as HTMLElement
  htmlEl.style.height = '0'
  htmlEl.style.overflow = 'hidden'
  // 强制重排，确保初始 height:0 先落到页面上
  void htmlEl.offsetHeight
  htmlEl.style.height = `${htmlEl.scrollHeight}px`
}

function onAfterEnter(el: Element): void {
  const htmlEl = el as HTMLElement
  htmlEl.style.height = ''
  htmlEl.style.overflow = ''
}

function onLeave(el: Element): void {
  const htmlEl = el as HTMLElement
  htmlEl.style.height = `${htmlEl.scrollHeight}px`
  htmlEl.style.overflow = 'hidden'
  // 强制重排，确保当前高度先落到页面上再过渡到 0
  void htmlEl.offsetHeight
  htmlEl.style.height = '0'
}
</script>

<style scoped>
/* 手风琴展开收起过渡 */
.accordion-enter-active,
.accordion-leave-active {
  transition: height 0.3s ease;
  overflow: hidden;
}
</style>
