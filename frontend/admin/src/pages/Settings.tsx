import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Divider,
  Grid,
  Switch,
  FormControlLabel,
  Tabs,
  Tab,
} from '@mui/material';
import { Save, Email, Phone, Payment } from '@mui/icons-material';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

export default function Settings() {
  const [activeTab, setActiveTab] = useState(0);
  const [settings, setSettings] = useState({
    baseFare: 3.50,
    perKmRate: 1.80,
    perMinuteRate: 0.35,
    minimumFare: 5.00,
    cancellationFee: 4.00,
    commissionRate: 15,
    surgePricingEnabled: true,
    maxSurgeMultiplier: 2.5,
    driverPayoutSchedule: 'weekly',
    supportEmail: 'support@gruenfahrt.de',
    supportPhone: '+49 800 1234567',
  });

  const handleSave = () => {
    console.log('Saving settings:', settings);
    // API call to save settings
  };

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3, fontWeight: 700 }}>
        Plattform-Einstellungen
      </Typography>

      <Card>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label="Preise" />
          <Tab label="Fahrer" />
          <Tab label="Support" />
          <Tab label="Allgemein" />
        </Tabs>

        <CardContent>
          <TabPanel value={activeTab} index={0}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Grundpreis (€)"
                  type="number"
                  value={settings.baseFare}
                  onChange={(e) => setSettings({ ...settings, baseFare: parseFloat(e.target.value) })}
                  inputProps={{ step: 0.1 }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Preis pro km (€)"
                  type="number"
                  value={settings.perKmRate}
                  onChange={(e) => setSettings({ ...settings, perKmRate: parseFloat(e.target.value) })}
                  inputProps={{ step: 0.1 }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Preis pro Minute (€)"
                  type="number"
                  value={settings.perMinuteRate}
                  onChange={(e) => setSettings({ ...settings, perMinuteRate: parseFloat(e.target.value) })}
                  inputProps={{ step: 0.05 }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Mindestpreis (€)"
                  type="number"
                  value={settings.minimumFare}
                  onChange={(e) => setSettings({ ...settings, minimumFare: parseFloat(e.target.value) })}
                  inputProps={{ step: 0.1 }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Stornierungsgebühr (€)"
                  type="number"
                  value={settings.cancellationFee}
                  onChange={(e) => setSettings({ ...settings, cancellationFee: parseFloat(e.target.value) })}
                  inputProps={{ step: 0.1 }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Provision (%)"
                  type="number"
                  value={settings.commissionRate}
                  onChange={(e) => setSettings({ ...settings, commissionRate: parseInt(e.target.value) })}
                  inputProps={{ min: 0, max: 100 }}
                />
              </Grid>
              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={settings.surgePricingEnabled}
                      onChange={(e) => setSettings({ ...settings, surgePricingEnabled: e.target.checked })}
                    />
                  }
                  label="Surge Pricing aktivieren"
                />
              </Grid>
              {settings.surgePricingEnabled && (
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Maximaler Surge-Multiplikator"
                    type="number"
                    value={settings.maxSurgeMultiplier}
                    onChange={(e) => setSettings({ ...settings, maxSurgeMultiplier: parseFloat(e.target.value) })}
                    inputProps={{ step: 0.1, min: 1, max: 5 }}
                  />
                </Grid>
              )}
            </Grid>
          </TabPanel>

          <TabPanel value={activeTab} index={1}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  select
                  label="Fahrer-Auszahlungsrhythmus"
                  value={settings.driverPayoutSchedule}
                  onChange={(e) => setSettings({ ...settings, driverPayoutSchedule: e.target.value })}
                  SelectProps={{ native: true }}
                >
                  <option value="daily">Täglich</option>
                  <option value="weekly">Wöchentlich</option>
                  <option value="biweekly">Zweiwöchentlich</option>
                  <option value="monthly">Monatlich</option>
                </TextField>
              </Grid>
            </Grid>
          </TabPanel>

          <TabPanel value={activeTab} index={2}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Support E-Mail"
                  value={settings.supportEmail}
                  onChange={(e) => setSettings({ ...settings, supportEmail: e.target.value })}
                  InputProps={{ startAdornment: <Email color="action" sx={{ mr: 1 }} /> }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Support Telefon"
                  value={settings.supportPhone}
                  onChange={(e) => setSettings({ ...settings, supportPhone: e.target.value })}
                  InputProps={{ startAdornment: <Phone color="action" sx={{ mr: 1 }} /> }}
                />
              </Grid>
            </Grid>
          </TabPanel>

          <TabPanel value={activeTab} index={3}>
            <Typography color="text.secondary">
              Allgemeine Plattformeinstellungen werden hier angezeigt.
            </Typography>
          </TabPanel>

          <Divider sx={{ my: 3 }} />

          <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="contained"
              color="primary"
              startIcon={<Save />}
              onClick={handleSave}
            >
              Einstellungen speichern
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}
