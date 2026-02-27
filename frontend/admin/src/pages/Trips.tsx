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
  Grid,
} from '@mui/material';
import {
  Search,
  Visibility,
  LocationOn,
  AccessTime,
  AttachMoney,
} from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { MapContainer, TileLayer, Marker, Popup, Polyline } from 'react-leaflet';
import { Trip } from '@/types';
import { mockTrips } from '@/services/mockData';
import L from 'leaflet';

// Fix for default markers
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

let DefaultIcon = L.icon({
  iconUrl: icon,
  shadowUrl: iconShadow,
  iconSize: [25, 41],
  iconAnchor: [12, 41],
});

L.Marker.prototype.options.icon = DefaultIcon;

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'success' | 'warning' | 'info' | 'error' | 'default' }> = {
    completed: { label: 'Abgeschlossen', color: 'success' },
    in_progress: { label: 'In Bearbeitung', color: 'info' },
    requested: { label: 'Angefragt', color: 'warning' },
    accepted: { label: 'Angenommen', color: 'info' },
    cancelled: { label: 'Storniert', color: 'error' },
  };
  const config = statusMap[status] || { label: status, color: 'default' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

export default function Trips() {
  const [trips] = useState<Trip[]>(mockTrips);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTrip, setSelectedTrip] = useState<Trip | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const handleViewTrip = (trip: Trip) => {
    setSelectedTrip(trip);
    setDetailOpen(true);
  };

  const filteredTrips = trips.filter((trip) =>
    trip.riderName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (trip.driverName && trip.driverName.toLowerCase().includes(searchQuery.toLowerCase())) ||
    trip.pickup.address.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const columns: GridColDef[] = [
    {
      field: 'id',
      headerName: 'Fahrt-ID',
      flex: 1,
      renderCell: (params) => <Typography variant="body2" fontFamily="monospace">{params.value.slice(-8)}</Typography>,
    },
    {
      field: 'riderName',
      headerName: 'Fahrgast',
      flex: 1.5,
    },
    {
      field: 'driverName',
      headerName: 'Fahrer',
      flex: 1.5,
      renderCell: (params) => params.value || '-',
    },
    {
      field: 'status',
      headerName: 'Status',
      flex: 1,
      renderCell: (params) => getStatusChip(params.value),
    },
    {
      field: 'pickup',
      headerName: 'Abholung',
      flex: 2,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <LocationOn fontSize="small" color="action" />
          <Typography variant="body2" noWrap>{params.value.address}</Typography>
        </Box>
      ),
    },
    {
      field: 'fare',
      headerName: 'Preis',
      flex: 0.8,
      renderCell: (params) => `€${params.value.total.toFixed(2)}`,
    },
    {
      field: 'createdAt',
      headerName: 'Zeit',
      flex: 1,
      renderCell: (params) => new Date(params.value).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' }),
    },
    {
      field: 'actions',
      headerName: 'Aktionen',
      flex: 0.8,
      sortable: false,
      renderCell: (params) => (
        <IconButton size="small" onClick={() => handleViewTrip(params.row)}>
          <Visibility fontSize="small" />
        </IconButton>
      ),
    },
  ];

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Fahrtenüberwachung
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <TextField
            placeholder="Fahrt suchen..."
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

      <Grid container spacing={3}>
        <Grid item xs={12} lg={8}>
          <Card>
            <DataGrid
              rows={filteredTrips}
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
        </Grid>
        <Grid item xs={12} lg={4}>
          <Card sx={{ height: 500 }}>
            <CardContent sx={{ height: '100%', p: 0 }}>
              <Typography variant="h6" sx={{ p: 2, pb: 1 }}>
                Live-Karte
              </Typography>
              <Box sx={{ height: 'calc(100% - 60px)', px: 2, pb: 2 }}>
                <MapContainer
                  center={[52.52, 13.405]}
                  zoom={12}
                  style={{ height: '100%', width: '100%', borderRadius: 12 }}
                >
                  <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                  />
                  {trips.filter(t => t.status === 'in_progress').map((trip) => (
                    <Marker key={trip.id} position={[trip.pickup.lat, trip.pickup.lng]}>
                      <Popup>
                        <Typography variant="body2">{trip.riderName}</Typography>
                        <Typography variant="caption">{trip.pickup.address}</Typography>
                      </Popup>
                    </Marker>
                  ))}
                </MapContainer>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Trip Detail Dialog */}
      <Dialog open={detailOpen} onClose={() => setDetailOpen(false)} maxWidth="md" fullWidth>
        {selectedTrip && (
          <>
            <DialogTitle>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="h6">Fahrt Details</Typography>
                {getStatusChip(selectedTrip.status)}
              </Box>
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Fahrgast
                  </Typography>
                  <Typography>{selectedTrip.riderName}</Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Fahrer
                  </Typography>
                  <Typography>{selectedTrip.driverName || 'Nicht zugewiesen'}</Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Route
                  </Typography>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LocationOn color="success" />
                      <Typography>{selectedTrip.pickup.address}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LocationOn color="error" />
                      <Typography>{selectedTrip.dropoff.address}</Typography>
                    </Box>
                  </Box>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Preisdetails
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 4 }}>
                    <Box>
                      <Typography variant="body2" color="text.secondary">Grundpreis</Typography>
                      <Typography>€{selectedTrip.fare.base.toFixed(2)}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary">Distanz</Typography>
                      <Typography>€{selectedTrip.fare.distance.toFixed(2)}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary">Zeit</Typography>
                      <Typography>€{selectedTrip.fare.time.toFixed(2)}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary">Gesamt</Typography>
                      <Typography fontWeight={600}>€{selectedTrip.fare.total.toFixed(2)}</Typography>
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
