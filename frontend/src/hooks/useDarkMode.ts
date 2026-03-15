import { useState, useEffect } from 'react'

export function useDarkMode(): { isDark: boolean; toggle: () => void } {
  const [isDark, setIsDark] = useState<boolean>(
    () => localStorage.getItem('theme') === 'dark'
  )

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    }
  }, [isDark])

  function toggle() {
    setIsDark((prev) => !prev)
  }

  return { isDark, toggle }
}
