<template>
  <div class="chat-view">
    <!-- 侧边栏：会话列表 -->
    <div class="chat-sidebar">
      <div class="sidebar-header">
        <h3>实时消息</h3>
        <div class="status-indicator">
          <span class="dot"></span> 实时接收中
        </div>
      </div>
      
      <!-- 连接群聊按钮 -->
      <div class="connect-group-bar">
        <button class="connect-btn" @click="showConnectDialog = true">
          <k-icon name="plus" /> 新建会话
        </button>
      </div>

      <div class="session-list">
        <div v-if="sessions.length === 0" class="empty-sessions">
          等待消息...
        </div>
        <div
          v-for="session in sessions"
          :key="session.key"
          class="session-item"
          :class="{ active: currentSessionId === session.key }"
          @click="selectSession(session.key)"
        >
          <div class="session-icon">
            <img v-if="session.avatar" :src="session.avatar" @error="handleAvatarError($event, true)" />
            <k-icon v-else :name="session.type === 'group' ? 'users' : 'user'" />
          </div>
          <div class="session-info">
            <div class="session-name" :title="session.name">{{ session.name }}</div>
            <div class="session-preview">{{ session.lastMessage?.content || '' }}</div>
          </div>
          <div class="session-meta">
            <span class="time">{{ formatTimeShort(session.lastMessage?.timestamp) }}</span>
            <span class="badge" v-if="session.unread > 0">{{ session.unread }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 主区域：聊天窗口 -->
    <div class="chat-main">
      <div v-if="currentSession" class="chat-container">
        <div class="chat-header">
          <div class="header-info">
            <div class="header-icon">
              <img v-if="currentSession.avatar" :src="currentSession.avatar" @error="handleAvatarError($event, true)" />
              <k-icon v-else :name="currentSession.type === 'group' ? 'users' : 'user'" />
            </div>
            <span class="header-name">{{ currentSession.name }}</span>
            <span class="header-id">{{ currentSession.id }}</span>
          </div>
          <div class="header-platform">
            <span class="platform-tag">{{ currentSession.platform }}</span>
          </div>
        </div>

        <div class="message-list" ref="messageListRef">
          <div
            v-for="msg in currentSession.messages"
            :key="msg.id"
            class="message-row"
            :class="{ self: isSelf(msg) }"
            @contextmenu.prevent="showContextMenu($event, msg)"
          >
            <div class="message-avatar">
              <img v-if="msg.avatar" :src="msg.avatar" @error="handleAvatarError" />
              <div v-else class="avatar-placeholder">{{ msg.username[0]?.toUpperCase() }}</div>
            </div>
            <div class="message-content-wrapper">
              <div class="message-meta">
                <span class="username">{{ msg.username }}</span>
                <span class="timestamp">{{ formatTimeDetail(msg.timestamp) }}</span>
              </div>
              <div class="message-bubble">
                <ChatMessageContent
                  :message="msg"
                  :members="members"
                  :session-messages="currentSession?.messages ?? []"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="chat-input-area">
          <!-- 待发送图片预览 -->
          <div class="pending-images" v-if="pendingImages.length > 0">
            <div
              v-for="(img, index) in pendingImages"
              :key="index"
              class="pending-image-item"
            >
              <img :src="img.dataUrl" />
              <button class="remove-image-btn" @click="removePendingImage(index)">×</button>
            </div>
          </div>
          <div class="input-row">
            <textarea
              ref="inputRef"
              v-model="inputText"
              class="chat-input"
              placeholder="发送消息... (Enter 发送, Shift+Enter 换行, 可粘贴图片)"
              @keydown.enter.exact.prevent="sendMessage"
              @paste="handlePaste"
            ></textarea>
            <button class="send-btn" @click="sendMessage" :disabled="(!inputText.trim() && pendingImages.length === 0) || sending">
              <k-icon name="send" v-if="!sending" />
              <k-icon name="loader" class="spin" v-else />
            </button>
          </div>
        </div>
      </div>

      <div v-else class="empty-chat">
        <div class="empty-content">
          <k-icon name="message-square" class="large-icon" />
          <h3>开始聊天</h3>
          <p>从左侧选择一个会话，或等待新消息接入</p>
        </div>
      </div>
    </div>

    <!-- 右侧：群成员列表（仅群聊显示） -->
    <div class="members-sidebar" v-if="currentSession?.type === 'group'" :class="{ collapsed: membersSidebarCollapsed }">
      <div class="members-header">
        <div class="members-title">
          <h3>群成员</h3>
          <span class="member-count" v-if="!loadingMembers">{{ members.length }}</span>
        </div>
        <button class="collapse-btn" @click="membersSidebarCollapsed = !membersSidebarCollapsed">
          {{ membersSidebarCollapsed ? '◀' : '▶' }}
        </button>
      </div>
      
      <template v-if="!membersSidebarCollapsed">
        <!-- 搜索框 -->
        <div class="members-search">
          <input
            type="text"
            v-model="memberSearch"
            placeholder="搜索成员..."
            class="search-input"
          />
        </div>

        <!-- 成员列表 -->
        <div class="members-list" v-if="!loadingMembers">
          <!-- 群主分组 -->
          <template v-if="filteredOwners.length > 0">
            <div class="member-group-header">
              <span class="crown-icon">👑</span> 群主 — {{ filteredOwners.length }}
            </div>
            <div
              v-for="member in filteredOwners"
              :key="member.id"
              class="member-item owner"
              @click="onMemberClick(member)"
            >
              <div class="member-avatar">
                <img :src="member.avatar" @error="handleMemberAvatarError" />
              </div>
              <div class="member-info">
                <div class="member-name">{{ member.name }}</div>
                <div class="member-title" v-if="member.title">{{ member.title }}</div>
              </div>
            </div>
          </template>

          <!-- 管理员分组 -->
          <template v-if="filteredAdmins.length > 0">
            <div class="member-group-header">
              <span class="admin-icon">⚙️</span> 管理员 — {{ filteredAdmins.length }}
            </div>
            <div
              v-for="member in filteredAdmins"
              :key="member.id"
              class="member-item admin"
              @click="onMemberClick(member)"
            >
              <div class="member-avatar">
                <img :src="member.avatar" @error="handleMemberAvatarError" />
              </div>
              <div class="member-info">
                <div class="member-name">{{ member.name }}</div>
                <div class="member-title" v-if="member.title">{{ member.title }}</div>
              </div>
            </div>
          </template>

          <!-- 普通成员分组 -->
          <template v-if="filteredNormalMembers.length > 0">
            <div class="member-group-header">
              <span class="member-icon">👤</span> 成员 — {{ filteredNormalMembers.length }}
            </div>
            <div
              v-for="member in filteredNormalMembers"
              :key="member.id"
              class="member-item"
              @click="onMemberClick(member)"
            >
              <div class="member-avatar">
                <img :src="member.avatar" @error="handleMemberAvatarError" />
              </div>
              <div class="member-info">
                <div class="member-name">{{ member.name }}</div>
                <div class="member-title" v-if="member.title">{{ member.title }}</div>
              </div>
            </div>
          </template>

          <!-- 无搜索结果 -->
          <div v-if="memberSearch && filteredMembers.length === 0" class="no-members">
            未找到匹配的成员
          </div>
        </div>

        <!-- 加载中 -->
        <div class="members-loading" v-else>
          <k-icon name="loader" class="spin" />
          <span>加载中...</span>
        </div>
      </template>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <div class="context-menu-item" @click="handleReply">
          <span class="menu-icon">↩️</span>
          <span>回复</span>
        </div>
        <div class="context-menu-item" @click="handleAt">
          <span class="menu-icon">@</span>
          <span>@TA</span>
        </div>
        <div class="context-menu-item" @click="handleCopy">
          <span class="menu-icon">📋</span>
          <span>复制</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item" @click="handleForward">
          <span class="menu-icon">📤</span>
          <span>转发</span>
        </div>
        <div class="context-menu-item danger" @click="handleRecall" v-if="canRecall">
          <span class="menu-icon">🗑️</span>
          <span>撤回</span>
        </div>
      </div>
      <div v-if="contextMenu.visible" class="context-menu-overlay" @click="hideContextMenu"></div>
    </Teleport>

    <!-- 连接群聊对话框 -->
    <div class="connect-dialog-overlay" v-if="showConnectDialog" @click.self="showConnectDialog = false">
      <div class="connect-dialog">
        <div class="dialog-header">
          <h3>新建会话</h3>
          <button class="close-btn" @click="showConnectDialog = false">×</button>
        </div>
        <div class="dialog-body">
          <div class="form-group">
            <label>会话类型</label>
            <div class="radio-group">
              <label class="radio-label">
                <input type="radio" v-model="connectForm.type" value="group" />
                群聊
              </label>
              <label class="radio-label">
                <input type="radio" v-model="connectForm.type" value="private" />
                私聊
              </label>
            </div>
          </div>
          <div class="form-group">
            <label>{{ connectForm.type === 'group' ? '群号 / 频道ID' : '用户ID' }}</label>
            <input
              v-model="connectForm.targetId"
              type="text"
              :placeholder="connectForm.type === 'group' ? '输入群号' : '输入QQ号/用户ID'"
              @keydown.enter="connectToChat"
            />
          </div>
          <div class="form-group">
            <label>显示名称 (可选)</label>
            <input
              v-model="connectForm.name"
              type="text"
              placeholder="自定义显示名称"
            />
          </div>
          <div class="form-group">
            <label>平台</label>
            <select v-model="connectForm.platform">
              <option value="onebot">OneBot</option>
              <option value="qq">QQ</option>
              <option value="red">Red</option>
            </select>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="cancel-btn" @click="showConnectDialog = false">取消</button>
          <button class="confirm-btn" @click="connectToChat" :disabled="!connectForm.targetId.trim()">
            连接
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { receive, message } from '@koishijs/client'
import { chatApi, GuildMember } from '../api'
import type { ChatMessage } from '../types'
import ChatMessageContent from './chat/ChatMessageContent.vue'

interface Session {
  key: string
  id: string // channelId
  type: 'group' | 'private'
  name: string
  platform: string
  guildId?: string
  avatar?: string
  messages: ChatMessage[]
  lastMessage?: ChatMessage
  unread: number
}

const sessions = ref<Session[]>([])
const currentSessionId = ref<string>('')
const inputText = ref('')
const sending = ref(false)
const messageListRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLTextAreaElement | null>(null)
const showConnectDialog = ref(false)

// 待发送的图片列表
interface PendingImage {
  dataUrl: string
  file: File
}
const pendingImages = ref<PendingImage[]>([])
const connectForm = reactive({
  type: 'group' as 'group' | 'private',
  targetId: '',
  name: '',
  platform: 'onebot'
})

const buildSessionKey = (params: {
  platform: string
  type: Session['type']
  channelId: string
  guildId?: string
}) => {
  const guildPart = params.guildId?.trim() || '-'
  return `${params.platform}:${params.type}:${guildPart}:${params.channelId}`
}

const findSessionByKey = (key: string) => {
  return sessions.value.find(session => session.key === key)
}

// 群成员相关
const members = ref<GuildMember[]>([])
const loadingMembers = ref(false)
const membersSidebarCollapsed = ref(false)
const memberSearch = ref('')

// 过滤后的成员列表
const filteredMembers = computed(() => {
  if (!memberSearch.value) return members.value
  const search = memberSearch.value.toLowerCase()
  return members.value.filter(m =>
    m.name?.toLowerCase().includes(search) ||
    m.id?.toLowerCase().includes(search) ||
    m.title?.toLowerCase().includes(search)
  )
})

// 分组成员
const filteredOwners = computed(() => filteredMembers.value.filter(m => m.isOwner))
const filteredAdmins = computed(() => filteredMembers.value.filter(m => m.isAdmin && !m.isOwner))
const filteredNormalMembers = computed(() => filteredMembers.value.filter(m => !m.isAdmin && !m.isOwner))

// 加载群成员
const loadGuildMembers = async (guildId: string) => {
  loadingMembers.value = true
  members.value = []
  
  try {
    const result = await chatApi.getGuildMembers(guildId)
    members.value = result.members || []
  } catch (e) {
    console.warn('Failed to load guild members:', e)
  } finally {
    loadingMembers.value = false
  }
}

// 处理成员头像加载错误
const handleMemberAvatarError = (e: Event) => {
  const img = e.target as HTMLImageElement
  img.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="%23999"%3E%3Cpath d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/%3E%3C/svg%3E'
}

// 点击成员
const onMemberClick = (member: GuildMember) => {
  // 使用 Koishi/OneBot 标准格式：<at id="用户ID" />
  const atElement = `<at id="${member.id}" />`
  if (inputText.value) {
    inputText.value += ` ${atElement} `
  } else {
    inputText.value = `${atElement} `
  }
}

// 右键菜单相关
const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  targetMsg: null as ChatMessage | null
})

