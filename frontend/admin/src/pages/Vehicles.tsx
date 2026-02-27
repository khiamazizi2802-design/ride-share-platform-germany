import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  TextField,
  InputAdornment,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Grid,
  LinearProgress,
} from '@mui/material';
import {
  Search,
  Visibility,
  CheckCircle,
  Warning,
  DirectionsCar,
} from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { Vehicle } from '@/types';
import { mockVehicles } from '@/services/mockData';

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'success' | 'error' | 'warning' | 'default' }> = {
    active: { label: 'Aktiv', color: 'success' },
    inactive: { label: 'Inaktiv', color: 'default' },
    maintenance: { label: 'Wartung', color: 'warning' },
  };
  const config = statusMap[status] || { label: status, color: 'default' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

const getTypeLabel = (type: string) => {
  const typeMap: Record<string, string> = {
    standard: 'Standard',
    xl: 'XL',
    premium: 'Premium',
    electric: 'Elektro',
  };
  return typeMap[type] || type;
};

export default function Vehicles() {
  const [vehicles] = useState<Vehicle[]>(mockVehicles);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedVehicle, setSelectedVehicle] = useState<Vehicle | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const handleViewVehicle = (vehicle: Vehicle) => {
    setSelectedVehicle(vehicle);
    setDetailOpen(true);
  };

  const filteredVehicles = vehicles.filter((vehicle) =>
    vehicle.licensePlate.toLowerCase().includes(searchQuery.toLowerCase()) ||
    vehicle.make.toLowerCase().includes(searchQuery.toLowerCase()) ||
    vehicle.model.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const getDaysUntilExpiry = (date: string) => {
    const expiry = new Date(date);
    const today = new Date();
    const diffTime = expiry.getTime() - today.getTime();
    return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  };

  const columns: GridColDef[] = [
    {
      field: 'vehicle',
      headerName: 'Fahrzeug',
      flex: 2,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <DirectionsCar color="action" />
          <Box>
            <Typography variant="body2" fontWeight={600}>
              {params.row.make} {params.row.model}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {params.row.licensePlate}
            </Typography>
          </Box>
        </Box>
      ),
    },
    {
      field: 'type',
      headerName: 'Typ',
      flex: 1,
      renderCell: (params) => getTypeLabel(params.value),
    },
    {
      field: 'year',
      headerName: 'Baujahr',
      flex: 0.8,
    },
    {
      field: 'status',
      headerName: 'Status',
      flex: 1,
      renderCell: (params) => getStatusChip(params.value),
    },
    {
      field: 'tuvExpiry',
      headerName: 'TÜV',
      flex: 1.2,
      renderCell: (params) => {
        const days = getDaysUntilExpiry(params.value);
        return (
          <Box sx={{ width: '100%' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="caption">{days > 0 ? `Noch ${days} Tage` : 'Abgelaufen'}</Typography>
            </Box>
            <LinearProgress
              variant="determinate"
              value={Math.max(0, Math.min(100, (days / 365) * 100))}
              color={days < 30 ? 'error' : days < 90 ? 'warning' : 'success'}
              sx={{ height: 6, borderRadius: 3 }}
            />
          </Box>
        );
      },
    },
    {
      field: 'actions',
      headerName: 'Aktionen',
      flex: 1,
      sortable: false,
      renderCell: (params) => (
        <IconButton size="small" onClick={() => handleViewVehicle(params.row)}>
          <Visibility fontSize="small" />
        </IconButton>
      ),
    },
  ];

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Fahrzeugverwaltung
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <TextField
            placeholder="Fahrzeug suchen..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <Search />
                </InputAdornment>
              ),
            }}
            sx={{ width: { xs: '100%', sm: 400 } }}
          />
        </CardContent>
      </Card>

      <Card>
        <DataGrid
          rows={filteredVehicles}
          columns={columns}
          initialState={{
            pagination: { paginationModel: { pageSize: 10 } },
          }}
          pageSizeOptions={[10, 25, 50]}
          disableRowSelectionOnClick
          autoHeight
          sx={{ border: 'none' }}
          getRowId={(row) => row.id}
        />
      </Card>

      <Dialog open={detailOpen} onClose={() => setDetailOpen(false)} maxWidth="sm" fullWidth>
        {selectedVehicle && (
          <>
            <DialogTitle>
              {selectedVehicle.make} {selectedVehicle.model}
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Kennzeichen</Typography>
                  <Typography fontWeight={600}>{selectedVehicle.licensePlate}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Typ</Typography>
                  <Typography>{getTypeLabel(selectedVehicle.type)}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Baujahr</Typography>
                  <Typography>{selectedVehicle.year}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Farbe</Typography>
                  <Typography>{selectedVehicle.color}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">TÜV gültig bis</Typography>
                  <Typography>{new Date(selectedVehicle.tuvExpiry).toLocaleDateString('de-DE')}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Versicherung gültig bis</Typography>
                  <Typography>{new Date(selectedVehicle.insuranceExpiry).toLocaleDateString('de-DE')}</Typography>
                </Grid>
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setDetailOpen(false)}>Schließen</Button>
              <Button variant="contained" color="primary">Bearbeiten</Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
}
