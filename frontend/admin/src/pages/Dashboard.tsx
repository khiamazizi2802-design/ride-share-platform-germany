import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  IconButton,
  LinearProgress,
} from '@mui/material';
import {
  TrendingUp,
  LocalTaxi,
  DirectionsCar,
  People,
  Warning,
  ArrowForward,
} from '@mui/icons-material';
import { useDashboardStore } from '@/store/dashboardStore';
import { mockDashboardStats } from '@/services/mockData';
import StatCard from '@/components/StatCard';
import RevenueChart from '@/components/RevenueChart';
import RecentTripsTable from '@/components/RecentTripsTable';

export default function Dashboard() {
  const navigate = useNavigate();
  const { stats, setStats } = useDashboardStore();

  useEffect(() => {
    // In production, fetch from API
    setStats(mockDashboardStats);
  }, [setStats]);

  const statCards = [
    {
      title: 'Heutiger Umsatz',
      value: `€${stats.todayRevenue.toFixed(2)}`,
      icon: TrendingUp,
      trend: '+12%',
      trendUp: true,
      color: '#22C55E',
    },
    {
      title: 'Heutige Fahrten',
      value: stats.todayTrips.toString(),
      icon: LocalTaxi,
      trend: '+5%',
      trendUp: true,
      color: '#0EA5E9',
    },
    {
      title: 'Aktive Fahrten',
      value: stats.activeTrips.toString(),
      icon: DirectionsCar,
      trend: 'Live',
      trendUp: true,
      color: '#8B5CF6',
    },
    {
      title: 'Online Fahrer',
      value: stats.onlineDrivers.toString(),
      icon: People,
      trend: '+3',
      trendUp: true,
      color: '#F59E0B',
    },
  ];

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Dashboard
      </Typography>

      {/* Stats Grid */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        {statCards.map((card, index) => (
          <Grid item xs={12} sm={6} lg={3} key={index}>
            <StatCard {...card} />
          </Grid>
        ))}
      </Grid>

      <Grid container spacing={3}>
        {/* Revenue Chart */}
        <Grid item xs={12} lg={8}>
          <Card sx={{ height: 400 }}>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6" fontWeight={600}>
                  Umsatzübersicht
                </Typography>
                <IconButton size="small" onClick={() => navigate('/analytics')}>
                  <ArrowForward />
                </IconButton>
              </Box>
              <RevenueChart />
            </CardContent>
          </Card>
        </Grid>

        {/* Alerts & Quick Actions */}
        <Grid item xs={12} lg={4}>
          <Grid container spacing={3}>
            {/* Pending Verifications */}
            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                    <Box
                      sx={{
                        p: 1.5,
                        borderRadius: 2,
                        bgcolor: 'warning.light',
                        color: 'warning.dark',
                      }}
                    >
                      <Warning />
                    </Box>
                    <Box>
                      <Typography variant="h6" fontWeight={600}>
                        {stats.pendingVerifications}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Ausstehende Verifizierungen
                      </Typography>
                    </Box>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={(stats.pendingVerifications / 20) * 100}
                    sx={{
                      height: 8,
                      borderRadius: 4,
                      bgcolor: 'warning.light',
                      '& .MuiLinearProgress-bar': {
                        bgcolor: 'warning.main',
                      },
                    }}
                  />
                  <Box sx={{ mt: 2, textAlign: 'right' }}>
                    <IconButton size="small" onClick={() => navigate('/drivers')}>
                      <Typography variant="body2" color="primary" sx={{ mr: 0.5 }}>
                        Ansehen
                      </Typography>
                      <ArrowForward fontSize="small" />
                    </IconButton>
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            {/* Support Tickets */}
            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                    <Box
                      sx={{
                        p: 1.5,
                        borderRadius: 2,
                        bgcolor: 'error.light',
                        color: 'error.dark',
                      }}
                    >
                      <Warning />
                    </Box>
                    <Box>
                      <Typography variant="h6" fontWeight={600}>
                        {stats.openSupportTickets}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Offene Support-Tickets
                      </Typography>
                    </Box>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={(stats.openSupportTickets / 10) * 100}
                    sx={{
                      height: 8,
                      borderRadius: 4,
                      bgcolor: 'error.light',
                      '& .MuiLinearProgress-bar': {
                        bgcolor: 'error.main',
                      },
                    }}
                  />
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </Grid>

        {/* Recent Trips */}
        <Grid item xs={12}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6" fontWeight={600}>
                  Aktuelle Fahrten
                </Typography>
                <IconButton size="small" onClick={() => navigate('/trips')}>
                  <Typography variant="body2" color="primary" sx={{ mr: 0.5 }}>
                    Alle anzeigen
                  </Typography>
                  <ArrowForward fontSize="small" />
                </IconButton>
              </Box>
              <RecentTripsTable />
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
