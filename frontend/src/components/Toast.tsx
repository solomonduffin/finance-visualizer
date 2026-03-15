import { useEffect } from 'react'

interface ToastProps {
  message: string
  onDismiss: () => void
}

export function Toast({ message, onDismiss }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onDismiss, 4000)
    return () => clearTimeout(timer)
  }, [onDismiss])

  return (
    <div
      role="status"
      className="fixed bottom-6 right-6 z-50 bg-green-600 text-white px-4 py-3 rounded-lg shadow-lg animate-[slideUp_0.3s_ease-out]"
      style={{
        animation: 'slideUp 0.3s ease-out',
      }}
    >
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium">{message}</span>
        <button
          type="button"
          onClick={onDismiss}
          className="text-white/80 hover:text-white text-lg leading-none"
          aria-label="Dismiss"
        >
          &times;
        </button>
      </div>
      <style>{`
        @keyframes slideUp {
          from { transform: translateY(100%); opacity: 0; }
          to { transform: translateY(0); opacity: 1; }
        }
      `}</style>
    </div>
  )
}
