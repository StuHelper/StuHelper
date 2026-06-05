<template>
  <div class="settings-view">
    <!-- Header -->
    <header class="view-header">
      <h2 class="view-title">全局设置</h2>
      <div class="header-actions">
        <button
          class="action-btn danger-outline"
          :disabled="!settingsLoaded || saving"
          @click="resetToDefault"
        >
          <k-icon name="rotate-ccw" class="btn-icon" />
          <span>恢复默认</span>
        </button>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <k-icon name="loader" class="spin" />
      <span class="loading-text">Loading...</span>
    </div>

    <div v-else-if="loadError" class="settings-load-error" role="alert">
      <div class="settings-load-error__body">
        <strong>加载全局设置失败</strong>
        <span>{{ loadError }}</span>
      </div>
      <button class="action-btn" @click="loadSettings">
        <k-icon name="refresh-cw" class="btn-icon" />
        <span>重试</span>
      </button>
    </div>

    <!-- Settings Content -->
    <div v-else-if="settingsLoaded" class="settings-content">
      <!-- Mobile Section Selector -->
      <div class="mobile-section-selector">
        <div class="selector-current" @click="sectionDropdownOpen = !sectionDropdownOpen">
          <span class="current-label">{{ currentSectionLabel }}</span>
          <k-icon :name="sectionDropdownOpen ? 'chevron-up' : 'chevron-down'" class="selector-arrow" />
        </div>
        <transition name="dropdown">
          <div class="selector-dropdown" v-if="sectionDropdownOpen">
            <div
              v-for="section in sections"
              :key="section.id"
              class="selector-option"
              :class="{ active: activeSection === section.id }"
              @click="selectSection(section.id)"
            >
              {{ section.label }}
            </div>
          </div>
        </transition>
      </div>
      <div class="mobile-section-backdrop" v-if="sectionDropdownOpen" @click="sectionDropdownOpen = false"></div>

      <!-- Sidebar (PC) -->
      <nav class="settings-sidebar">
        <div
          v-for="section in sections"
          :key="section.id"
          class="sidebar-item"
          :class="{ active: activeSection === section.id }"
          @click="activeSection = section.id"
        >
          <k-icon :name="section.icon" class="sidebar-icon" />
          <span class="sidebar-label">{{ section.label }}</span>
        </div>
      </nav>

      <!-- Main Content -->
      <div class="settings-main">
        <div v-if="actionError" class="settings-action-error" role="alert">
          <div class="settings-action-error__body">
            <strong>{{ actionErrorTitle }}</strong>
            <span>{{ actionError }}</span>
          </div>
          <button class="action-btn" type="button" @click="clearActionError">
            关闭
          </button>
        </div>

        <!-- Warn Settings -->
        <div v-show="activeSection === 'warn'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">警告设置</h3>
            <p class="section-desc">配置警告次数限制和自动禁言规则</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">警告次数限制</label>
              <div class="form-control">
                <el-input-number v-model="settings.warnLimit" :min="0" :max="99" size="small" />
                <span class="form-hint">达到此次数后触发自动禁言（设为0表示每次警告都立即禁言）</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">禁言时长表达式</label>
              <div class="form-control">
                <el-input v-model="settings.banTimes.expression" placeholder="{t}^2h" size="small" />
                <span class="form-hint">{t} 代表警告次数，如 {t}^2h 表示警告次数的平方小时</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Forbidden Keywords -->
        <div v-show="activeSection === 'forbidden'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">禁言关键词</h3>
            <p class="section-desc">配置全局禁言关键词和自动处理规则</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">自动撤回</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.forbidden.autoDelete" />
                  <span class="toggle-track"></span>
                </label>
                <span class="form-hint">自动撤回包含禁言关键词的消息</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">自动禁言</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.forbidden.autoBan" />
                  <span class="toggle-track"></span>
                </label>
                <span class="form-hint">自动禁言发送禁言关键词的用户</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">自动踢出</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.forbidden.autoKick" />
                  <span class="toggle-track"></span>
                </label>
                <span class="form-hint">自动踢出发送禁言关键词的用户</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">禁言时长 (ms)</label>
              <div class="form-control">
                <el-input-number v-model="settings.forbidden.muteDuration" :min="0" :step="60000" size="small" />
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">禁言关键词</label>
              <div class="form-control">
                <textarea
                  v-model="forbiddenKeywordsText"
                  rows="4"
                  placeholder="每行一个关键词"
                  class="form-textarea"
                ></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- Keywords -->
        <div v-show="activeSection === 'keywords'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">入群审核</h3>
            <p class="section-desc">配置入群审核关键词</p>
          </div>
          <div class="form-grid">
            <div class="form-row full">
              <label class="form-label">审核关键词</label>
              <div class="form-control">
                <textarea
                  v-model="keywordsText"
                  rows="6"
                  placeholder="每行一个关键词"
                  class="form-textarea"
                ></textarea>
                <span class="form-hint">入群申请需要包含这些关键词才能通过审核</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Dice -->
        <div v-show="activeSection === 'dice'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">掷骰子</h3>
            <p class="section-desc">配置掷骰子功能</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.dice.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">结果长度限制</label>
              <div class="form-control">
                <el-input-number v-model="settings.dice.lengthLimit" :min="100" :max="10000" size="small" />
                <span class="form-hint">超过此长度的结果将无法显示</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Banme -->
        <div v-show="activeSection === 'banme'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">Banme 自助禁言</h3>
            <p class="section-desc">配置自助禁言功能和金卡系统</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.banme.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">最小时长 (秒)</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.baseMin" :min="1" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">最大时长 (分钟)</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.baseMax" :min="1" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">递增系数</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.growthRate" :min="0" size="small" />
                <span class="form-hint">越大增长越快</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">自动禁言匹配</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.banme.autoBan" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
          </div>

          <div class="subsection-divider">金卡系统</div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用金卡</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.banme.jackpot.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">基础概率</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.jackpot.baseProb" :min="0" :max="1" :step="0.001" :precision="4" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">软保底 (抽)</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.jackpot.softPity" :min="0" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">硬保底 (抽)</label>
              <div class="form-control">
                <el-input-number v-model="settings.banme.jackpot.hardPity" :min="0" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">UP 奖励时长</label>
              <div class="form-control">
                <el-input v-model="settings.banme.jackpot.upDuration" placeholder="24h" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">歪奖励时长</label>
              <div class="form-control">
                <el-input v-model="settings.banme.jackpot.loseDuration" placeholder="12h" size="small" />
              </div>
            </div>
          </div>
        </div>

        <!-- Friend Request -->
        <div v-show="activeSection === 'friendRequest'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">好友申请</h3>
            <p class="section-desc">配置好友申请验证</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用验证</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.friendRequest.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">拒绝消息</label>
              <div class="form-control">
                <el-input v-model="settings.friendRequest.rejectMessage" placeholder="请输入正确的验证信息" size="small" />
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">通过关键词</label>
              <div class="form-control">
                <textarea
                  v-model="friendKeywordsText"
                  rows="4"
                  placeholder="每行一个关键词"
                  class="form-textarea"
                ></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- Guild Request -->
        <div v-show="activeSection === 'guildRequest'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">入群邀请</h3>
            <p class="section-desc">配置入群邀请处理</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">自动同意</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.guildRequest.enabled" />
                  <span class="toggle-track"></span>
                </label>
                <span class="form-hint">启用时同意所有邀请，禁用时拒绝所有</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">拒绝消息</label>
              <div class="form-control">
                <el-input v-model="settings.guildRequest.rejectMessage" placeholder="暂不接受入群邀请" size="small" />
              </div>
            </div>
          </div>
        </div>

        <!-- Title -->
        <div v-show="activeSection === 'title'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">头衔设置</h3>
            <p class="section-desc">配置头衔功能</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.setTitle.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">所需权限等级</label>
              <div class="form-control">
                <el-input-number v-model="settings.setTitle.authority" :min="1" :max="5" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">头衔最大长度</label>
              <div class="form-control">
                <el-input-number v-model="settings.setTitle.maxLength" :min="1" :max="50" size="small" />
              </div>
            </div>
          </div>
        </div>

        <!-- Anti Repeat -->
        <div v-show="activeSection === 'antiRepeat'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">反复读</h3>
            <p class="section-desc">配置反复读检测功能</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.antiRepeat.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">触发阈值</label>
              <div class="form-control">
                <el-input-number v-model="settings.antiRepeat.threshold" :min="2" :max="20" size="small" />
                <span class="form-hint">超过该次数将撤回除第一条外的所有复读消息</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Anti Recall -->
        <div v-show="activeSection === 'antiRecall'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">防撤回</h3>
            <p class="section-desc">配置防撤回功能</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.antiRecall.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">保存天数</label>
              <div class="form-control">
                <el-input-number v-model="settings.antiRecall.retentionDays" :min="1" :max="30" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">每用户最大记录数</label>
              <div class="form-control">
                <el-input-number v-model="settings.antiRecall.maxRecordsPerUser" :min="10" :max="200" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">显示原消息时间</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.antiRecall.showOriginalTime" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
          </div>
        </div>

        <!-- OpenAI -->
        <div v-show="activeSection === 'openai'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">AI 功能</h3>
            <p class="section-desc">配置 OpenAI 兼容 API</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.openai.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">启用对话</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.openai.chatEnabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">启用翻译</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.openai.translateEnabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">API 密钥</label>
              <div class="form-control">
                <el-input v-model="settings.openai.apiKey" type="password" show-password placeholder="sk-..." size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">API 地址</label>
              <div class="form-control">
                <el-input v-model="settings.openai.apiUrl" placeholder="https://api.openai.com/v1" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">模型名称</label>
              <div class="form-control">
                <el-input v-model="settings.openai.model" placeholder="gpt-3.5-turbo" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">最大 Token 数</label>
              <div class="form-control">
                <el-input-number v-model="settings.openai.maxTokens" :min="256" :max="32768" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">温度</label>
              <div class="form-control">
                <el-input-number v-model="settings.openai.temperature" :min="0" :max="2" :step="0.1" :precision="1" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">上下文消息数</label>
              <div class="form-control">
                <el-input-number v-model="settings.openai.contextLimit" :min="1" :max="50" size="small" />
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">系统提示词</label>
              <div class="form-control">
                <textarea
                  v-model="settings.openai.systemPrompt"
                  rows="4"
                  class="form-textarea"
                  placeholder="你是一个有帮助的AI助手..."
                ></textarea>
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">翻译提示词</label>
              <div class="form-control">
                <textarea
                  v-model="settings.openai.translatePrompt"
                  rows="4"
                  class="form-textarea"
                  placeholder="翻译提示词..."
                ></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- Report -->
        <div v-show="activeSection === 'report'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">举报功能</h3>
            <p class="section-desc">配置 AI 辅助举报审核</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用功能</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.report.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">所需权限等级</label>
              <div class="form-control">
                <el-input-number v-model="settings.report.authority" :min="1" :max="5" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">自动处理</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.report.autoProcess" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">举报时间限制 (分)</label>
              <div class="form-control">
                <el-input-number v-model="settings.report.maxReportTime" :min="0" size="small" />
                <span class="form-hint">0 表示不限制</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">冷却时间 (分)</label>
              <div class="form-control">
                <el-input-number v-model="settings.report.maxReportCooldown" :min="0" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">免冷却权限等级</label>
              <div class="form-control">
                <el-input-number v-model="settings.report.minAuthorityNoLimit" :min="1" :max="5" size="small" />
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">默认审核提示词</label>
              <div class="form-control">
                <textarea
                  v-model="settings.report.defaultPrompt"
                  rows="6"
                  class="form-textarea"
                  placeholder="AI 审核提示词..."
                ></textarea>
              </div>
            </div>
            <div class="form-row full">
              <label class="form-label">带上下文审核提示词</label>
              <div class="form-control">
                <textarea
                  v-model="settings.report.contextPrompt"
                  rows="6"
                  class="form-textarea"
                  placeholder="带上下文的 AI 审核提示词..."
                ></textarea>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
    
    <!-- Save Bar -->
    <transition name="slide-up">
      <div class="save-bar" v-if="settingsLoaded && hasChanges">
        <span class="save-bar-text">检测到未保存的修改</span>
        <div class="save-actions">
          <button class="save-bar-btn secondary" :disabled="saving" @click="resetChanges">放弃更改</button>
          <button class="save-bar-btn primary" :disabled="saving" @click="saveSettings">
            {{ saving ? '保存中...' : '保存更改' }}
          </button>
        </div>
      </div>
    </transition>

    <ConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :tone="confirmDialog.tone"
      :confirm-text="confirmDialog.confirmText"
      :cancel-text="confirmDialog.cancelText"
      @confirm="acceptConfirm"
      @cancel="cancelConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { message } from '@koishijs/client'
