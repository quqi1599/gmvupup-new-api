import { fireEvent, render, screen } from '@testing-library/react'
import { FormProvider, useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'

import { ZpayPreset } from '../zpay-preset'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))
function SettingsFixture() {
  const form = useForm({
    defaultValues: {
      PayAddress: 'https://existing.example',
      EpayId: 'existing-merchant',
      EpayKey: 'existing-secret',
    },
  })
  return (
    <FormProvider {...form}>
      <ZpayPreset />
      <input aria-label='Gateway' {...form.register('PayAddress')} />
      <input aria-label='Merchant' {...form.register('EpayId')} />
      <input aria-label='Secret' {...form.register('EpayKey')} />
    </FormProvider>
  )
}
describe('ZPay gateway preset', () => {
  it('fills the documented gateway only after a click and preserves merchant credentials', () => {
    render(<SettingsFixture />)
    expect(screen.getByLabelText('Gateway')).toHaveValue(
      'https://existing.example'
    )
    fireEvent.click(screen.getByRole('button', { name: 'Use ZPay gateway' }))
    expect(screen.getByLabelText('Gateway')).toHaveValue('https://zpayz.cn')
    expect(screen.getByLabelText('Merchant')).toHaveValue('existing-merchant')
    expect(screen.getByLabelText('Secret')).toHaveValue('existing-secret')
    expect(
      screen.getByRole('link', { name: 'ZPay documentation' })
    ).toHaveAttribute('href', 'https://7-pay.cn/doc.html')
  })
})
