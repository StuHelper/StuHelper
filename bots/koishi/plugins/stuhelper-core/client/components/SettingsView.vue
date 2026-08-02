<template>
  <div class="settings-view">
    <!-- Header -->
    <header class="view-header">
      <h2 class="view-title">全局设置</h2>
      <div class="header-actions">
        <button
          type="button"
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
      <button type="button" class="action-btn" @click="loadSettings">
        <k-icon name="refresh-cw" class="btn-icon" />
        <span>重试</span>
      </button>
    </div>

    <!-- Settings Content -->
    <div v-else-if="settingsLoaded" class="settings-content">
      <!-- Mobile Section Selector -->
      <div class="mobile-section-selector">
        <button
          type="button"
          class="selector-current"
          aria-controls="settings-mobile-sections"
          :aria-expanded="sectionDropdownOpen"
          :aria-label="`选择设置分类，当前为${currentSectionLabel}`"
          @click="sectionDropdownOpen = !sectionDropdownOpen"
          @keydown.escape.stop="sectionDropdownOpen = false"
        >
          <span class="current-label">{{ currentSectionLabel }}</span>
          <k-icon
            :name="sectionDropdownOpen ? 'chevron-up' : 'chevron-down'"
            class="selector-arrow"
            aria-hidden="true"
          />
        </button>
        <transition name="dropdown">
          <div
            v-if="sectionDropdownOpen"
            id="settings-mobile-sections"
            class="selector-dropdown"
            @keydown.escape.stop="sectionDropdownOpen = false"
          >
            <button
              v-for="section in sections"
              :key="section.id"
              type="button"
              class="selector-option"
              :class="{ active: activeSection === section.id }"
              :aria-current="activeSection === section.id ? 'page' : undefined"
              @click="selectSection(section.id)"
            >
              {{ section.label }}
            </button>
          </div>
        </transition>
      </div>
      <button
        v-if="sectionDropdownOpen"
        type="button"
        class="mobile-section-backdrop"
        aria-label="关闭设置分类菜单"
        @click="sectionDropdownOpen = false"
      ></button>

      <!-- Sidebar (PC) -->
      <nav class="settings-sidebar" aria-label="全局设置分类">
        <button
          v-for="section in sections"
          :key="section.id"
          type="button"
          class="sidebar-item"
          :class="{ active: activeSection === section.id }"
          :aria-current="activeSection === section.id ? 'page' : undefined"
          @click="activeSection = section.id"
        >
          <k-icon :name="section.icon" class="sidebar-icon" aria-hidden="true" />
          <span class="sidebar-label">{{ section.label }}</span>
        </button>
      </nav>

      <!-- Main Content -->
      <div class="settings-main">
        <div v-if="actionError" class="settings-action-error" role="alert">
          <div class="settings-action-error__body">
            <strong>{{ actionErrorTitle }}</strong>
            <span>{{ actionError }}</span>
            <ul
              v-if="saveStepResults.length"
              class="settings-save-results"
              aria-label="本次保存结果"
            >
              <li v-for="result in saveStepResults" :key="result.key">
                <span>{{ result.label }}</span>
                <span
                  class="settings-save-results__status"
                  :class="`is-${result.status}`"
                >
                  {{ settingsSaveStepStatusLabel(result.status) }}
                </span>
              </li>
            </ul>
          </div>
          <div class="settings-action-error__actions">
            <button
              v-if="hasIncompleteSave"
              class="action-btn"
              type="button"
              :disabled="saving || loading"
              @click="reloadSettingsAfterSaveFailure"
            >
              重新加载服务器设置
            </button>
            <button class="action-btn" type="button" @click="clearSaveError">
              关闭
            </button>
          </div>
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
            <div class="form-row">
              <label class="form-label">群管升级表达式</label>
              <div class="form-control">
                <el-input v-model="settings.groupGuard.moderation.warningThresholdExpression" placeholder="warnings >= 3" size="small" maxlength="256" />
                <span class="form-hint">群管消息风控用，变量包括 warnings、repeats、reports</span>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">群管默认禁言秒数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.moderation.defaultMuteSeconds" :min="1" :max="2592000" size="small" />
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
                  aria-label="禁言关键词"
                  rows="4"
                  placeholder="每行一个关键词"
                  class="form-textarea"
                ></textarea>
              </div>
            </div>
          </div>

          <div class="subsection-divider">群管关键词规则</div>
          <div class="keyword-rule-toolbar">
            <button class="action-btn" type="button" @click="startNewKeywordRule">
              <k-icon name="plus" class="btn-icon" />
              <span>新增规则</span>
            </button>
          </div>
          <div v-if="settings.keywordRules.length === 0" class="keyword-rule-empty">
            暂无群管关键词规则
          </div>
          <div v-else class="keyword-rule-list">
            <button
              v-for="rule in settings.keywordRules"
              :key="rule.id"
              type="button"
              class="keyword-rule-item"
              :class="{ active: selectedKeywordRuleID === rule.id }"
              @click="loadKeywordRule(rule)"
            >
              <span class="keyword-rule-item__main">
                <span class="keyword-rule-item__id">{{ rule.id }}</span>
                <span class="keyword-rule-item__pattern">{{ rule.pattern }}</span>
              </span>
              <span class="keyword-rule-item__meta">
                {{ keywordRuleGuildLabel(rule.guildId) }} · {{ keywordRuleMatchModeLabel(rule.matchMode) }} · {{ keywordRuleActionLabel(rule.action) }}
              </span>
              <span class="keyword-rule-status" :class="{ disabled: !rule.enabled }">
                {{ rule.enabled ? '启用' : '停用' }}
              </span>
            </button>
          </div>

          <div class="keyword-rule-editor">
            <div class="form-grid">
              <div class="form-row">
                <label class="form-label">规则 ID</label>
                <div class="form-control">
                  <el-input v-model="keywordRuleDraft.id" class="mono-control" placeholder="rule-id" size="small" maxlength="128" />
                </div>
              </div>
              <div class="form-row">
                <label class="form-label">作用群号</label>
                <div class="form-control">
                  <el-input v-model="keywordRuleDraft.guildId" class="mono-control" placeholder="*" size="small" maxlength="64" />
                  <span class="form-hint">*</span>
                </div>
              </div>
              <div class="form-row full">
                <label class="form-label">匹配内容</label>
                <div class="form-control">
                  <textarea
                    v-model="keywordRuleDraft.pattern"
                    aria-label="匹配内容"
                    rows="3"
                    maxlength="256"
                    class="form-textarea"
                    placeholder="关键词或正则表达式"
                  ></textarea>
                </div>
              </div>
              <div class="form-row">
                <label class="form-label">匹配模式</label>
                <div class="form-control">
                  <el-select v-model="keywordRuleDraft.matchMode" size="small" class="settings-select">
                    <el-option label="包含" value="includes" />
                    <el-option label="正则" value="regex" />
                  </el-select>
                </div>
              </div>
              <div class="form-row">
                <label class="form-label">命中动作</label>
                <div class="form-control">
                  <el-select v-model="keywordRuleDraft.action" size="small" class="settings-select">
                    <el-option label="警告" value="warn" />
                    <el-option label="撤回" value="delete" />
                    <el-option label="禁言" value="mute" />
                    <el-option label="复核" value="review" />
                  </el-select>
                </div>
              </div>
              <div class="form-row">
                <label class="form-label">启用规则</label>
                <div class="form-control">
                  <label class="toggle-switch">
                    <input type="checkbox" v-model="keywordRuleDraft.enabled" />
                    <span class="toggle-track"></span>
                  </label>
                </div>
              </div>
              <div class="form-row">
                <label class="form-label">禁言秒数</label>
                <div class="form-control">
                  <el-input-number v-model="keywordRuleDraft.muteSeconds" :min="0" :max="2592000" size="small" />
                </div>
              </div>
              <div class="form-row full">
                <label class="form-label">规则备注</label>
                <div class="form-control">
                  <textarea
                    v-model="keywordRuleNoteText"
                    aria-label="规则备注"
                    rows="2"
                    maxlength="512"
                    class="form-textarea"
                  ></textarea>
                </div>
              </div>
            </div>
            <div v-if="keywordRuleValidationError" class="keyword-rule-error" role="alert">
              {{ keywordRuleValidationError }}
            </div>
            <div class="keyword-rule-actions">
              <button class="action-btn primary" type="button" @click="stageKeywordRuleDraft">
                <k-icon name="check" class="btn-icon" />
                <span>暂存规则</span>
              </button>
              <button
                class="action-btn danger-outline"
                type="button"
                :disabled="!selectedKeywordRuleID"
                @click="removeSelectedKeywordRule"
              >
                <k-icon name="trash-2" class="btn-icon" />
                <span>删除规则</span>
              </button>
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
                  aria-label="审核关键词"
                  rows="6"
                  placeholder="每行一个关键词"
                  class="form-textarea"
                ></textarea>
                <span class="form-hint">入群申请需要包含这些关键词才能通过审核</span>
              </div>
            </div>
          </div>
        </div>

        <!-- QQ Binding -->
        <div v-show="activeSection === 'binding'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">QQ 绑定</h3>
            <p class="section-desc">配置绑定命令字和绑定流程提示</p>
          </div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">绑定命令字</label>
              <div class="form-control">
                <el-input v-model="settings.binding.command" placeholder="绑定" size="small" maxlength="32" />
              </div>
            </div>
            <div
              v-for="field in bindingMessageFields"
              :key="field.key"
              class="form-row full"
            >
              <label class="form-label">{{ field.label }}</label>
              <div class="form-control">
                <textarea
                  v-model="settings.binding.messages[field.key]"
                  :aria-label="field.label"
                  rows="3"
                  class="form-textarea"
                ></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- Admin Commands -->
        <div v-show="activeSection === 'admin'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">管理员命令</h3>
            <p class="section-desc">配置管理员文本命令、新生审核命令和权限拒绝提示</p>
          </div>
          <div class="form-grid">
            <div
              v-for="field in adminMessageFields"
              :key="field.key"
              class="form-row full"
            >
              <label class="form-label">{{ field.label }}</label>
              <div class="form-control">
                <textarea
                  v-model="settings.admin.messages[field.key]"
                  :aria-label="field.label"
                  rows="3"
                  class="form-textarea"
                ></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- Group Guard Messages -->
        <div v-show="activeSection === 'groupGuardMessages'" class="config-section">
          <div class="section-header">
            <h3 class="section-title">群管提示</h3>
            <p class="section-desc">配置入群认证、公开命令、风控和举报提示文案</p>
          </div>
          <div v-if="groupGuardMessageResetPending" class="runtime-reset-note">
            保存后将把群管提示恢复为后端默认文案；继续编辑任一文案会取消本项恢复默认。
          </div>
          <div class="form-grid">
            <div
              v-for="field in groupGuardMessageFields"
              :key="field.key"
              class="form-row full"
            >
              <label class="form-label">{{ field.label }}</label>
              <div class="form-control">
                <textarea
                  v-model="settings.groupGuardMessages.messages[field.key]"
                  :aria-label="field.label"
                  rows="3"
                  class="form-textarea"
                  @input="groupGuardMessageResetPending = false"
                ></textarea>
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
            <div class="form-row">
              <label class="form-label">默认面数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.fun.diceSides" :min="2" :max="1000" size="small" />
                <span class="form-hint">群管插件“骰子”命令未指定面数时使用</span>
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

          <div class="subsection-divider">群管抽禁言</div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">基础随机秒数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.fun.muteLotteryBaseSeconds" :min="1" :max="2592000" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">最大禁言秒数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.fun.muteLotteryMaxSeconds" :min="1" :max="2592000" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">保底抽数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.fun.muteLotteryPityThreshold" :min="1" :max="100000" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">保底秒数</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.fun.muteLotteryPitySeconds" :min="1" :max="2592000" size="small" />
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
                  aria-label="通过关键词"
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
            <div class="form-row">
              <label class="form-label">群管复读阈值</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.moderation.repeatThreshold" :min="2" :max="10000" size="small" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">群管检测窗口</label>
              <div class="form-control">
                <el-input-number v-model="settings.groupGuard.moderation.repeatWindowSize" :min="2" :max="10000" size="small" />
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
            <div class="form-row">
              <label class="form-label">群管撤回提示</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.groupGuard.moderation.antiRecallNotify" />
                  <span class="toggle-track"></span>
                </label>
                <span class="form-hint">记录撤回事件后是否在群内发送提示</span>
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
                <el-input
                  v-model="openAIApiKeyDraft"
                  type="password"
                  :disabled="clearOpenAIApiKey"
                  placeholder="输入新密钥；留空保留当前密钥"
                  size="small"
                />
                <span class="form-hint">{{ openAIApiKeyStatusText }}</span>
                <label v-if="openAIApiKeyConfigured" class="inline-checkbox">
                  <input v-model="clearOpenAIApiKey" type="checkbox" />
                  <span>保存时清除当前密钥</span>
                </label>
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
                  aria-label="系统提示词"
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
                  aria-label="翻译提示词"
                  rows="4"
                  class="form-textarea"
                  placeholder="翻译提示词..."
                ></textarea>
              </div>
            </div>
          </div>

          <div class="subsection-divider">举报 AI 审核</div>
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">启用审核</label>
              <div class="form-control">
                <label class="toggle-switch">
                  <input type="checkbox" v-model="settings.groupGuardAI.enabled" />
                  <span class="toggle-track"></span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">审核接口</label>
              <div class="form-control">
                <el-input v-model="settings.groupGuardAI.endpoint" placeholder="https://example.com/review" size="small" maxlength="2048" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">审核模型</label>
              <div class="form-control">
                <el-input v-model="settings.groupGuardAI.model" placeholder="gpt-4.1-mini" size="small" maxlength="256" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">审核密钥</label>
              <div class="form-control">
                <el-input
                  v-model="groupGuardAIApiKeyDraft"
                  type="password"
                  :disabled="clearGroupGuardAIApiKey"
                  placeholder="输入新密钥；留空保留当前密钥"
                  size="small"
                />
                <span class="form-hint">{{ groupGuardAIApiKeyStatusText }}</span>
                <label v-if="groupGuardAIApiKeyConfigured" class="inline-checkbox">
                  <input v-model="clearGroupGuardAIApiKey" type="checkbox" />
                  <span>保存时清除当前密钥</span>
                </label>
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
                  aria-label="默认审核提示词"
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
                  aria-label="带上下文审核提示词"
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
          <button type="button" class="save-bar-btn secondary" :disabled="saving" @click="resetChanges">放弃更改</button>
          <button type="button" class="save-bar-btn primary" :disabled="saving" @click="saveSettings">
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
import { ref, onMounted, computed, toRaw, watch } from 'vue'
import { message } from '@koishijs/client'
import { assertSafeKeywordRegex } from '../../../../packages/shared/src/keyword-pattern'
import {
  adminSettingsApi,
  bindingSettingsApi,
  groupGuardAISettingsApi,
  groupGuardBehaviorSettingsApi,
  groupGuardMessageSettingsApi,
  keywordRulesApi,
  settingsApi,
  type AdminRuntimeSettings,
  type BindingRuntimeSettings,
  type GroupGuardAISettings,
  type GroupGuardAISettingsUpdate,
  type GroupGuardBehaviorSettings,
  type GroupGuardMessageSettings,
  type KeywordRule,
  type KeywordRuleAction,
  type KeywordRuleInput,
  type KeywordRuleMatchMode,
} from '../api'
import { useActionError } from '../composables/use-action-error'
import { useConfirm } from '../composables/use-confirm'
import { deepMerge, isPlainRecord, type PlainRecord } from '../models/plain-record'
import {
  runSettingsSaveSteps,
  saveKeywordRuleChanges,
  SettingsSaveStepFailure,
  type SettingsSaveStepResult,
  type SettingsSaveStepStatus,
} from '../models/settings-save'
import ConfirmDialog from './primitives/ConfirmDialog.vue'

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
  binding: BindingRuntimeSettings
  admin: AdminRuntimeSettings
  groupGuardAI: GroupGuardAISettings
  groupGuard: GroupGuardBehaviorSettings
  groupGuardMessages: GroupGuardMessageSettings
  keywordRules: KeywordRule[]
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
  },
  binding: {
    command: '绑定',
    messages: {
      directOnly: '请在私聊中发送绑定命令。',
      missingCode: '请输入绑定码，例如：{command} ABCD1234',
      successVerified: '绑定成功，当前账号已完成学生认证，加入受控群时会自动放行。',
      successUnverified: '绑定成功。当前账号还未完成学生认证，请先回到 StuHelper 完成认证。',
      unavailable: '绑定失败，平台暂时不可用。',
      invalidCode: '绑定码无效或已过期，请重新生成后再试。',
      unauthorized: '机器人服务鉴权失败，请联系管理员检查后端配置。',
      conflict: '该 QQ 号或 StuHelper 账号已经绑定过其他对象。',
      notConfigured: '后端机器人接口尚未配置完成，请联系管理员。'
    }
  },
  admin: {
    messages: {
      guardStatusCommandDescription: '查看当前群待认证成员',
      guardWarningCommandDescription: '查看成员当前警告次数',
      guardReviewListCommandDescription: '查看当前群待复核队列',
      guardBatchMuteCommandDescription: '批量禁言待认证成员',
      guardKickReviewCommandDescription: '提交踢人复核申请',
      guardBlockReviewCommandDescription: '提交踢人并拉黑复核申请',
      freshmanViewCommandDescription: '查看新生认证申请',
      freshmanApproveCommandDescription: '通过新生认证申请',
      freshmanRejectCommandDescription: '驳回新生认证申请',
      freshmanBlacklistReleaseCommandDescription: '解除新生入群认证黑名单',
      commandAccessDenied: '命令权限不足。',
      adminCommandsDisabled: '管理员命令已由 StuHelper WebUI 关闭。',
      guardGuildContextRequired: '请在群聊中执行，或显式传入群号。',
      guardWarningMissingContext: '请在群聊中执行，或显式传入群号和成员 ID。',
      guardBatchMuteGroupOnly: '请在目标群聊中执行批量禁言。',
      guardBatchMuteInvalidPayload: '请提供禁言秒数和成员 ID 列表，例如：群审禁言 120 10001,10002',
      guardBatchMuteNoTargets: '没有找到可操作的待认证成员。',
      guardBatchMuteSuccess: '已批量禁言 {count} 名成员。',
      guardBatchMuteEventSummary: '管理员批量禁言了 {memberId}',
      guardBatchMuteEventReason: '管理员批量禁言',
      guardReviewRequestMissingArgs: '请在群聊中提供成员 ID 和原因。',
      guardReviewRequestSuccess: '已提交{actionLabel}复核申请：{memberId}，原因：{reason}',
      guardReviewRequestEventSummary: '{operatorMemberId} 提交了{actionLabel}复核申请',
      guardReviewKickActionLabel: '踢人',
      guardReviewKickAndBlockActionLabel: '踢人并拉黑',
      guardPendingMembersEmpty: '当前没有待认证成员。',
      guardPendingMembersHeader: '待认证成员：',
      guardPendingMemberLine: '{memberId} 截止 {deadlineAt}',
      guardWarningCounterEmpty: '{memberId} 当前没有警告记录。',
      guardWarningCounterNoReason: '无',
      guardWarningCounterSummary: '{memberId} 当前累计警告 {total} 次，最近原因：{lastReason}',
      guardPendingReviewsEmpty: '当前没有待复核事项。',
      guardPendingReviewsHeader: '待复核队列：',
      guardPendingReviewLine: '{memberId} [{actionType}] {reason}',
      freshmanManagementGroupOnly: '请在新生审核管理群中执行此命令。',
      freshmanMissingApplicationID: '请提供申请 ID。',
      freshmanApproveInvalidFormat: '通过命令格式为：新生审核通过 <申请ID> [+30d]。',
      freshmanApproveInvalidExtension: '审批延长天数格式应为 +30d，且必须是正整数天数。',
      freshmanRejectInvalidFormat: '驳回命令格式为：新生审核驳回 <申请ID> <原因>。',
      freshmanBlacklistReleaseInvalidFormat: '解除命令格式为：新生黑名单解除 <QQ号> <群号|global>。',
      freshmanApproveSuccess: '已通过新生认证申请 {applicationID}。',
      freshmanApproveSuccessWithExtension: '已通过新生认证申请 {applicationID}，临时身份 {expiresInDays} 天后过期。',
      freshmanRejectSuccess: '已驳回新生认证申请 {applicationID}。',
      freshmanBlacklistScopeGlobal: '全局',
      freshmanBlacklistScopeGuild: '群 {guildID} 的',
      freshmanBlacklistReleaseSuccess: '已解除 {qqID} 的{scope}入群认证黑名单。',
      freshmanApplicationSummary: [
        '申请 {applicationID}',
        '状态：{status}',
        '姓名：{applicantName}',
        '学校ID：{schoolID}',
        '专业：{departmentOrMajor}',
        '临时身份过期：{provisionalExpiresAt}',
      ].join('\n'),
      freshmanApplicationDepartmentFallback: '未提供',
      freshmanApplicationExpiryFallback: '未设置',
      freshmanCommandFailed: '新生审核命令执行失败：{error}',
      freshmanOperatorQQUnbound: '你的 QQ 未绑定 StuHelper 管理员账号，请先完成管理员 QQ 绑定。',
      freshmanOperatorForbidden: '你的 StuHelper 账号没有新生审核权限。',
      freshmanManagementGuildForbidden: '当前群不在新生审核管理群白名单内。',
      freshmanBackendForbidden: '后端拒绝执行该审核命令：{message}',
    }
  },
  groupGuardAI: {
    enabled: false,
    endpoint: '',
    model: '',
    apiKeyConfigured: false,
    apiKeyMasked: ''
  },
  groupGuard: {
    fun: {
      diceSides: 100,
      muteLotteryBaseSeconds: 120,
      muteLotteryMaxSeconds: 600,
      muteLotteryPityThreshold: 5,
      muteLotteryPitySeconds: 300
    },
    moderation: {
      repeatThreshold: 3,
      repeatWindowSize: 3,
      warningThresholdExpression: 'warnings >= 3',
      defaultMuteSeconds: 600,
      antiRecallNotify: true
    }
  },
  groupGuardMessages: {
    messages: {}
  },
  keywordRules: []
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
const saveStepResults = ref<SettingsSaveStepResult[]>([])
const openAIApiKeyDraft = ref('')
const openAIApiKeyConfigured = ref(false)
const openAIApiKeyMasked = ref('')
const clearOpenAIApiKey = ref(false)
const groupGuardAIApiKeyDraft = ref('')
const groupGuardAIApiKeyConfigured = ref(false)
const groupGuardAIApiKeyMasked = ref('')
const clearGroupGuardAIApiKey = ref(false)
const selectedKeywordRuleID = ref('')
const keywordRuleDraft = ref<KeywordRuleInput>(createEmptyKeywordRule())
const keywordRuleValidationError = ref('')
const activeSection = ref('warn')
const sectionDropdownOpen = ref(false)
const settingsLoaded = computed(() => Boolean(originalSettings.value) && !loadError.value)
const hasIncompleteSave = computed(() =>
  saveStepResults.value.some((result) => result.status !== 'confirmed'),
)
const groupGuardMessageResetPending = ref(false)
let loadRequestSeq = 0

