import EmptyState from './EmptyState.vue'

const meta = {
  title: 'Common/EmptyState',
  component: EmptyState,
  args: {
    title: '暂无数据',
    description: '当前筛选条件下没有可展示的内容。'
  }
}

export default meta

export const Default = {}

export const WithAction = {
  render: (args: Record<string, unknown>) => ({
    components: { EmptyState },
    setup() {
      return { args }
    },
    template: `
      <EmptyState v-bind="args">
        <template #action>
          <button class="px-4 py-2 rounded-lg bg-primary text-white border-none cursor-pointer">
            重新加载
          </button>
        </template>
      </EmptyState>
    `
  })
}