import { settingsApi } from '../api'
import { useActionError } from '../composables/use-action-error'
import { useConfirm } from '../composables/use-confirm'
import ConfirmDialog from './primitives/ConfirmDialog.vue'

type PlainRecord = Record<string, unknown>

interface SettingsModel extends PlainRecord {
  keywords: string[]
  warnLimit: number
  banTimes: {
    expression: string
  }
  forbidden: {
    autoDelete: boolean
    autoBan: boolean
    autoKick: boolean
    muteDuration: number
    keywords: string[]
  }
  dice: {
    enabled: boolean
    lengthLimit: number
  }
  banme: {
    enabled: boolean
    baseMin: number
    baseMax: number
    growthRate: number
    autoBan: boolean
    jackpot: {
      enabled: boolean
      baseProb: number
      softPity: number
      hardPity: number
      upDuration: string
      loseDuration: string
    }
  }
  friendRequest: {
    enabled: boolean
    keywords: string[]
    rejectMessage: string
  }
  guildRequest: {
    enabled: boolean
    rejectMessage: string
  }
  setTitle: {
    enabled: boolean
    authority: number
    maxLength: number
  }
  antiRepeat: {
    enabled: boolean
    threshold: number
  }
  openai: {
    enabled: boolean
    chatEnabled: boolean
    translateEnabled: boolean
    apiKey: string
    apiUrl: string
    model: string
    systemPrompt: string
    translatePrompt: string
    maxTokens: number
    temperature: number
    contextLimit: number
  }
  report: {
    enabled: boolean
    authority: number
    autoProcess: boolean
    defaultPrompt: string
    contextPrompt: string
    maxReportTime: number
    maxReportCooldown: number
    minAuthorityNoLimit: number
    guildConfigs: PlainRecord
  }
  antiRecall: {
    enabled: boolean
    retentionDays: number
    maxRecordsPerUser: number
    showOriginalTime: boolean
  }
}