// 计算是否可以撤回（只有自己发的消息可以撤回）
const canRecall = computed(() => {
  if (!contextMenu.targetMsg) return false
  return contextMenu.targetMsg.userId === contextMenu.targetMsg.selfId
})

const showContextMenu = (e: MouseEvent, msg: ChatMessage) => {
  contextMenu.visible = true
  contextMenu.x = e.clientX
  contextMenu.y = e.clientY
  contextMenu.targetMsg = msg
}

const hideContextMenu = () => {
  contextMenu.visible = false
  contextMenu.targetMsg = null
}

// 回复消息
const handleReply = () => {
  if (!contextMenu.targetMsg) return
  const msg = contextMenu.targetMsg
  // 使用 Koishi/OneBot 标准格式：<quote id="消息ID" />
  const quoteElement = `<quote id="${msg.id}" />`
  inputText.value = quoteElement + inputText.value
  hideContextMenu()
}

// @某人
const handleAt = () => {
  if (!contextMenu.targetMsg) return
  const msg = contextMenu.targetMsg
  // 使用 Koishi/OneBot 标准格式：<at id="用户ID" />
  const atElement = `<at id="${msg.userId}" />`
  if (inputText.value) {
    inputText.value += ` ${atElement} `
  } else {
    inputText.value = `${atElement} `
  }
  hideContextMenu()
}

