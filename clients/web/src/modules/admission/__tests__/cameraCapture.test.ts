// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  buildCameraConstraints,
  supportsCameraCapture,
} from '../cameraCapture'
import FreshmanCameraFlow from '../views/FreshmanCameraFlow.vue'

const mockAdmissionApi = vi.hoisted(() => ({
  submitFreshmanApplication: vi.fn(),
  uploadCameraCapture: vi.fn(),
}))

vi.mock('../api', () => ({
  admissionApi: mockAdmissionApi,
}))

describe('camera capture helpers', () => {
  it('reports unsupported when getUserMedia is unavailable', () => {
    expect(supportsCameraCapture({ mediaDevices: undefined } as Navigator))
      .toBe(false)
  })

  it('requests an environment-facing camera', () => {
    expect(buildCameraConstraints()).toEqual({
      audio: false,
      video: {
        facingMode: { ideal: 'environment' },
      },
    })
  })
})

describe('FreshmanCameraFlow material capture UI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders freshman camera flow without a file input', async () => {
    const wrapper = mount(FreshmanCameraFlow, {
      props: { maxMaterialBytes: 1024 },
    })
    await flushPromises()

    expect(wrapper.find('[data-admission-freshman-flow]').exists()).toBe(true)
    expect(wrapper.find('input[type="file"]').exists()).toBe(false)
  })
})