// 默认配置结构
const defaultSettings: SettingsModel = {
  keywords: [],
  warnLimit: 3,
  banTimes: { expression: '{t}^2h' },
  forbidden: {
    autoDelete: false,
    autoBan: false,
    autoKick: false,
    muteDuration: 600000,
    keywords: []
  },
  dice: { enabled: true, lengthLimit: 1000 },
  banme: {
    enabled: true,
    baseMin: 1,
    baseMax: 30,
    growthRate: 30,
    autoBan: false,
    jackpot: {
      enabled: true,
      baseProb: 0.006,
      softPity: 73,
      hardPity: 89,
      upDuration: '24h',
      loseDuration: '12h'
    }
  },
  friendRequest: {
    enabled: false,
    keywords: [],
    rejectMessage: '请输入正确的验证信息'
  },
  guildRequest: {
    enabled: false,
    rejectMessage: '暂不接受入群邀请'
  },
  setTitle: { enabled: true, authority: 3, maxLength: 18 },
  antiRepeat: { enabled: false, threshold: 3 },
  openai: {
    enabled: false,
    chatEnabled: true,
    translateEnabled: true,
    apiKey: '',
    apiUrl: 'https://api.openai.com/v1',
    model: 'gpt-3.5-turbo',
    systemPrompt: '你是一个有帮助的AI助手，请简短、准确地回答问题。',
    translatePrompt: '',
    maxTokens: 2048,
    temperature: 0.7,
    contextLimit: 10
  },
  report: {
    enabled: true,
    authority: 1,
    autoProcess: true,
    defaultPrompt: '',
    contextPrompt: '',
    maxReportTime: 30,
    maxReportCooldown: 60,
    minAuthorityNoLimit: 2,
    guildConfigs: {}
  },
  antiRecall: {
    enabled: false,
    retentionDays: 7,
    maxRecordsPerUser: 50,
    showOriginalTime: true
  }
}

