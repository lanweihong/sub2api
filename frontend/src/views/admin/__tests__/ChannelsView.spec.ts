import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import ChannelsView from '../ChannelsView.vue'

const {
  listChannels,
  createChannel,
  updateChannel,
  removeChannel,
  getAllGroups,
  getWebSearchEmulationConfig,
  listAccounts,
  getAccountById,
  showError,
  showSuccess,
} = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    listChannels: vi.fn(),
    createChannel: vi.fn(),
    updateChannel: vi.fn(),
    removeChannel: vi.fn(),
    getAllGroups: vi.fn(),
    getWebSearchEmulationConfig: vi.fn(),
    listAccounts: vi.fn(),
    getAccountById: vi.fn(),
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channels: {
      list: listChannels,
      create: createChannel,
      update: updateChannel,
      remove: removeChannel,
    },
    groups: {
      getAll: getAllGroups,
    },
    settings: {
      getWebSearchEmulationConfig,
    },
    accounts: {
      list: listAccounts,
      getById: getAccountById,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (error: unknown, fallback: string) => fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const translations: Record<string, string> = {
    'admin.channels.createChannel': 'Create Channel',
    'admin.channels.editChannel': 'Edit Channel',
    'admin.groups.platforms.anthropic': 'Anthropic',
    'admin.groups.platforms.anthropic-compatible': 'Anthropic-compatible',
    'admin.groups.platforms.anthropic-zhipu': 'Zhipu GLM',
    'admin.groups.platforms.anthropic-kimi': 'Kimi / Moonshot',
    'admin.groups.platforms.anthropic-minimax': 'MiniMax',
    'admin.groups.platforms.anthropic-qwen': 'Qwen / Tongyi',
    'admin.groups.platforms.anthropic-mimo': 'Xiaomi MiMo',
    'admin.groups.platforms.openai': 'OpenAI',
    'admin.groups.platforms.gemini': 'Gemini',
    'admin.groups.platforms.antigravity': 'Antigravity',
    'common.edit': 'Edit',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, arg1?: unknown, arg2?: unknown) => {
        if (translations[key]) {
          return translations[key]
        }
        if (typeof arg2 === 'string') {
          return arg2
        }
        if (typeof arg1 === 'string') {
          return arg1
        }
        return key
      },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: `
    <div>
      <slot name="filters" />
      <slot name="table" />
      <slot name="pagination" />
    </div>
  `,
}
const DataTableStub = defineComponent({
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => h('div', [
      ...(props.data as Array<Record<string, unknown>>).map((row, index) =>
        h('div', { key: String(row.id ?? index), class: 'data-row' }, slots['cell-actions']?.({ row }) ?? []),
      ),
      !(props.data as Array<unknown>).length && slots.empty ? slots.empty() : null,
    ])
  },
})
const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () => props.show
      ? h('div', { class: 'base-dialog-stub' }, [
        ...(slots.default?.() ?? []),
        ...(slots.footer?.() ?? []),
      ])
      : null
  },
})
const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    return () => h('select', {
      value: props.modelValue as string | number | undefined,
      onChange: (event: Event) => {
        const value = (event.target as HTMLSelectElement).value
        emit('update:modelValue', value)
        emit('change', value, null)
      },
    }, (props.options as Array<{ value: string; label: string }>).map(option =>
      h('option', { value: option.value }, option.label),
    ))
  },
})
const ToggleStub = defineComponent({
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('input', {
      type: 'checkbox',
      checked: props.modelValue,
      onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
    })
  },
})
const PricingEntryCardStub = defineComponent({
  props: {
    entry: {
      type: Object,
      required: true,
    },
    platform: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () => h(
      'div',
      { class: 'pricing-entry-stub', 'data-platform': props.platform },
      `${props.platform}:${((props.entry as { models?: string[] }).models ?? []).join(',')}`,
    )
  },
})

