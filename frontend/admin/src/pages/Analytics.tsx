import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  ToggleButtonGroup,
  ToggleButton,
} from '@mui/material';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import { mockRevenueData } from '@/services/mockData';

const tripData = [
  { name: 'Mo', trips: 45, revenue: 2450 },
  { name: 'Di', trips: 52, revenue: 2680 },
  { name: 'Mi', trips: 38, revenue: 2100 },
  { name: 'Do', trips: 62, revenue: 3200 },
  { name: 'Fr', trips: 58, revenue: 2950 },
  { name: 'Sa', trips: 72, revenue: 3800 },
  { name: 'So', trips: 48, revenue: 2650 },
];

const driverPerformance = [
  { name: 'Hans Müller', trips: 1247, rating: 4.8 },
  { name: 'Klaus Weber', trips: 523, rating: 4.2 },
  { name: 'Maria Schmidt', trips: 892, rating: 4.9 },
  { name: 'Peter Klein', trips: 445, rating: 4.5 },
];

const vehicleTypeData = [
  { name: 'Standard', value: 65, color: '#22C55E' },
  { name: 'XL', value: 20, color: '#0EA5E9' },
  { name: 'Premium', value: 10, color: '#8B5CF6' },
  { name: 'Elektro', value: 5, color: '#F59E0B' },
];

export default function Analytics() {
  const [period, setPeriod] = useState<'week' | 'month' | 'year'>('week');

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" fontWeight={700}>
          Analysen & Berichte
        </Typography>
        <ToggleButtonGroup
          value={period}
          exclusive
          onChange={(_, v) => v && setPeriod(v)}
          size="small"
        >
          <ToggleButton value="week">Woche</ToggleButton>
          <ToggleButton value="month">Monat</ToggleButton>
          <ToggleButton value="year">Jahr</ToggleButton>
        </ToggleButtonGroup>
      </Box>

      <Grid container spacing={3}>
        {/* Revenue Chart */}
        <Grid item xs={12} lg={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Umsatz & Fahrten
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={tripData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis yAxisId="left" />
                  <YAxis yAxisId="right" orientation="right" />
                  <Tooltip />
                  <Legend />
                  <Bar yAxisId="left" dataKey="trips" name="Fahrten" fill="#22C55E" radius={[4, 4, 0, 0]} />
                  <Bar yAxisId="right" dataKey="revenue" name="Umsatz (€)" fill="#0EA5E9" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Vehicle Type Distribution */}
        <Grid item xs={12} lg={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Fahrzeugtypen
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={vehicleTypeData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={100}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {vehicleTypeData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Driver Performance */}
        <Grid item xs={12}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Top Fahrer
              </Typography>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={driverPerformance} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis type="number" />
                  <YAxis dataKey="name" type="category" width={120} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="trips" name="Gesamtfahrten" fill="#22C55E" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Revenue Trend */}
        <Grid item xs={12}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Umsatzentwicklung
              </Typography>
              <ResponsiveContainer width="100%" height={250}>
                <LineChart data={mockRevenueData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" tickFormatter={(val) => new Date(val).toLocaleDateString('de-DE', { day: 'numeric', month: 'short' })} />
                  <YAxis />
                  <Tooltip formatter={(val: number) => `€${val.toFixed(2)}`} />
                  <Legend />
                  <Line type="monotone" dataKey="revenue" name="Umsatz" stroke="#22C55E" strokeWidth={2} />
                  <Line type="monotone" dataKey="commission" name="Provision" stroke="#0EA5E9" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