const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const {
  actionError,
  actionErrorTitle,
  setActionError,
  clearActionError,
  errorMessage,
} = useActionError({
  onError: (_title, details) => message.error(details),
})
const settings = ref<SettingsModel>(cloneDefaultSettings())
const originalSettings = ref<string>('') // 原始设置的 JSON 字符串用于比较
const activeSection = ref('warn')
const sectionDropdownOpen = ref(false)
const settingsLoaded = computed(() => Boolean(originalSettings.value) && !loadError.value)
let loadRequestSeq = 0

// 当前选中的 section 标签
const currentSectionLabel = computed(() => {
  const section = sections.find(s => s.id === activeSection.value)
  return section ? section.label : '警告设置'
})

// 选择 section（移动端下拉使用）
const selectSection = (id: string) => {
  activeSection.value = id
  sectionDropdownOpen.value = false
}

const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

// 检测是否有未保存的修改
const hasChanges = computed(() => {
  if (!originalSettings.value) return false
  return JSON.stringify(settings.value) !== originalSettings.value
})

// 将数组转换为文本
const keywordsText = computed({
  get: () => (settings.value.keywords || []).join('\n'),
  set: (val) => { settings.value.keywords = val.split('\n').map((s: string) => s.trim()).filter((s: string) => s) }
})

