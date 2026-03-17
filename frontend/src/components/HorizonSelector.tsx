import { useState, useRef, useEffect, useCallback } from 'react'

const PRESETS = [
  { label: '1y', years: 1 },
  { label: '5y', years: 5 },
  { label: '10y', years: 10 },
  { label: '20y', years: 20 },
  { label: 'Custom', years: -1 },
] as const

interface HorizonSelectorProps {
  years: number
  onChange: (years: number) => void
}

export function HorizonSelector({ years, onChange }: HorizonSelectorProps) {
  const isPresetValue = PRESETS.some((p) => p.years === years && p.years !== -1)
  const [customMode, setCustomMode] = useState(!isPresetValue)

  const [customValue, setCustomValue] = useState<string>(
    isPresetValue ? '5' : String(years),
  )
  const inputRef = useRef<HTMLInputElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Focus input when Custom is clicked
  useEffect(() => {
    if (customMode && inputRef.current) {
      inputRef.current.focus()
    }
  }, [customMode])

  // Cleanup debounce on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [])

  const handleCustomChange = useCallback(
    (value: string) => {
      setCustomValue(value)
      const num = parseInt(value, 10)
      if (!isNaN(num) && num >= 1 && num <= 50) {
        if (debounceRef.current) {
          clearTimeout(debounceRef.current)
        }
        debounceRef.current = setTimeout(() => {
          onChange(num)
        }, 500)
      }
    },
    [onChange],
  )

  const handleCustomBlur = useCallback(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
      debounceRef.current = null
    }
    const num = parseInt(customValue, 10)
    if (!isNaN(num) && num >= 1 && num <= 50) {
      onChange(num)
    }
  }, [customValue, onChange])

  const handlePresetClick = useCallback(
    (presetYears: number) => {
      if (presetYears === -1) {
        // Custom clicked: enter custom mode
        setCustomMode(true)
        const num = parseInt(customValue, 10)
        if (!isNaN(num) && num >= 1 && num <= 50) {
          onChange(num)
        } else {
          setCustomValue('5')
          onChange(5)
        }
      } else {
        setCustomMode(false)
        onChange(presetYears)
      }
    },
    [customValue, onChange],
  )

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div
        role="radiogroup"
        aria-label="Select projection horizon"
        className="inline-flex rounded-lg bg-gray-100 dark:bg-gray-800 p-1"
      >
        {PRESETS.map(({ label, years: presetYears }) => {
          const isActive =
            presetYears === -1
              ? customMode
              : !customMode && years === presetYears

          return (
            <button
              key={label}
              type="button"
              role="radio"
              aria-checked={isActive}
              onClick={() => handlePresetClick(presetYears)}
              className={`px-3 py-2 text-sm rounded-md transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white font-semibold shadow-sm'
                  : 'bg-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 font-normal'
              }`}
            >
              {label}
            </button>
          )
        })}
      </div>

      {customMode && (
        <div className="flex items-center gap-2 mt-2 md:mt-0 md:ml-3">
          <input
            ref={inputRef}
            type="number"
            min={1}
            max={50}
            step={1}
            value={customValue}
            onChange={(e) => handleCustomChange(e.target.value)}
            onBlur={handleCustomBlur}
            className="w-20 px-2 py-2 text-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            aria-label="Custom projection years"
          />
          <span className="text-sm text-gray-600 dark:text-gray-400">
            years
          </span>
        </div>
      )}
    </div>
  )
}
