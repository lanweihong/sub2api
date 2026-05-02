import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { createAccountMock, showErrorMock, showSuccessMock } = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false })
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({})
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn().mockResolvedValue([])
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
})

function mountModal() {
  return mount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        ConfirmDialog: true,
        Select: SelectStub,
        Icon: true,
        ProxySelector: true,
        GroupSelector: true,
        ModelWhitelistSelector: true,
        QuotaLimitCard: true,
        OAuthAuthorizationFlow: true
      }
    }
  })
}

describe('CreateAccountModal', () => {
  it('submits OpenAI Chat Completions direct forward when creating an API key account', async () => {
    createAccountMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
    createAccountMock.mockResolvedValue({ id: 1 })

    const wrapper = mountModal()

    await wrapper.get('input[type="text"]').setValue('OpenAI Key')
    const openAIButton = wrapper.findAll('button').find(button => button.text().includes('OpenAI'))
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')

    const apiKeyButton = wrapper.findAll('button').find(button => button.text().includes('API Key'))
    expect(apiKeyButton).toBeTruthy()
    await apiKeyButton!.trigger('click')

    await wrapper.get('input[type="password"]').setValue('sk-test')
    await wrapper.get('[data-testid="openai-cc-direct-forward-toggle"]').trigger('click')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.openai_cc_direct_forward).toBe(true)
  })
})