const forbiddenKeywordsText = computed({
  get: () => settings.value.forbidden.keywords.join('\n'),
  set: (val: string) => {
    settings.value.forbidden.keywords = splitLines(val)
  }
})

const friendKeywordsText = computed({
  get: () => settings.value.friendRequest.keywords.join('\n'),
  set: (val: string) => {
    settings.value.friendRequest.keywords = splitLines(val)
  }
})

const sections = [
  { id: 'warn', label: '警告设置', icon: 'stuhelperGroupCenter:octicons.warning' },
  { id: 'forbidden', label: '禁言关键词', icon: 'stuhelperGroupCenter:octicons.x' },
  { id: 'keywords', label: '入群审核', icon: 'stuhelperGroupCenter:octicons.personadd' },
  { id: 'dice', label: '掷骰子', icon: 'stuhelperGroupCenter:octicons.tools' },
  { id: 'banme', label: '自我禁言', icon: 'stuhelperGroupCenter:octicons.person' },
  { id: 'friendRequest', label: '好友申请', icon: 'stuhelperGroupCenter:octicons.personadd' },
  { id: 'guildRequest', label: '入群邀请', icon: 'stuhelperGroupCenter:octicons.people' },
  { id: 'title', label: '头衔设置', icon: 'stuhelperGroupCenter:octicons.gear' },
  { id: 'antiRepeat', label: '反复读', icon: 'stuhelperGroupCenter:octicons.discussion' },
  { id: 'antiRecall', label: '防撤回', icon: 'stuhelperGroupCenter:octicons.log' },
  { id: 'openai', label: 'AI功能', icon: 'stuhelperGroupCenter:octicons.graph' },
  { id: 'report', label: '举报功能', icon: 'stuhelperGroupCenter:octicons.warning' },
]

function splitLines(value: string): string[] {
  return value.split('\n').map((line) => line.trim()).filter((line) => line)
}

function cloneDefaultSettings(): SettingsModel {
  return structuredClone(defaultSettings)
}

function deepMerge<T extends PlainRecord>(target: T, source: unknown): T {
  if (!isPlainRecord(source)) {
    return target
  }

  const targetRecord = target as PlainRecord
  for (const [key, value] of Object.entries(source)) {
    const current = targetRecord[key]
    if (isPlainRecord(current) && isPlainRecord(value)) {
      deepMerge(current, value)
    } else {
      targetRecord[key] = value
    }
  }

  return target
}

function isPlainRecord(value: unknown): value is PlainRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function parseSettingsSnapshot(value: string): SettingsModel {
  const parsed: unknown = JSON.parse(value)
  return deepMerge(cloneDefaultSettings(), parsed)
}

