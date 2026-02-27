import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Box,
} from '@mui/material';
import { LocationOn, AccessTime } from '@mui/icons-material';
import { mockTrips } from '@/services/mockData';

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'success' | 'info' | 'warning' | 'error' }> = {
    completed: { label: 'Abgeschlossen', color: 'success' },
    in_progress: { label: 'Aktiv', color: 'info' },
    requested: { label: 'Angefragt', color: 'warning' },
    cancelled: { label: 'Storniert', color: 'error' },
  };
  const config = statusMap[status] || { label: status, color: 'info' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

export default function RecentTripsTable() {
  const recentTrips = mockTrips.slice(0, 5);

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Fahrgast</TableCell>
            <TableCell>Route</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Preis</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {recentTrips.map((trip) => (
            <TableRow key={trip.id}>
              <TableCell>{trip.riderName}</TableCell>
              <TableCell>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <LocationOn fontSize="small" color="success" />
                    <Typography variant="body2" noWrap sx={{ maxWidth: 150 }}>
                      {trip.pickup.address}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <LocationOn fontSize="small" color="error" />
                    <Typography variant="body2" noWrap sx={{ maxWidth: 150 }}>
                      {trip.dropoff.address}
                    </Typography>
                  </Box>
                </Box>
              </TableCell>
              <TableCell>{getStatusChip(trip.status)}</TableCell>
              <TableCell>€{trip.fare.total.toFixed(2)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

import { Typography } from '@mui/material';
