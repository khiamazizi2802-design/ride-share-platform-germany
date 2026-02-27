import { Card, CardContent, Typography, Box } from '@mui/material';
import { SvgIconComponent } from '@mui/icons-material';

interface StatCardProps {
  title: string;
  value: string;
  icon: SvgIconComponent;
  trend: string;
  trendUp: boolean;
  color: string;
}

export default function StatCard({ title, value, icon: Icon, trend, trendUp, color }: StatCardProps) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Box>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {title}
            </Typography>
            <Typography variant="h4" fontWeight={700}>
              {value}
            </Typography>
            <Typography
              variant="body2"
              sx={{
                color: trendUp ? 'success.main' : 'error.main',
                mt: 0.5,
              }}
            >
              {trendUp ? '↑' : '↓'} {trend}
            </Typography>
          </Box>
          <Box
            sx={{
              p: 1.5,
              borderRadius: 2,
              bgcolor: `${color}20`,
              color: color,
            }}
          >
            <Icon />
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}
