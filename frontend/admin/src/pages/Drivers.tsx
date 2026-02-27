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
  Avatar,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Link,
} from '@mui/material';
import {
  Search,
  FilterList,
  CheckCircle,
  Cancel,
  Visibility,
  DirectionsCar,
  Description,
  Phone,
  Email,
  LocationOn,
} from '@mui/icons-material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Driver, DriverDocument } from '@/types';
import { mockDrivers } from '@/services/mockData';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 2 }}>{children}</Box>}
    </div>
  );
}

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'success' | 'warning' | 'error' | 'default' }> = {
    approved: { label: 'Genehmigt', color: 'success' },
    pending: { label: 'Ausstehend', color: 'warning' },
    rejected: { label: 'Abgelehnt', color: 'error' },
    suspended: { label: 'Gesperrt', color: 'error' },
    verified: { label: 'Verifiziert', color: 'success' },
    expired: { label: 'Abgelaufen', color: 'error' },
  };
  const config = statusMap[status] || { label: status, color: 'default' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

export default function Drivers() {
  const [drivers] = useState<Driver[]>(mockDrivers);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDriver, setSelectedDriver] = useState<Driver | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const handleViewDriver = (driver: Driver) => {
    setSelectedDriver(driver);
    setDetailOpen(true);
    setActiveTab(0);
  };

  const handleApprove = (driverId: string) => {
    console.log('Approving driver:', driverId);
    // API call to approve driver
  };

  const handleReject = (driverId: string) => {
    console.log('Rejecting driver:', driverId);
    // API call to reject driver
  };

  const filteredDrivers = drivers.filter((driver) => {
    const matchesSearch =
      driver.firstName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      driver.lastName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      driver.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      driver.city.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = filterStatus === 'all' || driver.status === filterStatus;
    return matchesSearch && matchesStatus;
  });

  const columns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'Fahrer',
      flex: 2,
      renderCell: (params: GridRenderCellParams) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Avatar sx={{ bgcolor: '#22C55E' }}>
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
      field: 'pScheinStatus',
      headerName: 'P-Schein',
      flex: 1,
      renderCell: (params) => getStatusChip(params.value),
    },
    {
      field: 'city',
      headerName: 'Stadt',
      flex: 1,
    },
    {
      field: 'rating',
      headerName: 'Bewertung',
      flex: 0.8,
      renderCell: (params) => (
        <Typography variant="body2" fontWeight={600}>
          {params.value > 0 ? `⭐ ${params.value}` : '-'}
        </Typography>
      ),
    },
    {
      field: 'totalTrips',
      headerName: 'Fahrten',
      flex: 0.8,
    },
    {
      field: 'isOnline',
      headerName: 'Online',
      flex: 0.8,
      renderCell: (params) => (
        <Chip
          label={params.value ? 'Online' : 'Offline'}
          color={params.value ? 'success' : 'default'}
          size="small"
          variant={params.value ? 'filled' : 'outlined'}
        />
      ),
    },
    {
      field: 'actions',
      headerName: 'Aktionen',
      flex: 1.5,
      sortable: false,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <IconButton size="small" onClick={() => handleViewDriver(params.row)}>
            <Visibility fontSize="small" />
          </IconButton>
          {params.row.status === 'pending' && (
            <>
              <IconButton
                size="small"
                color="success"
                onClick={() => handleApprove(params.row.id)}
              >
                <CheckCircle fontSize="small" />
              </IconButton>
              <IconButton
                size="small"
                color="error"
                onClick={() => handleReject(params.row.id)}
              >
                <Cancel fontSize="small" />
              </IconButton>
            </>
          )}
        </Box>
      ),
    },
  ];

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Fahrerverwaltung
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <TextField
              placeholder="Fahrer suchen..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <Search />
                  </InputAdornment>
                ),
              }}
              sx={{ flex: 1, minWidth: 250 }}
            />
            <Button
              variant="outlined"
              startIcon={<FilterList />}
              onClick={() => setFilterStatus(filterStatus === 'all' ? 'pending' : 'all')}
            >
              {filterStatus === 'all' ? 'Alle' : 'Ausstehend'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card>
        <DataGrid
          rows={filteredDrivers}
          columns={columns}
          initialState={{
            pagination: { paginationModel: { pageSize: 10 } },
          }}
          pageSizeOptions={[10, 25, 50]}
          disableRowSelectionOnClick
          autoHeight
          sx={{
            border: 'none',
            '& .MuiDataGrid-cell:focus': { outline: 'none' },
          }}
          getRowId={(row) => row.id}
        />
      </Card>

      {/* Driver Detail Dialog */}
      <Dialog
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        maxWidth="md"
        fullWidth
      >
        {selectedDriver && (
          <>
            <DialogTitle>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ width: 56, height: 56, bgcolor: '#22C55E', fontSize: 24 }}>
                  {selectedDriver.firstName[0]}{selectedDriver.lastName[0]}
                </Avatar>
                <Box>
                  <Typography variant="h6">
                    {selectedDriver.firstName} {selectedDriver.lastName}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, mt: 0.5 }}>
                    {getStatusChip(selectedDriver.status)}
                    {getStatusChip(selectedDriver.pScheinStatus)}
                  </Box>
                </Box>
              </Box>
            </DialogTitle>
            <DialogContent>
              <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ mb: 2 }}>
                <Tab label="Übersicht" />
                <Tab label="Dokumente" />
                <Tab label="Fahrzeuge" />
              </Tabs>

              <TabPanel value={activeTab} index={0}>
                <Grid container spacing={3}>
                  <Grid item xs={12} md={6}>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      Kontaktinformationen
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Email fontSize="small" color="action" />
                        <Typography>{selectedDriver.email}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Phone fontSize="small" color="action" />
                        <Typography>{selectedDriver.phone}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <LocationOn fontSize="small" color="action" />
                        <Typography>{selectedDriver.city}</Typography>
                      </Box>
                    </Box>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      Führerschein
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                      <Typography>Nr: {selectedDriver.licenseNumber}</Typography>
                      <Typography>Gültig bis: {selectedDriver.licenseExpiryDate}</Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      P-Schein
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                      <Typography>Nr: {selectedDriver.pScheinNumber || 'N/A'}</Typography>
                      <Typography>Gültig bis: {selectedDriver.pScheinExpiryDate || 'N/A'}</Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      Statistiken
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                      <Typography>Gesamtfahrten: {selectedDriver.totalTrips}</Typography>
                      <Typography>Bewertung: {selectedDriver.rating > 0 ? selectedDriver.rating : '-'}/5</Typography>
                      <Typography>Gesamteinnahmen: €{selectedDriver.earnings.toFixed(2)}</Typography>
                    </Box>
                  </Grid>
                </Grid>
              </TabPanel>

              <TabPanel value={activeTab} index={1}>
                <TableContainer component={Paper} variant="outlined">
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell>Dokumenttyp</TableCell>
                        <TableCell>Status</TableCell>
                        <TableCell>Hochgeladen</TableCell>
                        <TableCell>Aktionen</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {selectedDriver.documents.map((doc: DriverDocument) => (
                        <TableRow key={doc.id}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Description fontSize="small" />
                              {doc.type === 'license' && 'Führerschein'}
                              {doc.type === 'p_schein' && 'P-Schein'}
                              {doc.type === 'insurance' && 'Versicherung'}
                              {doc.type === 'background_check' && 'Führungszeugnis'}
                            </Box>
                          </TableCell>
                          <TableCell>{getStatusChip(doc.status)}</TableCell>
                          <TableCell>{new Date(doc.uploadedAt).toLocaleDateString('de-DE')}</TableCell>
                          <TableCell>
                            <Link href={doc.fileUrl} target="_blank" sx={{ mr: 2 }}>
                              Ansehen
                            </Link>
                            {doc.status === 'pending' && (
                              <>
                                <Button size="small" color="success" sx={{ mr: 1 }}>
                                  Genehmigen
                                </Button>
                                <Button size="small" color="error">
                                  Ablehnen
                                </Button>
                              </>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </TabPanel>

              <TabPanel value={activeTab} index={2}>
                <Typography color="text.secondary">
                  Fahrzeuge werden vom Fahrzeug-Service geladen...
                </Typography>
              </TabPanel>
            </DialogContent>
            <DialogActions>
              {selectedDriver.status === 'pending' && (
                <>
                  <Button onClick={() => handleReject(selectedDriver.id)} color="error">
                    Ablehnen
                  </Button>
                  <Button onClick={() => handleApprove(selectedDriver.id)} variant="contained" color="success">
                    Genehmigen
                  </Button>
                </>
              )}
              <Button onClick={() => setDetailOpen(false)}>Schließen</Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
}
