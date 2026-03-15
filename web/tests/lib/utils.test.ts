import { describe, it, expect } from 'vitest'
import { errorMessage, cn } from '~/lib/utils'

describe('errorMessage', () => {
  it('extracts message from Error instances', () => {
    expect(errorMessage(new Error('something broke'))).toBe('something broke')
  })

  it('returns string errors as-is', () => {
    expect(errorMessage('network failure')).toBe('network failure')
  })

  it('returns fallback for unknown error types', () => {
    expect(errorMessage(42)).toBe('An unknown error occurred')
    expect(errorMessage(null)).toBe('An unknown error occurred')
    expect(errorMessage(undefined)).toBe('An unknown error occurred')
    expect(errorMessage({ code: 500 })).toBe('An unknown error occurred')
  })

  it('returns fallback for boolean values', () => {
    expect(errorMessage(true)).toBe('An unknown error occurred')
    expect(errorMessage(false)).toBe('An unknown error occurred')
  })
})

describe('cn', () => {
  it('merges class names', () => {
    expect(cn('px-2', 'py-1')).toBe('px-2 py-1')
  })

  it('handles tailwind conflicts by keeping the last one', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible')
  })

  it('handles empty input', () => {
    expect(cn()).toBe('')
  })

  it('filters out falsy values', () => {
    expect(cn('a', undefined, null, '', 'b')).toBe('a b')
  })
})