const loadSettings = async (): Promise<boolean> => {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  loadError.value = ''
  clearActionError()
  try {
    const data = await settingsApi.get()
    if (requestSeq !== loadRequestSeq) return false
    // 深度合并默认值和返回数据
    settings.value = deepMerge(cloneDefaultSettings(), data)
    // 保存原始设置用于比较
    originalSettings.value = JSON.stringify(settings.value)
    return true
  } catch (cause) {
    if (requestSeq !== loadRequestSeq) return false
    const details = errorMessage(cause, '加载设置失败')
    loadError.value = details
    originalSettings.value = ''
    message.error(details)
    return false
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

const saveSettings = async () => {
  if (saving.value) return
  if (!settingsLoaded.value) {
    setActionError('保存失败', '设置尚未加载，不能保存默认表单', '保存失败')
    return
  }

  saving.value = true
  clearActionError()
  try {
    await settingsApi.update(settings.value)
    // 更新原始设置
    originalSettings.value = JSON.stringify(settings.value)
    message.success('设置已保存')
  } catch (cause) {
    setActionError('保存失败', cause, '保存设置失败')
  } finally {
    saving.value = false
  }
}

const resetChanges = async () => {
  if (!settingsLoaded.value) return

  const confirmed = await confirm({
    title: '放弃更改',
    message: '确定要放弃当前所有未保存的修改吗？',
    tone: 'normal'
  })
  
  if (confirmed) {
    clearActionError()
    // 从原始设置恢复
    settings.value = parseSettingsSnapshot(originalSettings.value)
    message.success('已放弃更改')
  }
}

const resetToDefault = async () => {
  if (!settingsLoaded.value) return

  const confirmed = await confirm({
    title: '恢复默认设置',
    message: '确定要将所有设置恢复为默认值吗？此操作将覆盖当前所有设置，需要保存后才会生效。',
    tone: 'danger'
  })
  
  if (confirmed) {
    clearActionError()
    // 恢复为默认设置
    settings.value = cloneDefaultSettings()
    message.success('已恢复默认设置，请保存以应用更改')
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
/* ========================================
   GitHub Dimmed / Vercel Style
   ======================================== */

.settings-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', sans-serif;
}

/* Header */
.view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--k-color-divider);
}

.view-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--fg1);
  margin: 0;
  letter-spacing: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Action Buttons */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  color: var(--fg2);
  background: var(--bg2);
  border: 1px solid var(--k-color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover {
  color: var(--fg1);
  background: var(--bg3);
  border-color: var(--fg3);
}

.action-btn:active {
  background: var(--bg1);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn:disabled:hover {
  color: var(--fg2);
  background: var(--bg2);
  border-color: var(--k-color-border);
}

.action-btn.primary {
  color: var(--fg0);
  background: var(--k-color-primary);
  border-color: var(--k-color-primary);
}

.action-btn.primary:hover {
  background: var(--k-color-primary-shade);
  border-color: var(--k-color-primary-shade);
}

.action-btn.danger {
  color: var(--fg0);
  background: var(--k-color-danger);
  border-color: var(--k-color-danger);
}

.action-btn.danger:hover {
  background: var(--k-color-danger-shade);
  border-color: var(--k-color-danger-shade);
}

.action-btn.danger-outline {
  color: var(--k-color-danger);
  background: transparent;
  border-color: var(--k-color-border);
}

.action-btn.danger-outline:hover {
  background: var(--k-color-danger-fade);
  border-color: var(--k-color-danger);
}

.btn-icon {
  font-size: 13px;
  opacity: 0.8;
}

.action-btn:hover .btn-icon {
  opacity: 1;
}

/* Loading State */
.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--fg3);
}

.loading-text {
  font-size: 12px;
  font-family: 'SF Mono', 'Consolas', monospace;
}

.settings-load-error,
.settings-action-error {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  color: var(--k-color-danger);
  background: var(--k-color-danger-fade);
  border: 1px solid color-mix(in srgb, var(--k-color-danger) 34%, transparent);
  border-radius: 6px;
}

.settings-action-error {
  margin-bottom: 16px;
}

.settings-load-error__body,
.settings-action-error__body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  font-size: 13px;
}

.settings-load-error__body strong,
.settings-action-error__body strong {
  font-size: 14px;
  color: var(--fg1);
}

.settings-load-error__body span,
.settings-action-error__body span {
  overflow-wrap: anywhere;
}

/* Settings Content */
.settings-content {
  flex: 1;
  display: flex;
  background: var(--bg2);
  border: 1px solid var(--k-color-border);
  border-radius: 6px;
  overflow: hidden;
}

/* Sidebar */
.settings-sidebar {
  width: 160px;
  border-right: 1px solid var(--k-color-divider);
  padding: 8px;
  overflow-y: auto;
  background: var(--bg1);
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 4px;
  cursor: pointer;
  color: var(--fg2);
  font-size: 12px;
  transition: all 0.15s ease;
  margin-bottom: 2px;
}

