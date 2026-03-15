import { renderHook, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { useDarkMode } from './useDarkMode'

describe('useDarkMode', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  afterEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  it('initializes to light mode when localStorage is empty', () => {
    const { result } = renderHook(() => useDarkMode())
    expect(result.current.isDark).toBe(false)
  })

  it('initializes to dark mode when localStorage has "dark"', () => {
    localStorage.setItem('theme', 'dark')
    const { result } = renderHook(() => useDarkMode())
    expect(result.current.isDark).toBe(true)
  })

  it('adds dark class to html element when toggling to dark', () => {
    const { result } = renderHook(() => useDarkMode())
    act(() => {
      result.current.toggle()
    })
    expect(result.current.isDark).toBe(true)
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
  })

  it('removes dark class from html element when toggling to light', () => {
    localStorage.setItem('theme', 'dark')
    const { result } = renderHook(() => useDarkMode())
    act(() => {
      result.current.toggle()
    })
    expect(result.current.isDark).toBe(false)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
  })
})
