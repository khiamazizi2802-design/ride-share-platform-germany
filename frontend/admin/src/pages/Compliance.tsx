import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  TextField,
  MenuItem,
} from '@mui/material';
import {
  Visibility,
  CheckCircle,
  Warning,
  Error,
  Gavel,
  Assignment,
} from '@mui/icons-material';
import { ComplianceReport } from '@/types';
import { mockComplianceReports } from '@/services/mockData';

const getSeverityChip = (severity: string) => {
  const severityMap: Record<string, { label: string; color: 'error' | 'warning' | 'info' | 'success' }> = {
    critical: { label: 'Kritisch', color: 'error' },
    high: { label: 'Hoch', color: 'error' },
    medium: { label: 'Mittel', color: 'warning' },
    low: { label: 'Niedrig', color: 'info' },
  };
  const config = severityMap[severity] || { label: severity, color: 'info' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

const getStatusChip = (status: string) => {
  const statusMap: Record<string, { label: string; color: 'warning' | 'info' | 'success' }> = {
    open: { label: 'Offen', color: 'warning' },
    in_progress: { label: 'In Bearbeitung', color: 'info' },
    resolved: { label: 'Gelöst', color: 'success' },
  };
  const config = statusMap[status] || { label: status, color: 'warning' };
  return <Chip label={config.label} color={config.color} size="small" />;
};

const getTypeIcon = (type: string) => {
  switch (type) {
    case 'driver_verification':
      return <Assignment fontSize="small" />;
    case 'vehicle_inspection':
      return <Warning fontSize="small" />;
    case 'data_privacy':
      return <Gavel fontSize="small" />;
    case 'financial_audit':
      return <Error fontSize="small" />;
    default:
      return <Assignment fontSize="small" />;
  }
};

const getTypeLabel = (type: string) => {
  const typeMap: Record<string, string> = {
    driver_verification: 'Fahrerverifizierung',
    vehicle_inspection: 'Fahrzeugprüfung',
    data_privacy: 'Datenschutz',
    financial_audit: 'Finanzprüfung',
  };
  return typeMap[type] || type;
};

export default function Compliance() {
  const [reports] = useState<ComplianceReport[]>(mockComplianceReports);
  const [selectedReport, setSelectedReport] = useState<ComplianceReport | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [filterType, setFilterType] = useState<string>('all');

  const handleViewReport = (report: ComplianceReport) => {
    setSelectedReport(report);
    setDetailOpen(true);
  };

  const filteredReports = filterType === 'all' 
    ? reports 
    : reports.filter(r => r.type === filterType);

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Compliance & Berichte
      </Typography>

      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ p: 1.5, bgcolor: 'error.light', borderRadius: 2, color: 'error.dark' }}>
                  <Error />
                </Box>
                <Box>
                  <Typography variant="h5" fontWeight={600}>
                    {reports.filter(r => r.severity === 'critical' || r.severity === 'high').length}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">Kritische Meldungen</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ p: 1.5, bgcolor: 'warning.light', borderRadius: 2, color: 'warning.dark' }}>
                  <Warning />
                </Box>
                <Box>
                  <Typography variant="h5" fontWeight={600}>
                    {reports.filter(r => r.status === 'open').length}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">Offene Meldungen</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ p: 1.5, bgcolor: 'info.light', borderRadius: 2, color: 'info.dark' }}>
                  <Assignment />
                </Box>
                <Box>
                  <Typography variant="h5" fontWeight={600}>
                    {reports.filter(r => r.status === 'in_progress').length}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">In Bearbeitung</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ p: 1.5, bgcolor: 'success.light', borderRadius: 2, color: 'success.dark' }}>
                  <CheckCircle />
                </Box>
                <Box>
                  <Typography variant="h5" fontWeight={600}>
                    {reports.filter(r => r.status === 'resolved').length}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">Gelöst</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <TextField
              select
              label="Filter nach Typ"
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              sx={{ width: 200 }}
              size="small"
            >
              <MenuItem value="all">Alle</MenuItem>
              <MenuItem value="driver_verification">Fahrerverifizierung</MenuItem>
              <MenuItem value="vehicle_inspection">Fahrzeugprüfung</MenuItem>
              <MenuItem value="data_privacy">Datenschutz</MenuItem>
              <MenuItem value="financial_audit">Finanzprüfung</MenuItem>
            </TextField>
            <Button variant="contained" color="primary">
              Neuer Bericht
            </Button>
          </Box>

          <TableContainer component={Paper} variant="outlined">
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Typ</TableCell>
                  <TableCell>Titel</TableCell>
                  <TableCell>Schwere</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Erstellt</TableCell>
                  <TableCell>Aktionen</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredReports.map((report) => (
                  <TableRow key={report.id}>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        {getTypeIcon(report.type)}
                        {getTypeLabel(report.type)}
                      </Box>
                    </TableCell>
                    <TableCell>{report.title}</TableCell>
                    <TableCell>{getSeverityChip(report.severity)}</TableCell>
                    <TableCell>{getStatusChip(report.status)}</TableCell>
                    <TableCell>{new Date(report.createdAt).toLocaleDateString('de-DE')}</TableCell>
                    <TableCell>
                      <IconButton size="small" onClick={() => handleViewReport(report)}>
                        <Visibility fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      <Dialog open={detailOpen} onClose={() => setDetailOpen(false)} maxWidth="sm" fullWidth>
        {selectedReport && (
          <>
            <DialogTitle>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {getTypeIcon(selectedReport.type)}
                {selectedReport.title}
              </Box>
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Typ</Typography>
                  <Typography>{getTypeLabel(selectedReport.type)}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Schwere</Typography>
                  {getSeverityChip(selectedReport.severity)}
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Status</Typography>
                  {getStatusChip(selectedReport.status)}
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary">Zugewiesen an</Typography>
                  <Typography>{selectedReport.assignedTo || 'Nicht zugewiesen'}</Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="body2" color="text.secondary">Beschreibung</Typography>
                  <Typography>{selectedReport.description}</Typography>
                </Grid>
                {selectedReport.relatedEntityId && (
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary">
                      Verknüpfte Entität: {selectedReport.relatedEntityType} #{selectedReport.relatedEntityId.slice(-6)}
                    </Typography>
                  </Grid>
                )}
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setDetailOpen(false)}>Schließen</Button>
              {selectedReport.status !== 'resolved' && (
                <Button variant="contained" color="success">
                  Als gelöst markieren
                </Button>
              )}
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
}
