import React from 'react'

interface StatCardProps {
  title: string
  value: string | number
  subtitle?: string
  trend?: {
    value: number
    label?: string
    direction?: 'up' | 'down' | 'neutral'
  }
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple' | 'indigo'
}

const StatCard: React.FC<StatCardProps> = ({ title, value, subtitle, trend, color = 'blue' }) => {
  const colorClasses = {
    blue: 'bg-blue-500',
    green: 'bg-green-500',
    yellow: 'bg-yellow-500',
    red: 'bg-red-500',
    purple: 'bg-purple-500',
    indigo: 'bg-indigo-500',
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
          {subtitle && <p className="text-sm text-gray-500">{subtitle}</p>}
        </div>
        <div className={`w-12 h-12 rounded-full ${colorClasses[color]} flex items-center justify-center`}>
          <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0M13 12h8m0-8h8m0 0h8m0 0h8m0 0h8m0 0h8m0 0h8" />
          </svg>
        </div>
      </div>
      {trend && (
        <div className="flex items-center mt-2">
          <span className={`text-sm font-medium ${trend.direction === 'up' ? 'text-green-600' : trend.direction === 'down' ? 'text-red-600' : 'text-gray-600'}`}>
            {trend.direction === 'up' ? '↔' : trend.direction === 'down' ? '↔' : '≤'}
            {trend.value}%
          </span>
          {trend.label && <span className="text-sm text-gray-500 ml-1">{trend.label}</span>}
        </div>
      )}
    </div>
  )
}

const DashboardPage: React.FC = () => {
  return (
    <div className="space-y">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-x2 font-bold text-gray-900">Übersicht</h1>
        <span className="text-sm text-gray-500">{new Date().toLocaleDateString('de-DE')}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Gesamtumsatz"
          value="€58.450"
          subtitle="€5.2% vor Monat"
          trend={{ value: 5.2, direction: 'up', label: 'vor Monat' }}
          color="green"
        />
        <StatCard
          title="Aktive Fahren"
          value="245"
          subtitle="18 neu registriert"
          trend={{ value: 18, direction: 'up', label: 'neu registriert' }}
          color="blue"
        />
        <StatCard
          title="Aktive Fahrten"
          value="42"
          subtitle="8 in Realtime"
          trend={{ value: 8, direction: 'up', label: 'in Realtime' }}
          color="purple"
        />
        <StatCard
          title="Benutzerbewertungen"
          value="98%"
          subtitle="+2% im Snit"
          trend={{ value: 2, direction: 'up', label: 'im Snit' }}
          color="yellow"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-bold text-gray-900 mb-4">Übersicht Fahrten</h2>
          <div className="h-64 bg-gray-50 rounded-lg flex items-center justify-center">
            <p className="text-gray-500">Karten platzhalter</p>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-bold text-gray-900 mb-4">Süperre Fahrten</h2>
          <div className="h-64 bg-gray-50 rounded-lg flex items-center justify-center">
            <p className="text-gray-500">Liste platzhalter</p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default DashboardPage
