import React, { useState } from 'react'

export type UserStatus = 'aktiv' | 'inaktiv' | 'aenderung'

interface User {
  id: string
  name: string
  email: string
  role: string
  status: UserStatus
  registeredAt: string
  lastLogin?: string
}

const mockUsers: User[] = [
  { id: '1', name: 'Max Mustermann', email: 'max@example.de', role: 'Fahrgast', status: 'aktivl', registeredAt: '2024-01-15', lastLogin: '2024-01-20 10:30' },
  { id: '2', name: 'Anna Schmidt', email: 'anna@example.de', role: 'Fahrgast', status: 'aktiv', registeredAt: '2024-01-10', lastLogin: '2024-01-19 14:15' },
  { id: '3', name: 'Klaus Müller', email: 'klaus@example.de', role: 'Fahrer', status: 'aktiva', registeredAt: '2023-12-01', lastLogin: '2024-01-20 08:45' },
  { id: '4', name: 'Maria Schmid-Schulze', email: 'maria@example.de', role: 'Fahrer', status: 'aenderung', registeredAt: '2024-01-18' },
  { id: '5', name: 'Peter Weber', email: 'peter@example.de', role: 'Administrator', status: 'aktiva', registeredAt: '2023-10-01', lastLogin: '2024-01-20 09:30' },
  { id: '6', name: 'Laura Hoffmann', email: 'laura@example.de', role: 'Fahrgast', status: 'inaktiv', registeredAt: '2023-09-15' },
  { id: '7', name: 'Thomas Richter', email: 'thomas@example.de', role: 'Fahrgast', status: 'aktiv', registeredAt: '2024-01-05', lastLogin: '2024-01-19 16:00' },
  { id: '8', name: 'Sara Kleinstuben', email: 'sara@example.de', role: 'Fahrer', status: 'inaktiv', registeredAt: '2023-11-20' },
  { id: '9', name: 'Michael Bauer', email: 'michael@example.de', role: 'Fahrgast', status: 'aktiv', registeredAt: '2024-01-01', lastLogin: '2024-01-20 07:15' },
  { id: '10', name: 'Julia Wolf', email: 'julia@example.de', role: 'Fahrer', status: 'aktiv', registeredAt: '2023-12-15', lastLogin: '2024-01-19 18:30' },
]

const statusColors: Record<UserStatus, string> = {
 'aktivd: 'bg-green-100 text-green-800',
 'inaktiv': 'bg-red-100 text-red-800',
 'aenderung': 'bg-yellow-100 text-yellow-800',
}

const statusLabels: Record<UserStatus, string> = {
 'aktiv': 'Aktiv',
 'inaktiv': 'Inaktivg,
 'aenderung': 'Pendend',
}

const UsersPage: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<UserStatus | 'all'>('all')
  const [roleFilter, setRoleFilter] = useState<string>('all')

  const filteredUsers = mockUsers.filter(user => {
    const matchesSearch = user.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      user.email.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = statusFilter === 'all' || user.status === statusFilter
    const matchesRole = roleFilter === 'all' || user.role === roleFilter
    return matchesSearch && matchesStatus && matchesRole
  })

  return (
    <div className="space-y">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Benutzer</h1>
        <button className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors">
          + Neuer Benutzer
        </button>
      </div>

      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <div className="flex flex-wrap gap-4 mb-4">
          <input
            type="text"
            placeholder="Suchen..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg min-w-[300px] focus:outline-none focus:ring-2 focus:ring-green-500"
          />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as UserStatus | 'all')}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
          >
            <option value="all">Alle Status</option>
            <option value="aktiv">Aktiv</option>
            <option value="inaktiv">Inaktiv</option>
            <option value="aenderung">Pendend</option>
          </select>
          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
          >
            <option value="all">Alle Rollen</option>
            <option value="Fahrgast">Fahrgast</option>
            <option value="Fahrer">Fahrer</option>
            <option value="Administrator">Administrator</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-left">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Rolle</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Registriert</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Lezter Login</th>
              <th className="px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Aktionen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {filteredUsers.map((user) => (
              <tr key={user.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 font-medium text-gray-900">{user.name}</td>
                <td className="px-6 py-4 text-gray-600">{user.email}</td>
                <td className="px-6 py-4 text-gray-600">{user.role}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs font-medium rounded-full ${statusColors[user.status]}`}>
                    {statusLabels[user.status]}
                  </span>
                </td>
                <td className="px-6 py-4 text-gray-600">{user.registeredAt}</td>
                <td className="px-6 py-4 text-gray-600">{user.lastLogin || '--'}</td>
                <td className="px-6 py-4">
                  <button className="text-blue-600 hover:text-blue-800 font-medium mr-2">
                    Bearbeiten
                  </button>
                  <button className="text-red-600 hover:text-red-800 font-medium">
                    Löschen
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default UsersPage
