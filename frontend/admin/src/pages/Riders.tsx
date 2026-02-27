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
  Avatar,
  Grid,
} from '@mui/material';
import {
  Search,
  Block,
  Visibility,
  Phone,
  Email,
  Star,
} from '@mui/icons-material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Rider } from '@/types';
import { mockRiders } from '@/services/mockData';

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'success' | 'error' | 'warning' | 'default' }> = {
    active: { label: 'Aktiv', color: 'success' },
    suspended: { label: 'Gesperrt', color: 'warning' },
    banned: { label: 'Gesperrt', color: 'error' },
  };
  const config = statusMap[status] || { label: status, color: 'default' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

export default function Riders() {
  const [riders] = useState<Rider[]>(mockRiders);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRider, setSelectedRider] = useState<Rider | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const handleViewRider = (rider: Rider) => {
    setSelectedRider(rider);
    setDetailOpen(true);
  };

  const handleSuspend = (riderId: string) => {
    console.log('Suspending rider:', riderId);
  };

  const filteredRiders = riders.filter((rider) =>
    rider.firstName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    rider.lastName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    rider.email.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const columns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'Fahrgast',
      flex: 2,
      renderCell: (params: GridRenderCellParams) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Avatar sx={{ bgcolor: '#0EA5E9' }}>
            {params.row.firstName[0]}{params.row.lastName[0]}
          </Avatar>
          <Box>
            <Typography variant="body2" fontWeight={600}>
              {params.row.firstName} {params.row.lastName}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {params.row.email}
            </Typography>
          </Box>
        </Box>
      ),
    },
    {
      field: 'status',
      headerName: 'Status',
      flex: 1,
      renderCell: (params) => getStatusChip(params.value),
    },
    {
      field: 'rating',
      headerName: 'Bewertung',
      flex: 0.8,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Star fontSize="small" sx={{ color: '#F59E0B' }} />
          <Typography variant="body2">{params.value}</Typography>
        </Box>
      ),
    },
    {
      field: 'totalTrips',
      headerName: 'Fahrten',
      flex: 0.8,
    },
    {
      field: 'totalSpent',
      headerName: 'Ausgaben',
      flex: 1,
      renderCell: (params) => `€${params.value.toFixed(2)}`,
    },
    {
      field: 'createdAt',
      headerName: 'Registriert',
      flex: 1,
      renderCell: (params) => new Date(params.value).toLocaleDateString('de-DE'),
    },
    {
      field: 'actions',
      headerName: 'Aktionen',
      flex: 1,
      sortable: false,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <IconButton size="small" onClick={() => handleViewRider(params.row)}>
            <Visibility fontSize="small" />
          </IconButton>
          {params.row.status === 'active' && (
            <IconButton
              size="small"
              color="error"
              onClick={() => handleSuspend(params.row.id)}
            >
              <Block fontSize="small" />
            </IconButton>
          )}
        </Box>
      ),
    },
  ];

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Fahrgastverwaltung
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <TextField
            placeholder="Fahrgast suchen..."
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
          rows={filteredRiders}
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

      {/* Rider Detail Dialog */}
      <Dialog open={detailOpen} onClose={() => setDetailOpen(false)} maxWidth="sm" fullWidth>
        {selectedRider && (
          <>
            <DialogTitle>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ width: 56, height: 56, bgcolor: '#0EA5E9', fontSize: 24 }}>
                  {selectedRider.firstName[0]}{selectedRider.lastName[0]}
                </Avatar>
                <Box>
                  <Typography variant="h6">
                    {selectedRider.firstName} {selectedRider.lastName}
                  </Typography>
                  {getStatusChip(selectedRider.status)}
                </Box>
              </Box>
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Kontakt
                  </Typography>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Email fontSize="small" color="action" />
                      <Typography>{selectedRider.email}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Phone fontSize="small" color="action" />
                      <Typography>{selectedRider.phone}</Typography>
                    </Box>
                  </Box>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Statistiken
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 4 }}>
                    <Box>
                      <Typography variant="h6">{selectedRider.totalTrips}</Typography>
                      <Typography variant="body2" color="text.secondary">Fahrten</Typography>
                    </Box>
                    <Box>
                      <Typography variant="h6">€{selectedRider.totalSpent.toFixed(2)}</Typography>
                      <Typography variant="body2" color="text.secondary">Ausgaben</Typography>
                    </Box>
                    <Box>
                      <Typography variant="h6">{selectedRider.rating} ⭐</Typography>
                      <Typography variant="body2" color="text.secondary">Bewertung</Typography>
                    </Box>
                  </Box>
                </Grid>
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setDetailOpen(false)}>Schließen</Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
}
