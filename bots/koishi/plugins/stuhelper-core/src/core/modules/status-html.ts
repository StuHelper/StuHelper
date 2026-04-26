import { formatDuration } from '../../utils'
import type { StatusData } from './status-data'

const STATUS_STYLE = `
      @import url('https://fonts.googleapis.com/css2?family=Roboto:wght@400;500;700&display=swap');

      :root {
        --bg-color: #ffffff;
        --card-bg: #f8f9fa;
        --text-primary: #212529;
        --text-secondary: #868e96;
        --accent-color: #228be6;
        --border-color: #e9ecef;
      }

      body {
        margin: 0;
        padding: 20px;
        background: transparent;
        font-family: 'Roboto', 'Segoe UI', sans-serif;
        color: var(--text-primary);
        width: 800px;
      }

      .container {
        background: var(--bg-color);
        border-radius: 12px;
        padding: 32px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
        border: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        gap: 24px;
      }

      .header {
        display: flex;
        align-items: center;
        padding-bottom: 20px;
        border-bottom: 1px solid var(--border-color);
      }

      .logo {
        width: 48px;
        height: 48px;
        background: var(--accent-color);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20px;
        font-weight: bold;
        color: white;
        margin-right: 16px;
      }

      .title h1 {
        margin: 0;
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary);
      }

      .title p {
        margin: 2px 0 0;
        color: var(--text-secondary);
        font-size: 14px;
      }

      .main-stats {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 24px;
      }

      .chart-card {
        background: var(--card-bg);
        border-radius: 8px;
        padding: 24px;
        display: flex;
        align-items: center;
        gap: 24px;
        border: 1px solid var(--border-color);
      }

      .chart-info {
        flex: 1;
      }

      .chart-title {
        color: var(--text-secondary);
        font-size: 12px;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin-bottom: 8px;
      }

      .chart-value {
        font-size: 32px;
        font-weight: 500;
        color: var(--text-primary);
        margin-bottom: 4px;
      }

      .chart-sub {
        font-size: 13px;
        color: var(--text-secondary);
      }

      .circle-chart {
        width: 80px;
        height: 80px;
      }

      .circle-bg {
        fill: none;
        stroke: #dee2e6;
        stroke-width: 4;
      }

      .circle {
        fill: none;
        stroke: var(--accent-color);
        stroke-width: 4;
        stroke-linecap: round;
      }

      .grid-stats {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 24px;
      }

      .stat-column {
        display: flex;
        flex-direction: column;
        gap: 16px;
      }

      .stat-header {
        font-size: 14px;
        font-weight: 700;
        color: var(--text-primary);
        padding-bottom: 8px;
        border-bottom: 2px solid var(--accent-color);
        display: inline-block;
        margin-bottom: 8px;
      }

      .stat-item {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
        font-size: 14px;
      }

      .stat-key {
        color: var(--text-secondary);
      }

      .stat-val {
        font-weight: 500;
        color: var(--text-primary);
      }

      .footer {
        text-align: right;
        font-size: 12px;
        color: var(--text-secondary);
        margin-top: 8px;
        padding-top: 16px;
        border-top: 1px solid var(--border-color);
      }
    `

export function renderStatusHtml(data: StatusData): string {
  const memUsagePercent = (data.process.memory.rss / data.system.totalMem) * 100
  const sysMemUsagePercent = (data.system.usedMem / data.system.totalMem) * 100

  return `
      <!DOCTYPE html>
      <html>
      <head>
        <style>${STATUS_STYLE}</style>
      </head>
      <body>
        <div class="container">
          ${renderHeader()}
          ${renderMainStats(data, sysMemUsagePercent, memUsagePercent)}
          <div class="grid-stats">
            ${renderApplicationColumn(data)}
            ${renderSystemColumn(data)}
            ${renderStatisticsColumn(data)}
          </div>
          <div class="footer">
            Generated at ${new Date().toLocaleString('zh-CN')}
          </div>
        </div>
      </body>
      </html>
    `
}

function renderHeader(): string {
  return `<div class="header">
            <div class="logo">GH</div>
            <div class="title">
              <h1>System Status</h1>
              <p>StuHelper 群管中心 Monitor</p>
            </div>
          </div>`
}

function renderMainStats(data: StatusData, sysMemUsagePercent: number, memUsagePercent: number): string {
  return `<div class="main-stats">
            ${renderMemoryCard('System Memory', sysMemUsagePercent, `${formatBytes(data.system.usedMem)} / ${formatBytes(data.system.totalMem)}`)}
            ${renderMemoryCard('Process Memory', memUsagePercent, `${formatBytes(data.process.memory.rss)} Used`)}
          </div>`
}

function renderMemoryCard(title: string, percent: number, subText: string): string {
  return `<div class="chart-card">
              <div class="chart-info">
                <div class="chart-title">${title}</div>
                <div class="chart-value">${percent.toFixed(1)}%</div>
                <div class="chart-sub">${subText}</div>
              </div>
              ${renderCircle(percent)}
            </div>`
}

function renderApplicationColumn(data: StatusData): string {
  return `<div class="stat-column">
              <div class="stat-header">APPLICATION</div>
              ${renderStatItem('Koishi', `v${data.bot.version}`)}
              ${renderStatItem('StuHelper 群管中心', `v${data.stuhelperGroupCenter.version}`)}
              ${renderStatItem('Uptime', formatDuration(data.process.uptime * 1000))}
            </div>`
}

function renderSystemColumn(data: StatusData): string {
  return `<div class="stat-column">
              <div class="stat-header">SYSTEM</div>
              ${renderStatItem('OS', data.os.platform)}
              ${renderStatItem('Arch', data.os.arch)}
              ${renderStatItem('Load Avg', data.system.loadavg[0].toFixed(2))}
            </div>`
}

function renderStatisticsColumn(data: StatusData): string {
  return `<div class="stat-column">
              <div class="stat-header">STATISTICS</div>
              ${renderStatItem('Plugins', data.bot.plugins)}
              ${renderStatItem('Groups', data.stuhelperGroupCenter.groupCount)}
              ${renderStatItem('Logs', data.stuhelperGroupCenter.logCount)}
            </div>`
}

function renderStatItem(key: string, value: unknown): string {
  return `<div class="stat-item">
                 <span class="stat-key">${key}</span>
                 <span class="stat-val">${value}</span>
              </div>`
}

function renderCircle(percent: number): string {
  const radius = 18
  const circumference = radius * 2 * Math.PI
  const offset = circumference - (percent / 100) * circumference

  return `<svg class="circle-chart" viewBox="0 0 40 40">
                <path class="circle-bg" d="M20 2.0845 a 17.9155 17.9155 0 0 1 0 35.831 a 17.9155 17.9155 0 0 1 0 -35.831" />
                <path class="circle" stroke-dasharray="${circumference}, ${circumference}" stroke-dashoffset="${offset}" d="M20 2.0845 a 17.9155 17.9155 0 0 1 0 35.831 a 17.9155 17.9155 0 0 1 0 -35.831" />
            </svg>`
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}