// 复制消息内容
const handleCopy = async () => {
  if (!contextMenu.targetMsg) return
  const msg = contextMenu.targetMsg
  const text = msg.content || ''
  
  // 移除 HTML 标签，获取纯文本
  const tempDiv = document.createElement('div')
  tempDiv.innerHTML = text
  const plainText = tempDiv.textContent || tempDiv.innerText || ''
  
  try {
    await navigator.clipboard.writeText(plainText)
    message.success('已复制到剪贴板')
  } catch {
    // 回退方案
    const textarea = document.createElement('textarea')
    textarea.value = plainText
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    message.success('已复制到剪贴板')
  }
  hideContextMenu()
}

// 转发消息（暂时只是复制到输入框）
const handleForward = () => {
  if (!contextMenu.targetMsg) return
  const msg = contextMenu.targetMsg
  inputText.value = msg.content || ''
  hideContextMenu()
  message.info('消息已复制到输入框，选择其他会话发送即可转发')
}

// 处理粘贴事件
const handlePaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items
  if (!items) return

  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      if (file) {
        const reader = new FileReader()
        reader.onload = (event) => {
          const dataUrl = event.target?.result as string
          if (dataUrl) {
            pendingImages.value.push({ dataUrl, file })
          }
        }
        reader.readAsDataURL(file)
      }
    }
  }
}

