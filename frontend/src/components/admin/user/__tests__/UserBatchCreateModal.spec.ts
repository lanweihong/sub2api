import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserBatchCreateModal from '../UserBatchCreateModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      previewBatch: vi.fn(),
      createBatch: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params?.count != null) {
        return `${key}:${params.count}`
      }
      return key
    }
  })
}))

function mountModal() {
  return mount(UserBatchCreateModal, {
    props: { show: true },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true
      }
    }
  })
}

function readBlobAsText(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read blob'))
    reader.readAsText(blob)
  })
}

describe('UserBatchCreateModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.users.previewBatch).mockReset()
    vi.mocked(adminAPI.users.createBatch).mockReset()
  })

  it('上传 txt 后调用预览接口并渲染结果', async () => {
    vi.mocked(adminAPI.users.previewBatch).mockResolvedValue({
      items: [
        {
          row_no: 1,
          source_name: '张三',
          username: 'zhangsan',
          email: 'zhangsan@xssio.com',
          password: 'pass1234',
          notes: '',
          balance: 9999,
          concurrency: 3,
          errors: []
        }
      ]
    })

    const wrapper = mountModal()
    const input = wrapper.get('input[type="file"]')
    const file = new File(['张三\n'], 'users.txt', { type: 'text/plain' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('张三\n')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await flushPromises()

    expect(adminAPI.users.previewBatch).toHaveBeenCalledWith(['张三'])
    const emailInput = wrapper.find('input[type="email"]')
    expect((emailInput.element as HTMLInputElement).value).toBe('zhangsan@xssio.com')
  })

  it('提交后返回行错误时保留弹窗并展示错误', async () => {
    vi.mocked(adminAPI.users.previewBatch).mockResolvedValue({
      items: [
        {
          row_no: 1,
          source_name: '张三',
          username: 'zhangsan',
          email: 'zhangsan@xssio.com',
          password: 'pass1234',
          notes: '',
          balance: 9999,
          concurrency: 3,
          errors: []
        }
      ]
    })
    vi.mocked(adminAPI.users.createBatch).mockResolvedValue({
      created_count: 0,
      failed_count: 1,
      users: [],
      errors: [
        {
          row_no: 1,
          field: 'email',
          code: 'EMAIL_EXISTS',
          message: 'email already exists'
        }
      ]
    })

    const wrapper = mountModal()
    const input = wrapper.get('input[type="file"]')
    const file = new File(['张三\n'], 'users.txt', { type: 'text/plain' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('张三\n')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await flushPromises()

    await wrapper.get('.btn.btn-primary').trigger('click')
    await flushPromises()

    expect(adminAPI.users.createBatch).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('admin.users.batchCreate.fixErrors')
    expect(wrapper.text()).toContain('email already exists')
    expect(wrapper.emitted('close')).toBeFalsy()
  })

  it('提交成功后保留弹窗并可手动下载账号信息 txt', async () => {
    vi.mocked(adminAPI.users.previewBatch).mockResolvedValue({
      items: [
        {
          row_no: 1,
          source_name: '张三',
          username: 'zhangsan',
          email: 'zhangsan@xssio.com',
          password: 'pass1234',
          notes: '',
          balance: 9999,
          concurrency: 3,
          errors: []
        }
      ]
    })
    vi.mocked(adminAPI.users.createBatch).mockResolvedValue({
      created_count: 1,
      failed_count: 0,
      users: [
        {
          row_no: 1,
          id: 101,
          email: 'zhangsan-updated@xssio.com',
          username: 'zhangsan-updated'
        }
      ],
      errors: []
    })

    let exportedBlob: Blob | null = null
    let clickedDownload = ''
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn((blob: Blob | MediaSource) => {
      exportedBlob = blob as Blob
      return 'blob:batch-users'
    }) as typeof window.URL.createObjectURL
    window.URL.revokeObjectURL = vi.fn(() => {}) as typeof window.URL.revokeObjectURL
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      clickedDownload = this.download
    })
    try {
      const wrapper = mountModal()
      const input = wrapper.get('input[type="file"]')
      const file = new File(['张三\n'], 'users.txt', { type: 'text/plain' })
      Object.defineProperty(file, 'text', {
        value: () => Promise.resolve('张三\n')
      })
      Object.defineProperty(input.element, 'files', {
        value: [file]
      })

      await input.trigger('change')
      await flushPromises()

      const textInputs = wrapper.findAll('input[type="text"]')
      await textInputs[0].setValue('zhangsan-updated')
      await wrapper.get('input[type="email"]').setValue('zhangsan-updated@xssio.com')
      await textInputs[1].setValue('newpass5678')

      await wrapper.get('.btn.btn-primary').trigger('click')
      await flushPromises()

      expect(adminAPI.users.createBatch).toHaveBeenCalledTimes(1)
      expect(exportedBlob).toBeNull()
      expect(wrapper.text()).toContain('admin.users.batchCreate.completedTitle')
      expect(wrapper.text()).toContain('admin.users.batchCreate.completedDescription:1')
      expect(showSuccess).toHaveBeenCalledWith('admin.users.batchCreate.success:1')
      expect(wrapper.emitted('success')).toBeTruthy()
      expect(wrapper.emitted('close')).toBeFalsy()

      const downloadButton = wrapper.findAll('button').find(button => button.text().includes('admin.users.batchCreate.downloadCredentials'))
      expect(downloadButton).toBeTruthy()

      await downloadButton!.trigger('click')
      await flushPromises()

      expect(exportedBlob).not.toBeNull()
      const content = await readBlobAsText(exportedBlob as Blob)
      expect(content).toContain('zhangsan-updated\tzhangsan-updated@xssio.com\tnewpass5678')
      expect(clickedDownload).toMatch(/^batch-users-credentials-\d{4}-\d{2}-\d{2}\.txt$/)
      expect(window.URL.revokeObjectURL).toHaveBeenCalledWith('blob:batch-users')
    } finally {
      window.URL.createObjectURL = originalCreateObjectURL
      window.URL.revokeObjectURL = originalRevokeObjectURL
      clickSpy.mockRestore()
    }
  })
})