.sidebar-item:hover {
  background: var(--bg3);
  color: var(--fg1);
}

.sidebar-item.active {
  background: var(--k-color-primary-fade);
  color: var(--k-color-primary-tint);
}

.sidebar-icon {
  font-size: 14px;
  opacity: 0.7;
}

.sidebar-item.active .sidebar-icon {
  opacity: 1;
}

.sidebar-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Main Content */
.settings-main {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.config-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-header {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--k-color-divider);
}

.section-title {
  margin: 0 0 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--fg1);
}

.section-desc {
  margin: 0;
  font-size: 11px;
  color: var(--fg3);
}

/* Form Layout */
.form-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.form-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.form-row.full {
  flex-direction: column;
  gap: 6px;
}

.form-label {
  width: 130px;
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--fg2);
  padding-top: 6px;
}

.form-row.full .form-label {
  width: auto;
  padding-top: 0;
}

.form-control {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.form-row.full .form-control {
  width: 100%;
}

.form-hint {
  font-size: 11px;
  color: var(--fg3);
}

.form-textarea {
  width: 100%;
  max-width: 500px;
  padding: 8px 10px;
  border: 1px solid var(--k-color-border);
  border-radius: 6px;
  background: var(--bg0);
  color: var(--fg1);
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 12px;
  resize: vertical;
  transition: border-color 0.15s ease;
}

.form-textarea::placeholder {
  color: var(--fg3);
}

.form-textarea:focus {
  outline: none;
  border-color: var(--k-color-primary);
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-track {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--k-color-border);
  transition: 0.2s;
  border-radius: 20px;
}

.toggle-track:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: var(--fg0);
  transition: 0.2s;
  border-radius: 50%;
}

.toggle-switch input:checked + .toggle-track {
  background-color: var(--k-color-primary);
}

.toggle-switch input:checked + .toggle-track:before {
  transform: translateX(16px);
}

/* Subsection Divider */
.subsection-divider {
  font-size: 11px;
  font-weight: 500;
  color: var(--fg3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin: 12px 0 8px;
  padding-bottom: 8px;
  border-bottom: 1px dashed var(--k-color-divider);
}



/* Save Bar - Discord 风格 */
.save-bar {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  width: calc(100% - 48px);
  max-width: 560px;
  background: #111214;
  padding: 10px 12px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.4);
  z-index: 100;
}

.save-bar-text {
  font-size: 0.8125rem;
  color: #b5bac1;
  font-weight: 400;
}