// 移除待发送图片
const removePendingImage = (index: number) => {
  pendingImages.value.splice(index, 1)
}

// 撤回消息
const handleRecall = async () => {
  if (!contextMenu.targetMsg || !currentSession.value) return
  const msg = contextMenu.targetMsg
  const session = currentSession.value
  
  try {
    await chatApi.recall(session.id, msg.id, session.platform, session.guildId)
    // 从本地消息列表中移除
    const index = session.messages.findIndex(m => m.id === msg.id)
    if (index !== -1) {
      session.messages.splice(index, 1)
    }
    message.success('消息已撤回')
  } catch (e: any) {
    message.error(e.message || '撤回失败')
  }
  hideContextMenu()
}

// 辅助函数：判断是否是自己发送的消息
const isSelf = (msg: ChatMessage) => {
  return msg.userId === msg.selfId
}

// 接收消息监听
onMounted(() => {
  receive('stuhelperGroupCenter/chat/message', (data: ChatMessage) => {
    handleIncomingMessage(data)
  })
})

const handleIncomingMessage = async (msg: ChatMessage) => {
  const sessionType: Session['type'] = msg.guildId ? 'group' : 'private'
  const sessionKey = buildSessionKey({
    platform: msg.platform,
    type: sessionType,
    channelId: msg.channelId,
    guildId: msg.guildId
  })

  let session = findSessionByKey(sessionKey)

  if (!session) {
    // 新会话 - 先创建基础会话
    const isGroup = sessionType === 'group'
    let displayName = msg.channelName || msg.guildName || (isGroup ? `群聊 ${msg.channelId}` : `私聊 ${msg.username}`)
    let displayAvatar = isGroup ? msg.guildAvatar : msg.avatar

    session = {
      key: sessionKey,
      id: msg.channelId,
      type: sessionType,
      name: displayName,
      platform: msg.platform,
      guildId: msg.guildId,
      avatar: displayAvatar,
      messages: [],
      unread: 0
    }
    sessions.value.unshift(session)

    // 异步获取群名/用户名（不阻塞消息处理）
    if (isGroup && msg.guildId && !msg.guildName) {
      chatApi.getGuildInfo(msg.guildId).then(info => {
        if (info?.name) session!.name = info.name
        if (info?.avatar && !session!.avatar) session!.avatar = info.avatar
      }).catch(() => {})
    } else if (!isGroup && !msg.username) {
      chatApi.getUserInfo(msg.userId).then(info => {
        if (info?.name) session!.name = `私聊 ${info.name}`
        if (info?.avatar && !session!.avatar) session!.avatar = info.avatar
      }).catch(() => {})
    }
  } else {
    // 移到顶部
    const index = sessions.value.indexOf(session)
    if (index > 0) {
      sessions.value.splice(index, 1)
      sessions.value.unshift(session)
    }
  }

  // Update avatar if available and missing
  if (!session.avatar) {
    if (session.type === 'group' && msg.guildAvatar) session.avatar = msg.guildAvatar
    if (session.type === 'private' && msg.avatar) session.avatar = msg.avatar
  }

  session.messages.push(msg)
  session.lastMessage = msg
  
  // 如果不是当前会话，增加未读
  if (currentSessionId.value !== sessionKey) {
    session.unread++
  } else {
    scrollToBottom()
  }
}

const currentSession = ref<Session | undefined>(undefined)

watch(currentSessionId, (newId) => {
  const session = findSessionByKey(newId)
  if (session) {
    session.unread = 0
    currentSession.value = session
    nextTick(() => scrollToBottom())
    
    // 如果是群聊，加载群成员
    if (session.type === 'group' && session.guildId) {
      loadGuildMembers(session.guildId)
    } else {
      members.value = []
    }
  } else {
    currentSession.value = undefined
    members.value = []
  }
})

const selectSession = (id: string) => {
  currentSessionId.value = id
}

// 连接到会话
const connectToChat = async () => {
  const targetId = connectForm.targetId.trim()
  if (!targetId) return
  
  let displayName = connectForm.name.trim()
  const isGroup = connectForm.type === 'group'
  
  // 如果名称为空，尝试自动获取名称
  if (!displayName) {
    try {
      if (isGroup) {
        const info = await chatApi.getGuildInfo(targetId)
        if (info?.name) displayName = info.name
      } else {
        const info = await chatApi.getUserInfo(targetId)
        if (info?.name) displayName = info.name
      }
    } catch (e) {
      console.warn('Failed to fetch info:', e)
    }
  }
  
  // 如果仍然没有名称，使用默认名称
  if (!displayName) {
    displayName = isGroup ? `群聊 ${targetId}` : `私聊 ${targetId}`
  }

  const sessionKey = buildSessionKey({
    platform: connectForm.platform,
    type: connectForm.type,
    channelId: targetId,
    guildId: isGroup ? targetId : undefined
  })
  
  // 检查是否已存在该会话
  let session = findSessionByKey(sessionKey)
  
  if (!session) {
    // 创建新会话
    session = {
      key: sessionKey,
      id: targetId,
      type: connectForm.type,
      name: displayName,
      platform: connectForm.platform,
      guildId: isGroup ? targetId : undefined,
      avatar: isGroup
        ? `https://p.qlogo.cn/gh/${targetId}/${targetId}/640/`
        : `https://q1.qlogo.cn/g?b=qq&nk=${targetId}&s=640`,
      messages: [],
      unread: 0
    }
    sessions.value.unshift(session)
  } else {
    // 如果获取到了新名称，更新现有会话
    if (displayName && displayName !== session.name) {
      session.name = displayName
    }
  }
  
  // 选中该会话
  currentSessionId.value = sessionKey
  
  // 关闭对话框并重置表单
  showConnectDialog.value = false
  connectForm.targetId = ''
  connectForm.name = ''
  connectForm.type = 'group'
}

