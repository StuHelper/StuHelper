import type { Preview } from '@storybook/vue3'
import '../src/styles/tailwind.css'

const preview: Preview = {
  parameters: {
    layout: 'centered',
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i
      }
    }
  },
  decorators: [
    story => ({
      components: { story },
      template: `
        <div class="min-h-screen min-w-[320px] bg-bg-base text-text-primary p-6">
          <story />
        </div>
      `
    })
  ]
}

export default preview