// 当前选中的 section 标签
const currentSectionLabel = computed(() => {
  const section = sections.find(s => s.id === activeSection.value)
  return section ? section.label : '警告设置'
})

const openAIApiKeyStatusText = computed(() => {
  if (clearOpenAIApiKey.value) return '保存后将清除当前密钥'
  if (!openAIApiKeyConfigured.value) return '当前未配置密钥'
  return openAIApiKeyMasked.value
    ? `当前已配置：${openAIApiKeyMasked.value}`
    : '当前已配置密钥'
})

const groupGuardAIApiKeyStatusText = computed(() => {
  if (clearGroupGuardAIApiKey.value) return '保存后将清除当前密钥'
  if (!groupGuardAIApiKeyConfigured.value) return '当前未配置密钥'
  return groupGuardAIApiKeyMasked.value
    ? `当前已配置：${groupGuardAIApiKeyMasked.value}`
    : '当前已配置密钥'
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
  return (
    JSON.stringify(settings.value) !== originalSettings.value ||
    openAIApiKeyDraft.value.trim() !== '' ||
    clearOpenAIApiKey.value ||
    groupGuardAIApiKeyDraft.value.trim() !== '' ||
    clearGroupGuardAIApiKey.value ||
    groupGuardMessageResetPending.value
  )
})