const scrollToBottom = () => {
  if (messageListRef.value) {
    messageListRef.value.scrollTop = messageListRef.value.scrollHeight
  }
}

const sendMessage = async () => {
  const text = inputText.value.trim()
  const hasImages = pendingImages.value.length > 0
  
  if (!text && !hasImages) return
  if (!currentSession.value) return

  sending.value = true
  try {
    const session = currentSession.value
    
    // 构建消息内容
    let content = text
    
    // 添加图片（使用 base64 格式）
    for (const img of pendingImages.value) {
      // 使用 Koishi 的 img 元素格式，src 为 base64 dataUrl
      content += `<img src="${img.dataUrl}" />`
    }
    
    await chatApi.send(session.id, content, session.platform, session.guildId)
    
    // 清空输入框和待发送图片
    inputText.value = ''
    pendingImages.value = []
  } catch (e: any) {
    message.error(e.message || '发送失败')
  } finally {
    sending.value = false
    // 聚焦回输入框
    inputRef.value?.focus()
  }
}

const formatTimeShort = (ts?: number) => {
  if (!ts) return ''
  const date = new Date(ts)
  const now = new Date()
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

const formatTimeDetail = (ts: number) => {
  return new Date(ts).toLocaleString('zh-CN')
}

const handleAvatarError = (e: Event, isSession = false) => {
  const img = e.target as HTMLImageElement
  img.style.display = 'none'
  if (isSession) {
    // Session icon fallback logic handled in template by v-if/v-else check?
    // Actually if src errors, we might want to show the k-icon.
    // But since v-else is not tied to @error, we need to manipulate DOM or state.
    // Simple way: hide img, show sibling if it exists? No sibling in template for fallback.
    // Better: set src to null? modifying prop is bad. modifying state?
    // Let's just let it hide.
  }
}
</script>

<style scoped>
/* ============================================
   GitHub Dimmed / Vercel Dark Style
   去 AI 味：专业、硬核、高信噪比
   ============================================ */

/* Font: Inter + Monospace for numbers */
.chat-view {
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', 'SF Mono', Consolas, monospace;
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  /* Muted status colors */
  --status-online: #2ea043;
  --status-offline: #6e7681;
  --status-warning: #d29922;
  --status-danger: #f85149;
}

.chat-view {
  height: 100%;
  display: flex;
  background: var(--bg1);
  border: 1px solid var(--k-color-divider);
  border-radius: var(--radius-lg);
  overflow: hidden;
  font-family: var(--font-sans);
}

/* Sidebar */
.chat-sidebar {
  width: 240px;
  border-right: 1px solid var(--k-color-divider);
  display: flex;
  flex-direction: column;
  background: var(--bg2);
}

.sidebar-header {
  padding: 12px 14px;
  border-bottom: 1px solid var(--k-color-divider);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--fg1);
  letter-spacing: -0.01em;
}

.status-indicator {
  font-size: 11px;
  color: var(--k-color-success);
  display: flex;
  align-items: center;
  gap: 5px;
  font-family: var(--font-mono);
  font-weight: 500;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--k-color-success);
  /* 实心小圆点，无发光效果 */
}

/* Session List */
.session-list {
  flex: 1;
  overflow-y: auto;
}

.empty-sessions {
  padding: 32px 16px;
  text-align: center;
  color: var(--fg3);
  font-size: 12px;
}

.session-item {
  display: flex;
  padding: 10px 12px;
  gap: 10px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  border-left: 2px solid transparent;
}

.session-item:hover {
  background: var(--bg3);
}

.session-item.active {
  background: var(--k-color-primary-fade);
  border-left-color: var(--k-color-primary);
}

.session-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--bg3);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--fg3);
  flex-shrink: 0;
  overflow: hidden;
  border: 1px solid var(--k-color-divider);
}

.session-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.session-info {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
}

.session-name {
  font-weight: 500;
  color: var(--fg1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;
}

.session-preview {
  font-size: 11px;
  color: var(--fg3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}

.session-meta .time {
  font-size: 10px;
  color: var(--fg3);
  font-family: var(--font-mono);
}

.badge {
  background: var(--k-color-danger);
  color: #fff;
  font-size: 10px;
  font-family: var(--font-mono);
  font-weight: 600;
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  min-width: 16px;
  text-align: center;
}

/* Main Chat Area */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg1);
}

.empty-chat {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--fg3);
}

.empty-content {
  text-align: center;
}

.empty-content h3 {
  margin: 12px 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--fg2);
}

.empty-content p {
  margin: 0;
  font-size: 12px;
  color: var(--fg3);
}