const platformGroups = [
  { id: 1, name: 'Anthropic Group', platform: 'anthropic', rate_multiplier: 1, account_count: 1 },
  { id: 2, name: 'Compat Group', platform: 'anthropic-compatible', rate_multiplier: 1, account_count: 1 },
  { id: 3, name: 'Zhipu Group', platform: 'anthropic-zhipu', rate_multiplier: 1, account_count: 1 },
  { id: 4, name: 'Kimi Group', platform: 'anthropic-kimi', rate_multiplier: 1, account_count: 1 },
  { id: 5, name: 'MiniMax Group', platform: 'anthropic-minimax', rate_multiplier: 1, account_count: 1 },
  { id: 6, name: 'Qwen Group', platform: 'anthropic-qwen', rate_multiplier: 1, account_count: 1 },
  { id: 7, name: 'MiMo Group', platform: 'anthropic-mimo', rate_multiplier: 1, account_count: 1 },
  { id: 8, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 1, account_count: 1 },
  { id: 9, name: 'Gemini Group', platform: 'gemini', rate_multiplier: 1, account_count: 1 },
  { id: 10, name: 'Antigravity Group', platform: 'antigravity', rate_multiplier: 1, account_count: 1 },
]

function mountView() {
  return mount(ChannelsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        Icon: true,
        PlatformIcon: true,
        Toggle: ToggleStub,
        PricingEntryCard: PricingEntryCardStub,
      },
    },
  })
}

describe('admin ChannelsView compat platforms', () => {
  beforeEach(() => {
    listChannels.mockReset()
    createChannel.mockReset()
    updateChannel.mockReset()
    removeChannel.mockReset()
    getAllGroups.mockReset()
    getWebSearchEmulationConfig.mockReset()
    listAccounts.mockReset()
    getAccountById.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    listChannels.mockResolvedValue({
      items: [],
      total: 0,
    })
    getAllGroups.mockResolvedValue(platformGroups)
    getWebSearchEmulationConfig.mockResolvedValue({ enabled: false, providers: [] })
    listAccounts.mockResolvedValue({ items: [] })
  })

  it('shows all anthropic-compatible platforms in the channel platform selector', async () => {
    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper.findAll('button').find(button => button.text().includes('Create Channel'))
    expect(createButton).toBeDefined()

    await createButton!.trigger('click')
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Anthropic-compatible')
    expect(text).toContain('Zhipu GLM')
    expect(text).toContain('Kimi / Moonshot')
    expect(text).toContain('MiniMax')
    expect(text).toContain('Qwen / Tongyi')
    expect(text).toContain('Xiaomi MiMo')
  })

  it('rehydrates compat platform sections with existing mappings and pricing when editing a channel', async () => {
    listChannels.mockResolvedValue({
      items: [
        {
          id: 99,
          name: 'Compat Channel',
          description: 'compat',
          status: 'active',
          billing_model_source: 'channel_mapped',
          restrict_models: false,
          features_config: {},
          group_ids: [2, 3],
          model_pricing: [
            {
              platform: 'anthropic-compatible',
              models: ['vendor-*'],
              billing_mode: 'token',
              input_price: 1e-6,
              output_price: 2e-6,
              cache_write_price: null,
              cache_read_price: null,
              image_output_price: null,
              per_request_price: null,
              intervals: [],
            },
            {
              platform: 'anthropic-zhipu',
              models: ['glm-4-plus'],
              billing_mode: 'token',
              input_price: 2e-6,
              output_price: 4e-6,
              cache_write_price: null,
              cache_read_price: null,
              image_output_price: null,
              per_request_price: null,
              intervals: [],
            },
          ],
          model_mapping: {
            'anthropic-compatible': {
              'vendor-*': 'vendor-target',
            },
            'anthropic-zhipu': {
              'glm-4-plus': 'glm-4-air',
            },
          },
          apply_pricing_to_account_stats: false,
          account_stats_pricing_rules: [],
          created_at: '2026-04-01T00:00:00Z',
          updated_at: '2026-04-01T00:00:00Z',
        },
      ],
      total: 1,
    })

    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('Edit'))
    expect(editButton).toBeDefined()

    await editButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Anthropic-compatible')
    expect(wrapper.text()).toContain('Zhipu GLM')

    expect(wrapper.find('[data-platform="anthropic-compatible"]').text()).toContain('vendor-*')
    expect(wrapper.find('[data-platform="anthropic-zhipu"]').text()).toContain('glm-4-plus')

    expect(wrapper.find('input[value="vendor-*"]').exists()).toBe(true)
    expect(wrapper.find('input[value="vendor-target"]').exists()).toBe(true)
    expect(wrapper.find('input[value="glm-4-plus"]').exists()).toBe(true)
    expect(wrapper.find('input[value="glm-4-air"]').exists()).toBe(true)
  })
})