.save-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.save-bar-btn {
  padding: 6px 14px;
  font-size: 0.8125rem;
  font-weight: 500;
  font-family: inherit;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.save-bar-btn.secondary {
  background: transparent;
  border: none;
  color: #b5bac1;
}

.save-bar-btn.secondary:hover {
  text-decoration: underline;
}

.save-bar-btn.primary {
  background: #248046;
  border: none;
  color: #fff;
}

.save-bar-btn.primary:hover {
  background: #1a6334;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translate(-50%, 20px);
  opacity: 0;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background-color: var(--k-color-divider);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background-color: var(--fg3);
}

/* Mono Text */
.mono {
  font-family: 'SF Mono', 'Consolas', monospace;
}

/* Mobile Section Selector - 默认隐藏 */
.mobile-section-selector {
  display: none;
}

.mobile-section-backdrop {
  display: none;
}

/* ========== Mobile Responsive ========== */
@media (max-width: 768px) {
  .settings-view {
    padding: 0;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .view-header {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
    padding: 0.75rem;
    border-bottom: 1px solid var(--k-color-divider);
    flex-shrink: 0;
  }

  .view-title {
    font-size: 1.1rem;
  }

  .header-actions {
    width: 100%;
  }

  .action-btn {
    width: 100%;
    justify-content: center;
    padding: 0.625rem 1rem;
  }

  /* 设置内容布局 */
  .settings-content {
    flex-direction: column;
    gap: 0;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    position: relative;
    z-index: 1; /* 确保不会超过顶部导航的 z-index: 10 */
    isolation: isolate; /* 创建新的堆叠上下文 */
  }

  /* 隐藏 PC 端侧边栏 */
  .settings-sidebar {
    display: none;
  }

  /* 移动端下拉选择器 */
  .mobile-section-selector {
    display: block;
    position: relative;
    z-index: 20;
    flex-shrink: 0;
    background: var(--bg1);
    border-bottom: 1px solid var(--k-color-divider);
  }

  .selector-current {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.875rem 1rem;
    cursor: pointer;
    background: var(--bg2);
    transition: background 0.15s ease;
  }

  .selector-current:hover {
    background: var(--bg3);
  }

  .current-label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--fg1);
  }

  .selector-arrow {
    font-size: 14px;
    color: var(--fg3);
    transition: transform 0.2s ease;
  }

  .selector-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: var(--bg1);
    border: 1px solid var(--k-color-border);
    border-top: none;
    border-radius: 0 0 8px 8px;
    max-height: 60vh;
    overflow-y: auto;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    z-index: 21;
  }

  .selector-option {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--fg2);
    cursor: pointer;
    border-bottom: 1px solid var(--k-color-divider);
    transition: all 0.15s ease;
  }

  .selector-option:last-child {
    border-bottom: none;
  }

  .selector-option:hover {
    background: var(--bg3);
    color: var(--fg1);
  }

  .selector-option.active {
    background: var(--k-color-primary-fade);
    color: var(--k-color-primary);
    font-weight: 500;
  }

  .mobile-section-backdrop {
    display: block;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.4);
    z-index: 19;
  }

  /* 下拉动画 */
  .dropdown-enter-active,
  .dropdown-leave-active {
    transition: opacity 0.2s ease, transform 0.2s ease;
  }

  .dropdown-enter-from,
  .dropdown-leave-to {
    opacity: 0;
    transform: translateY(-8px);
  }

  /* 主设置区域 */
  .settings-main {
    flex: 1;
    padding: 0.75rem;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  .config-section {
    padding: 0;
  }

  .section-header {
    margin-bottom: 1rem;
  }

  .section-title {
    font-size: 1rem;
  }

  .section-desc {
    font-size: 0.75rem;
  }

  /* 表单布局 */
  .form-grid {
    gap: 1rem;
  }

  .form-row {
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-label {
    width: 100%;
    font-size: 0.8rem;
    font-weight: 500;
  }

  .form-control {
    width: 100%;
  }

  .form-hint {
    font-size: 0.7rem;
    margin-top: 0.25rem;
  }

  /* 开关 */
  .toggle-switch {
    align-self: flex-start;
  }

  /* 输入框样式 */
  .form-control :deep(.el-input__wrapper),
  .form-control :deep(.el-input-number) {
    width: 100%;
  }

  .form-control :deep(.el-input__inner) {
    font-size: 16px; /* 防止 iOS 缩放 */
  }

  .form-textarea {
    font-size: 16px;
  }

  /* 关键词列表 */
  .keyword-list {
    gap: 0.375rem;
  }

  .keyword-item {
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
  }

  .add-keyword-row {
    flex-direction: column;
    gap: 0.5rem;
  }

  .add-keyword-row input {
    font-size: 16px;
    padding: 0.625rem 0.75rem;
  }

  .add-keyword-row button {
    width: 100%;
    justify-content: center;
  }

  /* 子区域分隔线 */
  .subsection-divider {
    margin: 1rem 0 0.5rem;
    font-size: 0.65rem;
  }

  /* 保存栏 */
  .save-bar {
    bottom: 12px;
    width: calc(100% - 24px);
    max-width: none;
    padding: 0.5rem 0.75rem;
  }

  .save-bar-text {
    font-size: 0.75rem;
  }

  .save-actions {
    gap: 0.5rem;
  }

  .save-bar-btn {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
  }

}

@media (max-width: 480px) {
  .view-header {
    padding: 0.5rem;
  }

  .view-title {
    font-size: 1rem;
  }

  .selector-current {
    padding: 0.75rem;
  }

  .current-label {
    font-size: 0.8125rem;
  }

  .selector-option {
    padding: 0.625rem 0.75rem;
    font-size: 0.8125rem;
  }

  .settings-main {
    padding: 0.5rem;
  }

  .section-title {
    font-size: 0.9rem;
  }

  .form-label {
    font-size: 0.75rem;
  }

  .form-hint {
    font-size: 0.65rem;
  }

  .keyword-item {
    font-size: 0.7rem;
    padding: 0.2rem 0.4rem;
  }
}
</style>