.large-icon {
  font-size: 40px;
  opacity: 0.3;
}

.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* Chat Header */
.chat-header {
  padding: 10px 16px;
  border-bottom: 1px solid var(--k-color-divider);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg2);
}

.header-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--bg3);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--k-color-divider);
}

.header-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.header-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--fg1);
}

.header-id {
  font-size: 11px;
  color: var(--fg3);
  font-family: var(--font-mono);
  background: var(--bg3);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.platform-tag {
  background: var(--bg3);
  color: var(--fg2);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-family: var(--font-mono);
  font-weight: 500;
  text-transform: uppercase;
  border: 1px solid var(--k-color-divider);
}

/* Message List */
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-row {
  display: flex;
  gap: 10px;
  max-width: 80%;
}

.message-row.self {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.message-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
}

.message-avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--k-color-divider);
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: var(--k-color-primary, #7459ff);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 13px;
}

.message-content-wrapper {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.message-row.self .message-content-wrapper {
  align-items: flex-end;
}

.message-meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: var(--fg3);
  align-items: baseline;
}

.message-row.self .message-meta {
  flex-direction: row-reverse;
}

.message-row.self :deep(.msg-at) {
  color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.15);
}

.username {
  font-weight: 500;
  color: var(--fg2);
}

.timestamp {
  font-family: var(--font-mono);
  font-size: 10px;
}

.message-bubble {
  background: var(--bg3);
  padding: 10px 12px;
  border-radius: var(--radius-md);
  color: var(--fg1);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  border: 1px solid var(--k-color-divider);
  width: fit-content;
  max-width: 100%;
  font-size: 13px;
}

/* Deep selector required for v-html content in scoped css */
.message-bubble :deep(.msg-img) {
  max-width: 180px;
  max-height: 180px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  margin: 4px 0;
  display: block;
  border: 1px solid var(--k-color-divider);
}

.message-bubble :deep(.msg-img.loading) {
  width: 80px;
  height: 80px;
  background: linear-gradient(90deg, var(--bg3) 25%, var(--bg2) 50%, var(--bg3) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border: 1px dashed var(--k-color-divider);
}

.message-bubble :deep(.msg-img.error) {
  width: 80px;
  height: 50px;
  background: var(--bg3);
  border: 1px dashed var(--k-color-danger);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--k-color-danger);
  font-size: 10px;
}

.message-bubble :deep(.msg-img.error)::before {
  content: '图片已过期';
  display: block;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.message-bubble :deep(.msg-at) {
  color: var(--k-color-primary);
  background: var(--k-color-primary-fade);
  padding: 1px 4px;
  border-radius: var(--radius-sm);
  margin: 0 1px;
  font-weight: 500;
  font-size: 12px;
  display: inline-block;
}

.message-bubble :deep(.msg-face) {
  display: inline-block;
  color: var(--fg3);
  font-size: 12px;
}

.message-bubble :deep(.msg-quote) {
  background: var(--bg2);
  border-left: 2px solid var(--k-color-primary);
  padding: 6px 10px;
  margin-bottom: 6px;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  font-size: 11px;
  color: var(--fg3);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.message-bubble :deep(.msg-quote .quote-user) {
  font-weight: 600;
  color: var(--k-color-primary);
  font-size: 10px;
}

.message-bubble :deep(.msg-quote .quote-content) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 260px;
}

.message-row.self .message-bubble :deep(.msg-quote) {
  background: rgba(255, 255, 255, 0.1);
  border-left-color: rgba(255, 255, 255, 0.4);
}

.message-row.self .message-bubble :deep(.msg-quote .quote-user) {
  color: rgba(255, 255, 255, 0.9);
}

.message-row.self .message-bubble :deep(.msg-quote .quote-content) {
  color: rgba(255, 255, 255, 0.7);
}

.message-row.self .message-bubble {
  background: var(--k-color-primary);
  color: #fff;
  border-color: transparent;
}

/* Input Area */
.chat-input-area {
  padding: 12px 16px;
  background: var(--bg2);
  border-top: 1px solid var(--k-color-divider);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pending-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.pending-image-item {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--k-color-divider);
}

.pending-image-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.remove-image-btn {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 16px;
  height: 16px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.7);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  line-height: 1;
  transition: background 0.15s;
}

.remove-image-btn:hover {
  background: var(--k-color-danger);
}

.input-row {
  display: flex;
  gap: 10px;
  align-items: flex-end;
}

.chat-input {
  flex: 1;
  height: 52px;
  padding: 10px 12px;
  border: 1px solid var(--k-color-divider);
  border-radius: var(--radius-md);
  background: var(--bg1);
  color: var(--fg1);
  resize: none;
  font-family: var(--font-sans);
  font-size: 13px;
}

.chat-input:focus {
  outline: none;
  border-color: var(--k-color-primary);
}

.chat-input::placeholder {
  color: var(--fg3);
}

.send-btn {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--k-color-primary);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.15s ease;
}

.send-btn:hover {
  opacity: 0.85;
}