watch(clearOpenAIApiKey, (clear) => {
  if (clear) {
    openAIApiKeyDraft.value = ''
  }
})

watch(clearGroupGuardAIApiKey, (clear) => {
  if (clear) {
    groupGuardAIApiKeyDraft.value = ''
  }
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

const keywordRuleNoteText = computed({
  get: () => keywordRuleDraft.value.note || '',
  set: (value: string) => {
    const trimmed = value.trim()
    keywordRuleDraft.value.note = trimmed || null
  }
})

const sections = [
  { id: 'warn', label: '警告设置', icon: 'stuhelperGroupCenter:octicons.warning' },
  { id: 'forbidden', label: '禁言关键词', icon: 'stuhelperGroupCenter:octicons.x' },
  { id: 'keywords', label: '入群审核', icon: 'stuhelperGroupCenter:octicons.personadd' },
  { id: 'binding', label: 'QQ绑定', icon: 'stuhelperGroupCenter:octicons.link' },
  { id: 'admin', label: '管理员命令', icon: 'stuhelperGroupCenter:octicons.shield' },
  { id: 'groupGuardMessages', label: '群管提示', icon: 'stuhelperGroupCenter:octicons.discussion' },
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

const bindingMessageFields = [
  { key: 'directOnly', label: '群聊误用提示' },
  { key: 'missingCode', label: '缺少绑定码提示' },
  { key: 'successVerified', label: '绑定成功且已认证' },
  { key: 'successUnverified', label: '绑定成功但未认证' },
  { key: 'unavailable', label: '平台不可用提示' },
  { key: 'invalidCode', label: '绑定码无效提示' },
  { key: 'unauthorized', label: '服务鉴权失败提示' },
  { key: 'conflict', label: '绑定冲突提示' },
  { key: 'notConfigured', label: '后端未配置提示' },
] as const

const adminMessageFields = [
  { key: 'guardStatusCommandDescription', label: '群审状态命令说明' },
  { key: 'guardWarningCommandDescription', label: '群审警告命令说明' },
  { key: 'guardReviewListCommandDescription', label: '群审复核命令说明' },
  { key: 'guardBatchMuteCommandDescription', label: '群审批量禁言命令说明' },
  { key: 'guardKickReviewCommandDescription', label: '踢人复核申请命令说明' },
  { key: 'guardBlockReviewCommandDescription', label: '拉黑复核申请命令说明' },
  { key: 'freshmanViewCommandDescription', label: '新生审核查看命令说明' },
  { key: 'freshmanApproveCommandDescription', label: '新生审核通过命令说明' },
  { key: 'freshmanRejectCommandDescription', label: '新生审核驳回命令说明' },
  { key: 'freshmanBlacklistReleaseCommandDescription', label: '新生黑名单解除命令说明' },
  { key: 'commandAccessDenied', label: '命令权限不足提示' },
  { key: 'adminCommandsDisabled', label: '管理员命令关闭提示' },
  { key: 'guardGuildContextRequired', label: '群管理命令缺少群上下文' },
  { key: 'guardWarningMissingContext', label: '警告命令缺少上下文' },
  { key: 'guardBatchMuteGroupOnly', label: '批量禁言群聊限制' },
  { key: 'guardBatchMuteInvalidPayload', label: '批量禁言格式错误' },
  { key: 'guardBatchMuteNoTargets', label: '批量禁言无目标' },
  { key: 'guardBatchMuteSuccess', label: '批量禁言成功' },
  { key: 'guardBatchMuteEventSummary', label: '批量禁言事件摘要' },
  { key: 'guardBatchMuteEventReason', label: '批量禁言事件原因' },
  { key: 'guardReviewRequestMissingArgs', label: '复核申请缺少参数' },
  { key: 'guardReviewRequestSuccess', label: '复核申请成功' },
  { key: 'guardReviewRequestEventSummary', label: '复核申请事件摘要' },
  { key: 'guardReviewKickActionLabel', label: '踢人动作标签' },
  { key: 'guardReviewKickAndBlockActionLabel', label: '踢人并拉黑动作标签' },
  { key: 'guardPendingMembersEmpty', label: '待认证列表为空' },
  { key: 'guardPendingMembersHeader', label: '待认证列表标题' },
  { key: 'guardPendingMemberLine', label: '待认证成员行' },
  { key: 'guardWarningCounterEmpty', label: '警告记录为空' },
  { key: 'guardWarningCounterNoReason', label: '警告原因为空占位' },
  { key: 'guardWarningCounterSummary', label: '警告次数摘要' },
  { key: 'guardPendingReviewsEmpty', label: '复核队列为空' },
  { key: 'guardPendingReviewsHeader', label: '复核队列标题' },
  { key: 'guardPendingReviewLine', label: '复核队列行' },
  { key: 'freshmanManagementGroupOnly', label: '新生审核管理群限制' },
  { key: 'freshmanMissingApplicationID', label: '新生申请 ID 缺失' },
  { key: 'freshmanApproveInvalidFormat', label: '新生通过格式错误' },
  { key: 'freshmanApproveInvalidExtension', label: '新生延期格式错误' },
  { key: 'freshmanRejectInvalidFormat', label: '新生驳回格式错误' },
  { key: 'freshmanBlacklistReleaseInvalidFormat', label: '新生黑名单解除格式错误' },
  { key: 'freshmanApproveSuccess', label: '新生通过成功' },
  { key: 'freshmanApproveSuccessWithExtension', label: '新生通过并延期成功' },
  { key: 'freshmanRejectSuccess', label: '新生驳回成功' },
  { key: 'freshmanBlacklistScopeGlobal', label: '全局黑名单范围标签' },
  { key: 'freshmanBlacklistScopeGuild', label: '群黑名单范围标签' },
  { key: 'freshmanBlacklistReleaseSuccess', label: '新生黑名单解除成功' },
  { key: 'freshmanApplicationSummary', label: '新生申请详情摘要' },
  { key: 'freshmanApplicationDepartmentFallback', label: '新生专业为空占位' },
  { key: 'freshmanApplicationExpiryFallback', label: '新生临时身份过期为空占位' },
  { key: 'freshmanCommandFailed', label: '新生审核命令失败' },
  { key: 'freshmanOperatorQQUnbound', label: '审核员 QQ 未绑定' },
  { key: 'freshmanOperatorForbidden', label: '审核员权限不足' },
  { key: 'freshmanManagementGuildForbidden', label: '管理群未授权' },
  { key: 'freshmanBackendForbidden', label: '后端拒绝审核命令' },
] as const

const groupGuardMessageLabels: Record<string, string> = {
  publicReportCommandDescription: '举报命令说明',
  diceCommandDescription: '骰子命令说明',
  muteLotteryCommandDescription: '抽禁言命令说明',
  admissionQueryCommandDescription: '入群认证查询命令说明',
  admissionResendCommandDescription: '重发认证链接命令说明',
  admissionRegenerateCommandDescription: '重新生成认证链接命令说明',
  admissionSkipCommandDescription: '跳过入群认证命令说明',
  admissionResetFailureCountCommandDescription: '清空未认证次数命令说明',
  admissionReleaseBlacklistCommandDescription: '解除入群拉黑命令说明',
  admissionReminder: '入群认证提醒',
  admissionTimeoutNormal: '入群认证普通超时提示',
  admissionTimeoutWithFailures: '入群认证带失败次数超时提示',
  admissionTimeoutBlacklist: '入群认证拉黑前超时提示',
  backendPendingReminder: '后端暂不可用兜底提醒',
  admissionReleaseCompleted: '认证通过解除禁言提示',
  admissionKickTimeout: '认证超时踢出提示',
  admissionBlacklistKick: '认证失败拉黑踢出提示',
  antiRecallNotify: '防撤回群内提示',
  moderationMuteNotice: '禁言通知',
  moderationUnmuteNotice: '解除禁言通知',
  moderationKickNotice: '踢出通知',
  freshmanForwardSummary: '新生材料转发摘要',
  freshmanForwardUnknownField: '新生材料未知字段占位',
  freshmanMaterialTypeAdmissionNotice: '录取通知书材料标签',
  freshmanMaterialTypeAdmissionCertificate: '录取证明材料标签',
  publicReportMissingArgs: '举报命令缺少参数',
  publicCommandsDisabled: '公开命令关闭提示',
  muteLotteryGroupOnly: '抽禁言群聊限制',
  commandAccessDenied: '命令权限不足提示',
  diceResult: '骰子结果',
  muteLotteryResult: '抽禁言结果',
  muteLotteryPityResult: '抽禁言保底结果',
  reportGroupOnly: '举报命令群聊限制',
  reportRecordedAIUnavailable: '举报记录后 AI 不可用',
  reportAIReviewFailed: '举报 AI 审核失败',
  reportHighRisk: '举报高风险提示',
  reportMediumRisk: '举报中风险提示',
  reportLowRisk: '举报低风险提示',
  reportNoAction: '举报无动作提示',
  admissionCommandGroupOnly: '入群认证命令群聊限制',
  admissionCommandsDisabled: '入群认证管理员命令关闭提示',
  admissionCommandMissingQQ: '入群认证命令缺少 QQ',
  admissionCommandMissingOperator: '入群认证命令缺少操作人',
  admissionCommandUnsupportedPlatform: '入群认证命令平台不支持',
  admissionCommandPolicyDisabled: '当前群未启用入群认证',
  admissionCommandNotFound: '未找到入群认证记录',
  admissionCommandInvalidState: '入群认证状态不允许操作',
  admissionCommandUnauthorized: '入群认证接口未授权',
  admissionCommandFailed: '入群认证命令失败',
  admissionCommandPlatformError: '入群认证平台接口错误',
  admissionCommandMissingResendURL: '后端未返回认证链接',
  admissionCommandStaleRecord: '入群认证记录已过期',
  admissionReminderDeliveryDisabled: '认证提醒投递方式关闭',
  admissionReminderDeliveryFailure: '认证提醒投递失败',
  admissionReminderDeliveryGroupChannelLabel: '群内提醒通道标签',
  admissionReminderDeliveryDirectChannelLabel: '私聊/临时会话通道标签',
  admissionTimeCodeReminder: '入群验证码提醒',
  admissionTimeCodeVerified: '入群验证码通过提示',
  admissionTimeCodeInvalid: '入群验证码错误提示',
  admissionTimeCodeKickTimeout: '入群验证码超时踢出提示',
  admissionSkipSuccess: '跳过入群认证成功',
  admissionSkipUnmuteFailed: '跳过入群认证但解除禁言失败',
  admissionAlreadyVerifiedRegenerate: '重新生成时已认证提示',
  admissionResetFailureCountSuccess: '清空未认证次数成功',
  admissionReleaseBlacklistNotFound: '解除拉黑未找到记录',
  admissionReleaseBlacklistSuccess: '解除拉黑成功',
  admissionQuerySummary: '入群认证查询摘要',
  admissionQueryDeadlineLink: '查询链接有效期行',
  admissionQueryDeadlineSubmission: '查询学生认证截止行',
  admissionQueryDeadlineManualReview: '查询人工审核截止行',
  admissionQueryDeadlineUnset: '查询截止时间为空占位',
  admissionQueryLastBotError: '查询最近机器人错误行',
  admissionQueryQQLinked: '查询 QQ 已绑定标签',
  admissionQueryQQUnlinked: '查询 QQ 未绑定标签',
  admissionQueryStudentVerified: '查询学生已认证标签',
  admissionQueryStudentFreshmanPending: '查询新生材料待审核标签',
  admissionQueryStudentUnverified: '查询学生未认证标签',
  admissionStatusJoinedMuted: '状态：等待绑定 QQ',
  admissionStatusLinked: '状态：已绑定待认证',
  admissionStatusMaterialSubmitted: '状态：新生材料待审核',
  admissionStatusVerified: '状态：学生认证已通过',
  admissionStatusExpiredKicked: '状态：已超时移出',
  admissionStatusCancelled: '状态：已取消',
  admissionNextStepJoinedMuted: '下一步：打开链接认证',
  admissionNextStepLinked: '下一步：继续学生认证',
  admissionNextStepMaterialSubmitted: '下一步：等待人工审核',
  admissionNextStepVerifiedWithBotError: '下一步：已认证但机器人失败',
  admissionNextStepVerified: '下一步：已认证解除禁言',
  admissionNextStepExpiredKickedLinked: '下一步：超时已绑定',
  admissionNextStepExpiredKicked: '下一步：超时未绑定',
  admissionNextStepCancelled: '下一步：已取消',
  admissionNextStepDefault: '下一步默认提示',
  admissionConsoleSettingsSaved: '控制台设置保存成功',
  admissionConsoleRecordNotFound: '控制台记录不存在',
  admissionConsoleStaleRecord: '控制台记录已过期',
  admissionConsoleResendSuccess: '控制台重发成功',
  admissionConsoleVerifiedReleaseSuccess: '控制台已认证解除禁言成功',
  admissionConsoleRegenerateSuccess: '控制台重新生成成功',
  admissionConsoleSkipSuccess: '控制台跳过成功',
  admissionConsoleSkipUnmuteFailed: '控制台跳过但解除禁言失败',
  admissionConsoleResetFailureCountSuccess: '控制台清空失败次数成功',
  admissionConsoleReleaseBlacklistNotFound: '控制台未找到活动拉黑记录',
  admissionConsoleReleaseBlacklistSuccess: '控制台解除拉黑成功',
  admissionConsoleMissingResendURL: '控制台缺少认证链接',
  admissionConsoleInvalidMuteDeadline: '控制台禁言期限无效',
  admissionConsoleBotNotFound: '控制台找不到 Bot',
  admissionConsoleErrorNotFound: '控制台错误：未找到',
  admissionConsoleErrorInvalidState: '控制台错误：状态不允许',
  admissionConsoleErrorUnauthorized: '控制台错误：未授权',
  admissionConsoleErrorPlatform: '控制台错误：平台接口',
  admissionConsoleErrorFallback: '控制台错误兜底',
  moderationWarnEventSummary: '风控警告事件摘要',
  moderationMuteEventSummary: '风控禁言事件摘要',
  moderationUnmuteEventSummary: '风控解除禁言事件摘要',
  moderationKickEventSummary: '风控踢出事件摘要',
  moderationAntiRecallEventSummary: '防撤回事件摘要',
  moderationKeywordHitEventSummary: '关键词命中事件摘要',
  moderationKeywordHitReason: '关键词命中原因',
  moderationKeywordRuleHitReason: '关键词规则命中原因',
  moderationRepeatHitEventSummary: '复读命中事件摘要',
  moderationRepeatHitReason: '复读命中原因',
  moderationRepeatAutoMuteReason: '复读自动处罚原因',
  moderationMuteLotteryEventSummary: '抽禁言事件摘要',
  reportCreatedEventSummary: '举报创建事件摘要',
  reportAIReviewedEventSummary: '举报 AI 审核事件摘要',
  reportAIWarnReason: 'AI 举报警告原因',
  reportAIMuteReason: 'AI 举报禁言原因',
  reportAISummaryFallback: 'AI 举报摘要为空占位',
  admissionBlacklistEventSummary: '入群黑名单事件摘要',
  admissionJoinMutedEventSummary: '入群禁言事件摘要',
  admissionJoinAlreadyVerifiedEventSummary: '入群已认证事件摘要',
  admissionJoinBackendUnavailableEventSummary: '入群后端不可用事件摘要',
  admissionJoinBackendVerifiedEventSummary: '后端同步已认证事件摘要',
  admissionTimeCodeJoinGuardedEventSummary: '入群验证码待验证事件摘要',
  admissionTimeCodeVerifiedEventSummary: '入群验证码通过事件摘要',
  admissionTimeCodeExpiredEventSummary: '入群验证码超时事件摘要',
  admissionCommandInvalidMuteDeadline: '入群认证命令禁言期限无效',
}

const groupGuardMessageFields = computed(() => {
  const keys = Object.keys(settings.value.groupGuardMessages.messages || {})
  return keys.map((key) => ({
    key,
    label: groupGuardMessageLabels[key] || key,
  }))
})

function splitLines(value: string): string[] {
  return value.split('\n').map((line) => line.trim()).filter((line) => line)
}

function cloneDefaultSettings(): SettingsModel {
  return structuredClone(defaultSettings)
}

function parseSettingsSnapshot(value: string): SettingsModel {
  const parsed: unknown = JSON.parse(value)
  return normalizeSettingsSnapshot(parsed)
}

function normalizeSettingsSnapshot(data: unknown): SettingsModel {
  const record = isPlainRecord(data) ? data : {}
  const merged = deepMerge(cloneDefaultSettings(), record)
  merged.binding = normalizeBindingSettings(record.binding)
  merged.admin = normalizeAdminSettings(record.admin)
  merged.groupGuardAI = normalizeGroupGuardAISettings(record.groupGuardAI)
  merged.groupGuard = normalizeGroupGuardBehaviorSettings(record.groupGuard)
  merged.groupGuardMessages = normalizeGroupGuardMessageSettings(record.groupGuardMessages)
  merged.keywordRules = normalizeKeywordRules(record.keywordRules)
  stripOpenAIApiKeyMetadata(merged)
  stripGroupGuardAIApiKeyMetadata(merged)
  return merged
}

function normalizeBindingSettings(data: unknown): BindingRuntimeSettings {
  return deepMerge(
    structuredClone(defaultSettings.binding) as unknown as PlainRecord,
    data,
  ) as unknown as BindingRuntimeSettings
}

function normalizeAdminSettings(data: unknown): AdminRuntimeSettings {
  return deepMerge(
    structuredClone(defaultSettings.admin) as unknown as PlainRecord,
    data,
  ) as unknown as AdminRuntimeSettings
}

function normalizeGroupGuardAISettings(data: unknown): GroupGuardAISettings {
  return deepMerge(
    structuredClone(defaultSettings.groupGuardAI) as unknown as PlainRecord,
    data,
  ) as unknown as GroupGuardAISettings
}

function normalizeGroupGuardBehaviorSettings(data: unknown): GroupGuardBehaviorSettings {
  return deepMerge(
    structuredClone(defaultSettings.groupGuard) as unknown as PlainRecord,
    data,
  ) as unknown as GroupGuardBehaviorSettings
}

function normalizeGroupGuardMessageSettings(data: unknown): GroupGuardMessageSettings {
  return deepMerge(
    structuredClone(defaultSettings.groupGuardMessages) as unknown as PlainRecord,
    data,
  ) as unknown as GroupGuardMessageSettings
}

function normalizeKeywordRules(data: unknown): KeywordRule[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data
    .map((item) => normalizeKeywordRule(item))
    .filter((item): item is KeywordRule => Boolean(item))
    .sort(sortKeywordRules)
}

function normalizeKeywordRule(data: unknown): KeywordRule | null {
  if (!isPlainRecord(data)) {
    return null
  }
  const id = readStringField(data.id)
  const guildId = readStringField(data.guildId)
  const pattern = readStringField(data.pattern)
  const matchMode = readKeywordRuleMatchMode(data.matchMode)
  const action = readKeywordRuleAction(data.action)
  if (!id || !guildId || !pattern || !matchMode || !action || typeof data.enabled !== 'boolean') {
    return null
  }
  const muteSeconds = typeof data.muteSeconds === 'number' && Number.isFinite(data.muteSeconds)
    ? Math.max(0, Math.trunc(data.muteSeconds))
    : 0
  return {
    id,
    guildId,
    pattern,
    matchMode,
    action,
    enabled: data.enabled,
    muteSeconds,
    note: typeof data.note === 'string' && data.note.trim() ? data.note.trim() : null,
    createdAt: readStringField(data.createdAt) || undefined,
    updatedAt: readStringField(data.updatedAt) || undefined,
  }
}

function applyOpenAIApiKeyMetadata(data: unknown) {
  const openai = isPlainRecord(data) && isPlainRecord(data.openai) ? data.openai : {}
  openAIApiKeyConfigured.value = openai.apiKeyConfigured === true
  openAIApiKeyMasked.value = typeof openai.apiKeyMasked === 'string' ? openai.apiKeyMasked : ''
  openAIApiKeyDraft.value = ''
  clearOpenAIApiKey.value = false
}

function applyGroupGuardAIApiKeyMetadata(data: unknown, clearIntent = true) {
  const ai = isPlainRecord(data) ? data : {}
  groupGuardAIApiKeyConfigured.value = ai.apiKeyConfigured === true
  groupGuardAIApiKeyMasked.value = typeof ai.apiKeyMasked === 'string' ? ai.apiKeyMasked : ''
  if (clearIntent) {
    groupGuardAIApiKeyDraft.value = ''
    clearGroupGuardAIApiKey.value = false
  }
}

function stripOpenAIApiKeyMetadata(model: SettingsModel) {
  const openai = model.openai as unknown as PlainRecord
  delete openai.apiKeyConfigured
  delete openai.apiKeyMasked
  delete openai.newApiKey
  delete openai.clearApiKey
  model.openai.apiKey = ''
}

function stripGroupGuardAIApiKeyMetadata(model: SettingsModel) {
  stripGroupGuardAISettingsMetadata(model.groupGuardAI)
}

function stripGroupGuardAISettingsMetadata(settings: GroupGuardAISettings) {
  const ai = settings as unknown as PlainRecord
  delete ai.id
  delete ai.createdAt
  delete ai.updatedAt
  delete ai.apiKey
  delete ai.newApiKey
  delete ai.clearApiKey
  delete ai.apiKeyConfigured
  delete ai.apiKeyMasked
}

interface APIKeySaveIntent {
  newApiKey: string
  clear: boolean
}

interface SettingsSavePlan {
  model: SettingsModel
  openAIKey: APIKeySaveIntent
  groupGuardAIKey: APIKeySaveIntent
  resetGroupGuardMessages: boolean
}

type DedicatedSettingsKey =
  | 'binding'
  | 'admin'
  | 'groupGuardAI'
  | 'groupGuard'
  | 'groupGuardMessages'

function captureSettingsSavePlan(): SettingsSavePlan {
  return {
    model: structuredClone(toRaw(settings.value)),
    openAIKey: {
      newApiKey: openAIApiKeyDraft.value.trim(),
      clear: clearOpenAIApiKey.value,
    },
    groupGuardAIKey: {
      newApiKey: groupGuardAIApiKeyDraft.value.trim(),
      clear: clearGroupGuardAIApiKey.value,
    },
    resetGroupGuardMessages: groupGuardMessageResetPending.value,
  }
}

function buildSettingsUpdatePayload(
  model: SettingsModel,
  keyIntent: APIKeySaveIntent,
): PlainRecord {
  const payload = structuredClone(model) as unknown as PlainRecord
  const openai = isPlainRecord(payload.openai) ? { ...payload.openai } : {}

  delete payload.binding
  delete payload.admin
  delete payload.groupGuardAI
  delete payload.groupGuard
  delete payload.groupGuardMessages
  delete payload.keywordRules
  delete openai.apiKey
  delete openai.apiKeyConfigured
  delete openai.apiKeyMasked
  if (keyIntent.clear) {
    openai.clearApiKey = true
  } else if (keyIntent.newApiKey) {
    openai.newApiKey = keyIntent.newApiKey
  }

  payload.openai = openai
  return payload
}

function buildBindingSettingsPayload(model: SettingsModel): BindingRuntimeSettings {
  const binding = model.binding
  return {
    command: binding.command,
    messages: structuredClone(binding.messages),
  }
}

function buildAdminSettingsPayload(model: SettingsModel): AdminRuntimeSettings {
  return {
    messages: structuredClone(model.admin.messages),
  }
}

function buildGroupGuardAISettingsPayload(
  model: SettingsModel,
  keyIntent: APIKeySaveIntent,
): GroupGuardAISettingsUpdate {
  const ai = model.groupGuardAI
  const payload: GroupGuardAISettingsUpdate = {
    enabled: ai.enabled,
    endpoint: ai.endpoint,
    model: ai.model,
  }
  if (keyIntent.clear) {
    payload.clearApiKey = true
  } else if (keyIntent.newApiKey) {
    payload.newApiKey = keyIntent.newApiKey
  }
  return payload
}

function buildGroupGuardBehaviorSettingsPayload(model: SettingsModel): GroupGuardBehaviorSettings {
  const groupGuard = model.groupGuard
  return {
    fun: structuredClone(groupGuard.fun),
    moderation: structuredClone(groupGuard.moderation),
  }
}

function buildGroupGuardMessageSettingsPayload(model: SettingsModel): GroupGuardMessageSettings {
  return {
    messages: structuredClone(model.groupGuardMessages.messages),
  }
}

function buildKeywordRulePayload(rule: KeywordRule): KeywordRuleInput {
  return {
    id: rule.id,
    guildId: rule.guildId,
    pattern: rule.pattern,
    matchMode: rule.matchMode,
    action: rule.action,
    enabled: rule.enabled,
    muteSeconds: rule.muteSeconds,
    note: rule.note,
  }
}

async function saveKeywordRules(submitted: readonly KeywordRule[]) {
  const original = parseSettingsSnapshot(originalSettings.value).keywordRules
  await saveKeywordRuleChanges({
    original,
    next: submitted,
    compareRules: sortKeywordRules,
    deleteRule: async (id) => {
      await keywordRulesApi.delete(id)
    },
    upsertRule: async (rule) => {
      await keywordRulesApi.upsert(buildKeywordRulePayload(rule))
    },
    onBaselineChange: (rules) => {
      updateOriginalSettings((baseline) => {
        baseline.keywordRules = structuredClone(rules) as KeywordRule[]
      })
    },
  })
}

function updateOriginalSettings(mutator: (baseline: SettingsModel) => void) {
  const baseline = parseSettingsSnapshot(originalSettings.value)
  mutator(baseline)
  originalSettings.value = JSON.stringify(baseline)
}

function commitCoreSettingsBaseline(submitted: SettingsModel) {
  const baseline = parseSettingsSnapshot(originalSettings.value)
  const next = structuredClone(submitted)
  next.binding = baseline.binding
  next.admin = baseline.admin
  next.groupGuardAI = baseline.groupGuardAI
  next.groupGuard = baseline.groupGuard
  next.groupGuardMessages = baseline.groupGuardMessages
  next.keywordRules = baseline.keywordRules
  originalSettings.value = JSON.stringify(next)
}

function commitConfirmedSettingsSlice<K extends DedicatedSettingsKey>(
  key: K,
  submitted: SettingsModel[K],
  persisted: SettingsModel[K],
) {
  updateOriginalSettings((baseline) => {
    baseline[key] = structuredClone(persisted)
  })
  if (JSON.stringify(settings.value[key]) === JSON.stringify(submitted)) {
    settings.value[key] = structuredClone(persisted)
  }
}

function confirmOpenAIKeyIntent(intent: APIKeySaveIntent) {
  if (
    openAIApiKeyDraft.value.trim() !== intent.newApiKey ||
    clearOpenAIApiKey.value !== intent.clear
  ) {
    return
  }
  if (intent.clear) {
    openAIApiKeyConfigured.value = false
    openAIApiKeyMasked.value = ''
  } else if (intent.newApiKey) {
    openAIApiKeyConfigured.value = true
    openAIApiKeyMasked.value = ''
  } else {
    return
  }
  openAIApiKeyDraft.value = ''
  clearOpenAIApiKey.value = false
}

function confirmGroupGuardAIKeyIntent(
  intent: APIKeySaveIntent,
  persisted: GroupGuardAISettings,
) {
  applyGroupGuardAIApiKeyMetadata(persisted, false)
  if (
    groupGuardAIApiKeyDraft.value.trim() !== intent.newApiKey ||
    clearGroupGuardAIApiKey.value !== intent.clear
  ) {
    return
  }
  groupGuardAIApiKeyDraft.value = ''
  clearGroupGuardAIApiKey.value = false
}

function settingsSaveStepStatusLabel(status: SettingsSaveStepStatus) {
  const labels: Record<SettingsSaveStepStatus, string> = {
    confirmed: '已确认',
    unconfirmed: '结果未确认',
    'not-run': '未执行',
  }
  return labels[status]
}

function clearSaveError() {
  clearActionError()
  saveStepResults.value = []
}

const loadSettings = async (): Promise<boolean> => {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  loadError.value = ''
  clearActionError()
  try {
    const [data, bindingData, adminData, groupGuardAIData, groupGuardData, groupGuardMessageData, keywordRulesData] = await Promise.all([
      settingsApi.get(),
      bindingSettingsApi.get(),
      adminSettingsApi.get(),
      groupGuardAISettingsApi.get(),
      groupGuardBehaviorSettingsApi.get(),
      groupGuardMessageSettingsApi.get(),
      keywordRulesApi.list(),
    ])
    if (requestSeq !== loadRequestSeq) return false
    const merged = normalizeSettingsSnapshot({
      ...(isPlainRecord(data) ? data : {}),
      binding: bindingData,
      admin: adminData,
      groupGuardAI: groupGuardAIData,
      groupGuard: groupGuardData,
      groupGuardMessages: groupGuardMessageData,
      keywordRules: keywordRulesData,
    })
    applyOpenAIApiKeyMetadata(data)
    applyGroupGuardAIApiKeyMetadata(groupGuardAIData)
    settings.value = merged
    groupGuardMessageResetPending.value = false
    resetKeywordRuleEditor()
    // 保存原始设置用于比较
    originalSettings.value = JSON.stringify(settings.value)
    saveStepResults.value = []
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
  clearSaveError()
  try {
    const plan = captureSettingsSavePlan()
    await runSettingsSaveSteps([
      {
        key: 'core',
        label: '群管中心设置',
        run: async () => {
          await settingsApi.update(buildSettingsUpdatePayload(plan.model, plan.openAIKey))
          commitCoreSettingsBaseline(plan.model)
          confirmOpenAIKeyIntent(plan.openAIKey)
        },
      },
      {
        key: 'binding',
        label: 'QQ 绑定提示',
        run: async () => {
          const persisted = await bindingSettingsApi.update(buildBindingSettingsPayload(plan.model))
          commitConfirmedSettingsSlice('binding', plan.model.binding, persisted)
        },
      },
      {
        key: 'admin',
        label: '管理员命令提示',
        run: async () => {
          const persisted = await adminSettingsApi.update(buildAdminSettingsPayload(plan.model))
          commitConfirmedSettingsSlice('admin', plan.model.admin, persisted)
        },
      },
      {
        key: 'group-guard-ai',
        label: '群管 AI 设置',
        run: async () => {
          const response = await groupGuardAISettingsApi.update(
            buildGroupGuardAISettingsPayload(plan.model, plan.groupGuardAIKey),
          )
          const persisted = normalizeGroupGuardAISettings(response)
          confirmGroupGuardAIKeyIntent(plan.groupGuardAIKey, persisted)
          stripGroupGuardAISettingsMetadata(persisted)
          commitConfirmedSettingsSlice('groupGuardAI', plan.model.groupGuardAI, persisted)
        },
      },
      {
        key: 'group-guard-behavior',
        label: '群管行为设置',
        run: async () => {
          const persisted = await groupGuardBehaviorSettingsApi.update(
            buildGroupGuardBehaviorSettingsPayload(plan.model),
          )
          commitConfirmedSettingsSlice('groupGuard', plan.model.groupGuard, persisted)
        },
      },
      {
        key: 'group-guard-messages',
        label: '群管提示设置',
        run: async () => {
          const persisted = plan.resetGroupGuardMessages
            ? await groupGuardMessageSettingsApi.reset()
            : await groupGuardMessageSettingsApi.update(
              buildGroupGuardMessageSettingsPayload(plan.model),
            )
          commitConfirmedSettingsSlice(
            'groupGuardMessages',
            plan.model.groupGuardMessages,
            persisted,
          )
          if (
            plan.resetGroupGuardMessages &&
            groupGuardMessageResetPending.value
          ) {
            groupGuardMessageResetPending.value = false
          }
        },
      },
      {
        key: 'keyword-rules',
        label: '关键词规则',
        run: async () => {
          await saveKeywordRules(plan.model.keywordRules)
        },
      },
    ], (results) => {
      saveStepResults.value = results.map((result) => ({ ...result }))
    })
    message.success('设置已保存')
    await loadSettings()
  } catch (cause) {
    const title = cause instanceof SettingsSaveStepFailure
      ? '设置未完全保存'
      : '保存失败'
    setActionError(title, cause, '保存设置失败')
  } finally {
    saving.value = false
  }
}

const reloadSettingsAfterSaveFailure = async () => {
  const confirmed = await confirm({
    title: '重新加载服务器设置',
    message: '重新加载会放弃当前表单中尚未确认保存的内容，并以服务器实际状态为准。确定继续吗？',
    tone: 'normal',
  })
  if (confirmed) {
    await loadSettings()
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
    clearSaveError()
    // 从原始设置恢复
    settings.value = parseSettingsSnapshot(originalSettings.value)
    openAIApiKeyDraft.value = ''
    clearOpenAIApiKey.value = false
    groupGuardAIApiKeyDraft.value = ''
    clearGroupGuardAIApiKey.value = false
    groupGuardMessageResetPending.value = false
    resetKeywordRuleEditor()
    message.success('已放弃更改')
  }
}

const resetToDefault = async () => {
  if (!settingsLoaded.value) return

  const confirmed = await confirm({
    title: '恢复默认设置',
    message: '确定要将所有设置恢复为默认值吗？此操作将覆盖当前所有设置；如果当前已配置 AI API 密钥，保存时也会清除。',
    tone: 'danger'
  })
  
  if (confirmed) {
    clearSaveError()
    const currentGroupGuardMessages = settings.value.groupGuardMessages
    // 恢复为默认设置
    settings.value = cloneDefaultSettings()
    settings.value.groupGuardMessages = currentGroupGuardMessages
    openAIApiKeyDraft.value = ''
    clearOpenAIApiKey.value = openAIApiKeyConfigured.value
    groupGuardAIApiKeyDraft.value = ''
    clearGroupGuardAIApiKey.value = groupGuardAIApiKeyConfigured.value
    groupGuardMessageResetPending.value = true
    resetKeywordRuleEditor()
    message.success('已恢复默认设置，请保存以应用更改')
  }
}

function createEmptyKeywordRule(): KeywordRuleInput {
  return {
    id: '',
    guildId: '*',
    pattern: '',
    matchMode: 'includes',
    action: 'warn',
    enabled: true,
    muteSeconds: 0,
    note: null,
  }
}

function startNewKeywordRule() {
  selectedKeywordRuleID.value = ''
  keywordRuleDraft.value = createEmptyKeywordRule()
  keywordRuleValidationError.value = ''
}

function loadKeywordRule(rule: KeywordRuleInput) {
  selectedKeywordRuleID.value = rule.id
  keywordRuleDraft.value = cloneKeywordRuleInput(rule)
  keywordRuleValidationError.value = ''
}

async function removeSelectedKeywordRule() {
  if (!selectedKeywordRuleID.value) return
  const target = selectedKeywordRuleID.value
  const confirmed = await confirm({
    title: '删除关键词规则',
    message: `确定删除规则 ${target} 吗？`,
    tone: 'danger',
  })
  if (!confirmed) return
  settings.value.keywordRules = settings.value.keywordRules.filter((rule) => rule.id !== target)
  startNewKeywordRule()
}

function stageKeywordRuleDraft() {
  try {
    const rule = validateKeywordRuleDraft(keywordRuleDraft.value)
    const duplicate = settings.value.keywordRules.find((item) => item.id === rule.id && item.id !== selectedKeywordRuleID.value)
    if (duplicate) {
      throw new Error('规则 ID 已存在')
    }
    const index = selectedKeywordRuleID.value
      ? settings.value.keywordRules.findIndex((item) => item.id === selectedKeywordRuleID.value)
      : settings.value.keywordRules.findIndex((item) => item.id === rule.id)
    const nextRule: KeywordRule = { ...rule }
    if (index >= 0) {
      settings.value.keywordRules.splice(index, 1, nextRule)
    } else {
      settings.value.keywordRules.push(nextRule)
    }
    settings.value.keywordRules.sort(sortKeywordRules)
    selectedKeywordRuleID.value = rule.id
    keywordRuleDraft.value = cloneKeywordRuleInput(rule)
    keywordRuleValidationError.value = ''
  } catch (cause) {
    keywordRuleValidationError.value = errorMessage(cause, '关键词规则无效')
  }
}

function resetKeywordRuleEditor() {
  startNewKeywordRule()
}

function validateKeywordRuleDraft(input: KeywordRuleInput): KeywordRuleInput {
  const id = input.id.trim()
  const guildId = input.guildId.trim()
  const pattern = input.pattern.trim()
  const note = input.note?.trim() || null
  if (!id) throw new Error('规则 ID 不能为空')
  if (!/^[A-Za-z0-9._:-]+$/.test(id)) throw new Error('规则 ID 只能包含字母、数字、点、下划线、冒号或连字符')
  if (!guildId) throw new Error('作用群号不能为空')
  if (guildId !== '*' && !/^\d+$/.test(guildId)) throw new Error('作用群号必须是数字群号或 *')
  if (!pattern) throw new Error('匹配内容不能为空')
  if (pattern.length > 256) throw new Error('匹配内容最多 256 字符')
  if (!isKeywordRuleMatchMode(input.matchMode)) throw new Error('匹配模式无效')
  if (!isKeywordRuleAction(input.action)) throw new Error('命中动作无效')
  if (!Number.isInteger(input.muteSeconds) || input.muteSeconds < 0 || input.muteSeconds > 2592000) {
    throw new Error('禁言秒数必须是 0 到 2592000 之间的整数')
  }
  if (note && note.length > 512) throw new Error('规则备注最多 512 字符')
  if (input.matchMode === 'regex') {
    try {
      assertSafeKeywordRegex(pattern, 256)
    } catch {
      throw new Error('正则表达式无效或存在高风险回溯')
    }
  }
  return {
    id,
    guildId,
    pattern,
    matchMode: input.matchMode,
    action: input.action,
    enabled: input.enabled,
    muteSeconds: input.muteSeconds,
    note,
  }
}

function cloneKeywordRuleInput(rule: KeywordRuleInput): KeywordRuleInput {
  return {
    id: rule.id,
    guildId: rule.guildId,
    pattern: rule.pattern,
    matchMode: rule.matchMode,
    action: rule.action,
    enabled: rule.enabled,
    muteSeconds: rule.muteSeconds,
    note: rule.note,
  }
}

function keywordRuleGuildLabel(guildId: string) {
  return guildId === '*' ? '全局' : guildId
}

function keywordRuleMatchModeLabel(mode: KeywordRuleMatchMode) {
  return mode === 'regex' ? '正则' : '包含'
}

function keywordRuleActionLabel(action: KeywordRuleAction) {
  const labels: Record<KeywordRuleAction, string> = {
    warn: '警告',
    delete: '撤回',
    mute: '禁言',
    review: '复核',
  }
  return labels[action]
}

function sortKeywordRules(left: KeywordRule, right: KeywordRule) {
  const scopeOrder = (left.guildId === '*' ? 0 : 1) - (right.guildId === '*' ? 0 : 1)
  if (scopeOrder !== 0) return scopeOrder
  const guildOrder = left.guildId.localeCompare(right.guildId)
  if (guildOrder !== 0) return guildOrder
  return left.id.localeCompare(right.id)
}

function readStringField(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function readKeywordRuleMatchMode(value: unknown): KeywordRuleMatchMode | null {
  return isKeywordRuleMatchMode(value) ? value : null
}

function readKeywordRuleAction(value: unknown): KeywordRuleAction | null {
  return isKeywordRuleAction(value) ? value : null
}

function isKeywordRuleMatchMode(value: unknown): value is KeywordRuleMatchMode {
  return value === 'includes' || value === 'regex'
}

function isKeywordRuleAction(value: unknown): value is KeywordRuleAction {
  return value === 'warn' || value === 'delete' || value === 'mute' || value === 'review'
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

.settings-action-error__actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
  align-items: center;
}

.settings-save-results {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px 12px;
  padding: 8px 0 0;
  margin: 0;
  list-style: none;
}

.settings-save-results li {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  color: var(--fg2);
}

.settings-save-results__status {
  flex-shrink: 0;
  font-weight: 600;
}

.settings-save-results__status.is-confirmed {
  color: var(--k-color-success);
}

.settings-save-results__status.is-unconfirmed {
  color: var(--k-color-danger);
}

.settings-save-results__status.is-not-run {
  color: var(--fg3);
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
  width: 100%;
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
  border: 0;
  background: transparent;
  font-family: inherit;
  text-align: left;
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

.runtime-reset-note {
  padding: 10px 12px;
  color: var(--k-color-warning);
  background: var(--k-color-warning-fade);
  border: 1px solid color-mix(in srgb, var(--k-color-warning) 34%, transparent);
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
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

.inline-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--fg2);
  cursor: pointer;
}

.inline-checkbox input {
  margin: 0;
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

.keyword-rule-toolbar {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 10px;
}

.keyword-rule-empty {
  padding: 10px 12px;
  color: var(--fg3);
  background: var(--bg1);
  border: 1px solid var(--k-color-border);
  border-radius: 6px;
  font-size: 12px;
}

.keyword-rule-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.keyword-rule-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(150px, auto) auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 42px;
  padding: 8px 10px;
  color: var(--fg2);
  background: var(--bg0);
  border: 1px solid var(--k-color-border);
  border-radius: 6px;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.keyword-rule-item:hover,
.keyword-rule-item.active {
  border-color: color-mix(in srgb, var(--k-color-primary) 55%, var(--k-color-border));
  background: var(--bg1);
}

.keyword-rule-item__main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.keyword-rule-item__id,
.keyword-rule-item__pattern {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.keyword-rule-item__id {
  max-width: 160px;
  color: var(--fg1);
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 12px;
  font-weight: 600;
}

.keyword-rule-item__pattern {
  color: var(--fg3);
  font-size: 12px;
}

.keyword-rule-item__meta {
  min-width: 0;
  overflow: hidden;
  color: var(--fg3);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.keyword-rule-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 22px;
  padding: 0 8px;
  color: var(--k-color-success);
  background: var(--k-color-success-fade);
  border-radius: 999px;
  font-size: 11px;
  line-height: 1;
  white-space: nowrap;
}

.keyword-rule-status.disabled {
  color: var(--fg3);
  background: var(--bg2);
}

.keyword-rule-editor {
  padding: 12px 0 0;
  border-top: 1px solid var(--k-color-divider);
}

.keyword-rule-error {
  margin-top: 10px;
  padding: 8px 10px;
  color: var(--k-color-danger);
  background: var(--k-color-danger-fade);
  border: 1px solid color-mix(in srgb, var(--k-color-danger) 35%, transparent);
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
}

.keyword-rule-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.mono-control {
  max-width: 260px;
  font-family: 'SF Mono', 'Consolas', monospace;
}

.settings-select {
  width: 160px;
}

.mono-control :deep(.el-input__inner) {
  font-family: 'SF Mono', 'Consolas', monospace;
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

  .settings-action-error {
    flex-direction: column;
    align-items: stretch;
  }

  .settings-action-error__actions {
    width: 100%;
  }

  .settings-action-error__actions .action-btn {
    flex: 1;
  }

  .settings-save-results {
    grid-template-columns: minmax(0, 1fr);
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
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.875rem 1rem;
    cursor: pointer;
    background: var(--bg2);
    transition: background 0.15s ease;
    border: 0;
    color: inherit;
    font-family: inherit;
    text-align: left;
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
    display: block;
    width: 100%;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--fg2);
    cursor: pointer;
    border-bottom: 1px solid var(--k-color-divider);
    transition: all 0.15s ease;
    background: transparent;
    border-top: 0;
    border-left: 0;
    border-right: 0;
    font-family: inherit;
    text-align: left;
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
    border: 0;
    cursor: default;
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

  .keyword-rule-item {
    grid-template-columns: minmax(0, 1fr);
    align-items: flex-start;
    gap: 6px;
  }

  .keyword-rule-item__main {
    width: 100%;
  }

  .keyword-rule-item__id {
    max-width: 46%;
  }

  .keyword-rule-item__meta {
    width: 100%;
  }

  .keyword-rule-status {
    justify-self: flex-start;
  }

  .keyword-rule-actions {
    flex-direction: column;
  }

  .keyword-rule-actions .action-btn {
    width: 100%;
    justify-content: center;
  }

  .mono-control,
  .settings-select {
    width: 100%;
    max-width: none;
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
