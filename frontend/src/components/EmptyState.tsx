import { Link } from 'react-router-dom'

export function EmptyState() {
  return (
    <div className="flex items-center justify-center min-h-[400px] px-4">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-10 max-w-md w-full text-center">
        {/* Icon */}
        <div className="text-5xl mb-4" aria-hidden="true">
          🏦
        </div>

        {/* Heading */}
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
          Connect your accounts to get started
        </h2>

        {/* Subtext */}
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
          Link your SimpleFIN account to see your balances across all your
          financial accounts in one place.
        </p>

        {/* CTA */}
        <Link
          to="/settings"
          className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-6 rounded-lg transition-colors"
        >
          Go to Settings
        </Link>
      </div>
    </div>
  )
}