.send-btn:disabled {
  background: var(--bg3);
  color: var(--fg3);
  cursor: not-allowed;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Scrollbar - 简洁细窄 */
::-webkit-scrollbar {
  width: 5px;
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

/* Connect Group Button */
.connect-group-bar {
  padding: 10px 12px;
  border-bottom: 1px solid var(--k-color-divider);
}

.connect-btn {
  width: 100%;
  padding: 8px 12px;
  border: 1px dashed var(--k-color-divider);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--fg3);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  font-family: var(--font-sans);
  transition: all 0.15s ease;
}

.connect-btn:hover {
  border-color: var(--k-color-primary);
  color: var(--k-color-primary);
  background: var(--k-color-primary-fade);
}

/* Connect Dialog */
.connect-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.connect-dialog {
  background: var(--bg2);
  border-radius: var(--radius-lg);
  width: 360px;
  max-width: 90%;
  border: 1px solid var(--k-color-divider);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.dialog-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--k-color-divider);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.dialog-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--fg1);
}

.close-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--fg3);
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: background-color 0.15s ease;
}

.close-btn:hover {
  background: var(--bg3);
  color: var(--fg1);
}

.dialog-body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 11px;
  color: var(--fg2);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.form-group input,
.form-group select {
  padding: 8px 10px;
  border: 1px solid var(--k-color-divider);
  border-radius: var(--radius-md);
  background: var(--bg1);
  color: var(--fg1);
  font-size: 13px;
  font-family: var(--font-sans);
}

.form-group select {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%236e7681' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  padding-right: 28px;
}

.form-group select option {
  background: var(--bg1);
  color: var(--fg1);
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--k-color-primary);
}

.form-group input::placeholder {
  color: var(--fg3);
}

.radio-group {
  display: flex;
  gap: 16px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: var(--fg2);
  font-size: 13px;
}

.radio-label input[type="radio"] {
  accent-color: var(--k-color-primary);
}

.dialog-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--k-color-divider);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.cancel-btn,
.confirm-btn {
  padding: 6px 14px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  font-family: var(--font-sans);
}

.cancel-btn {
  border: 1px solid var(--k-color-divider);
  background: transparent;
  color: var(--fg2);
}

.cancel-btn:hover {
  background: var(--bg3);
  color: var(--fg1);
}

.confirm-btn {
  border: none;
  background: var(--k-color-primary);
  color: #fff;
}

.confirm-btn:hover {
  opacity: 0.85;
}

.confirm-btn:disabled {
  background: var(--bg3);
  color: var(--fg3);
  cursor: not-allowed;
}

/* Members Sidebar */
.members-sidebar {
  width: 200px;
  border-left: 1px solid var(--k-color-divider);
  display: flex;
  flex-direction: column;
  background: var(--bg2);
  transition: width 0.2s ease;
}

.members-sidebar.collapsed {
  width: 36px;
}

.members-header {
  padding: 10px 12px;
  border-bottom: 1px solid var(--k-color-divider);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.members-title {
  display: flex;
  align-items: center;
  gap: 6px;
}

.members-sidebar.collapsed .members-title {
  display: none;
}

.members-header h3 {
  margin: 0;
  font-size: 11px;
  color: var(--fg2);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.member-count {
  font-size: 10px;
  font-family: var(--font-mono);
  background: var(--bg3);
  color: var(--fg2);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--k-color-divider);
}

.collapse-btn {
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: var(--fg3);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  border-radius: var(--radius-sm);
  transition: background-color 0.15s ease;
}

.collapse-btn:hover {
  background: var(--bg3);
  color: var(--fg1);
}

.members-search {
  padding: 8px 10px;
  border-bottom: 1px solid var(--k-color-divider);
}

.members-search .search-input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--k-color-divider);
  border-radius: var(--radius-md);
  background: var(--bg1);
  color: var(--fg1);
  font-size: 11px;
  font-family: var(--font-sans);
}

.members-search .search-input:focus {
  outline: none;
  border-color: var(--k-color-primary);
}

.members-search .search-input::placeholder {
  color: var(--fg3);
}

.members-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px 0;
}

.member-group-header {
  padding: 8px 10px 4px;
  font-size: 10px;
  color: var(--fg3);
  font-weight: 600;
  text-transform: uppercase;
  display: flex;
  align-items: center;
  gap: 4px;
  letter-spacing: 0.02em;
}

.crown-icon,
.admin-icon,
.member-icon {
  font-size: 10px;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.member-item:hover {
  background: var(--bg3);
}

.member-item.owner .member-name {
  color: var(--k-color-warning);
  font-weight: 600;
}

.member-item.admin .member-name {
  color: var(--k-color-success);
  font-weight: 500;
}

.member-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--bg3);
  border: 1px solid var(--k-color-divider);
}

.member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-info {
  flex: 1;
  overflow: hidden;
}

.member-name {
  font-size: 12px;
  color: var(--fg1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-title {
  font-size: 10px;
  color: var(--fg3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 1px;
}

.members-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  gap: 6px;
  color: var(--fg3);
  font-size: 11px;
}

.no-members {
  padding: 24px;
  text-align: center;
  color: var(--fg3);
  font-size: 11px;
}

/* Responsive */
@media (max-width: 900px) {
  .members-sidebar {
    display: none;
  }
}

/* Context Menu */
.context-menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
}

.context-menu {
  position: fixed;
  z-index: 10000;
  background: var(--bg2);
  border: 1px solid var(--k-color-divider);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
  min-width: 130px;
  padding: 4px 0;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  cursor: pointer;
  color: var(--fg1);
  font-size: 12px;
  transition: background-color 0.1s ease;
}

.context-menu-item:hover {
  background: var(--bg3);
}

.context-menu-item.danger {
  color: var(--k-color-danger);
}

.context-menu-item.danger:hover {
  background: var(--k-color-danger-fade);
}

.menu-icon {
  font-size: 12px;
  width: 16px;
  text-align: center;
  opacity: 0.7;
}

.context-menu-divider {
  height: 1px;
  background: var(--k-color-divider);
  margin: 4px 0;
}

/* ========== Mobile Responsive ========== */
@media (max-width: 768px) {
  .chat-view {
    flex-direction: column;
  }

  /* Sidebar becomes horizontal session list */
  .chat-sidebar {
    width: 100%;
    height: auto;
    max-height: 180px;
    border-right: none;
    border-bottom: 1px solid var(--k-color-divider);
    flex-shrink: 0;
  }

  .sidebar-header {
    padding: 10px 12px;
  }

  .sidebar-header h3 {
    font-size: 14px;
  }

  .status-indicator {
    font-size: 11px;
  }

  .connect-group-bar {
    padding: 8px 12px;
  }

  .connect-btn {
    padding: 6px 12px;
    font-size: 12px;
  }

  .session-list {
    display: flex;
    flex-direction: row;
    overflow-x: auto;
    overflow-y: hidden;
    padding: 8px;
    gap: 8px;
    -webkit-overflow-scrolling: touch;
  }

  .session-item {
    flex-shrink: 0;
    min-width: 140px;
    max-width: 180px;
    padding: 8px 10px;
  }

  .session-icon {
    width: 28px;
    height: 28px;
  }

  .session-name {
    font-size: 12px;
  }

  .session-preview {
    font-size: 11px;
  }

  .session-meta {
    display: none;
  }

  /* Chat main area */
  .chat-main {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .chat-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .chat-header {
    padding: 10px 12px;
    flex-shrink: 0;
  }

  .header-icon {
    width: 28px;
    height: 28px;
  }

  .header-name {
    font-size: 13px;
  }

  .header-id {
    font-size: 11px;
  }

  .platform-tag {
    font-size: 10px;
    padding: 2px 6px;
  }

  .message-list {
    flex: 1;
    padding: 12px;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  .message-row {
    gap: 8px;
  }

  .message-avatar {
    width: 32px;
    height: 32px;
    flex-shrink: 0;
  }

  .avatar-placeholder {
    width: 32px;
    height: 32px;
    font-size: 12px;
  }

  .username {
    font-size: 12px;
  }

  .timestamp {
    font-size: 10px;
  }

  .message-bubble {
    font-size: 13px;
    padding: 8px 10px;
    max-width: 85%;
  }

  /* Input area */
  .chat-input-area {
    flex-shrink: 0;
    padding: 8px 12px;
    border-top: 1px solid var(--k-color-divider);
  }

  .input-row {
    gap: 8px;
  }

  .input-row textarea {
    font-size: 16px; /* Prevent iOS zoom */
    min-height: 36px;
    max-height: 100px;
    padding: 8px 10px;
  }

  .send-btn {
    padding: 8px 16px;
    font-size: 12px;
  }

  .pending-images {
    gap: 6px;
  }

  .pending-image-item {
    width: 50px;
    height: 50px;
  }

  /* Members sidebar - hidden on mobile */
  .members-sidebar {
    display: none;
  }

  /* Connect dialog */
  .connect-dialog {
    width: 95%;
    max-width: none;
    margin: 16px;
  }

  .dialog-header {
    padding: 12px 16px;
  }

  .dialog-header h3 {
    font-size: 1rem;
  }

  .dialog-body {
    padding: 16px;
    max-height: 60vh;
    overflow-y: auto;
  }

  .form-group {
    gap: 6px;
  }

  .form-group label {
    font-size: 12px;
  }

  .form-group input,
  .form-group select {
    font-size: 16px;
    padding: 10px 12px;
  }

  .dialog-footer {
    padding: 12px 16px;
  }

  /* Context menu */
  .context-menu {
    min-width: 120px;
  }

  .context-menu-item {
    padding: 10px 12px;
    font-size: 13px;
  }

  /* Empty session state */
  .empty-state {
    padding: 40px 20px;
  }

  .empty-icon {
    font-size: 48px;
  }

  .empty-state p {
    font-size: 13px;
  }
}

@media (max-width: 480px) {
  .chat-sidebar {
    max-height: 150px;
  }

  .session-item {
    min-width: 120px;
    padding: 6px 8px;
  }

  .session-icon {
    width: 24px;
    height: 24px;
  }

  .session-info {
    gap: 2px;
  }

  .message-avatar {
    width: 28px;
    height: 28px;
  }

  .avatar-placeholder {
    width: 28px;
    height: 28px;
    font-size: 11px;
  }

  .message-bubble {
    font-size: 12px;
    padding: 6px 8px;
    max-width: 90%;
  }

  .chat-input-area {
    padding: 6px 8px;
  }

  .send-btn {
    padding: 8px 12px;
  }

  .pending-image-item {
    width: 40px;
    height: 40px;
  }
}
</style>
