import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Box,
  Drawer,
  AppBar,
  Toolbar,
  List,
  Typography,
  Divider,
  IconButton,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Avatar,
  Menu,
  MenuItem,
  Badge,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Dashboard,
  People,
  DirectionsCar,
  LocalTaxi,
  Assessment,
  Gavel,
  Settings,
  ChevronLeft,
  Notifications,
  AccountCircle,
  Logout,
} from '@mui/icons-material';
import { useAuthStore } from '@/store/authStore';
import { useDashboardStore } from '@/store/dashboardStore';

const drawerWidth = 280;

const menuItems = [
  { path: '/', label: 'Dashboard', icon: Dashboard },
  { path: '/drivers', label: 'Fahrer', icon: DirectionsCar },
  { path: '/riders', label: 'Fahrgäste', icon: People },
  { path: '/trips', label: 'Fahrten', icon: LocalTaxi },
  { path: '/vehicles', label: 'Fahrzeuge', icon: DirectionsCar },
  { path: '/analytics', label: 'Analysen', icon: Assessment },
  { path: '/compliance', label: 'Compliance', icon: Gavel },
  { path: '/settings', label: 'Einstellungen', icon: Settings },
];

export default function MainLayout() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { isSidebarOpen, toggleSidebar, setSidebarOpen, notifications } = useDashboardStore();
  
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [notificationAnchor, setNotificationAnchor] = useState<null | HTMLElement>(null);

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleNotificationOpen = (event: React.MouseEvent<HTMLElement>) => {
    setNotificationAnchor(event.currentTarget);
  };

  const handleNotificationClose = () => {
    setNotificationAnchor(null);
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const unreadCount = notifications.filter((n) => !n.isRead).length;

  const drawer = (
    <Box>
      <Toolbar
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          px: [1],
          background: 'linear-gradient(135deg, #22C55E 0%, #16A34A 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar
            src="/gruenfahrt-icon.svg"
            sx={{ width: 40, height: 40, bgcolor: 'white' }}
          >
            G
          </Avatar>
          <Typography variant="h6" sx={{ color: 'white', fontWeight: 700 }}>
            GruenFahrt
          </Typography>
        </Box>
        {isMobile && (
          <IconButton onClick={() => setSidebarOpen(false)} sx={{ color: 'white' }}>
            <ChevronLeft />
          </IconButton>
        )}
      </Toolbar>
      <Divider />
      <List sx={{ pt: 2 }}>
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          return (
            <ListItem key={item.path} disablePadding>
              <ListItemButton
                onClick={() => {
                  navigate(item.path);
                  if (isMobile) setSidebarOpen(false);
                }}
                sx={{
                  mx: 1,
                  borderRadius: 2,
                  mb: 0.5,
                  backgroundColor: isActive ? 'rgba(34, 197, 94, 0.1)' : 'transparent',
                  color: isActive ? '#16A34A' : 'inherit',
                  '&:hover': {
                    backgroundColor: isActive ? 'rgba(34, 197, 94, 0.15)' : 'rgba(0, 0, 0, 0.04)',
                  },
                }}
              >
                <ListItemIcon sx={{ color: isActive ? '#16A34A' : 'inherit', minWidth: 40 }}>
                  <Icon />
                </ListItemIcon>
                <ListItemText
                  primary={item.label}
                  primaryTypographyProps={{ fontWeight: isActive ? 600 : 400 }}
                />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <AppBar
        position="fixed"
        sx={{
          width: { md: `calc(100% - ${drawerWidth}px)` },
          ml: { md: `${drawerWidth}px` },
          bgcolor: 'white',
          color: 'text.primary',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            onClick={toggleSidebar}
            sx={{ mr: 2, display: { md: 'none' } }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 600 }}>
            {menuItems.find((item) => item.path === location.pathname)?.label || 'Dashboard'}
          </Typography>
          <IconButton color="inherit" onClick={handleNotificationOpen}>
            <Badge badgeContent={unreadCount} color="error">
              <Notifications />
            </Badge>
          </IconButton>
          <IconButton color="inherit" onClick={handleMenuOpen} sx={{ ml: 1 }}>
            <Avatar sx={{ width: 32, height: 32, bgcolor: '#22C55E' }}>
              {user?.firstName?.[0] || 'A'}
            </Avatar>
          </IconButton>
        </Toolbar>
      </AppBar>

      {/* Notifications Menu */}
      <Menu
        anchorEl={notificationAnchor}
        open={Boolean(notificationAnchor)}
        onClose={handleNotificationClose}
        PaperProps={{ sx: { width: 360, maxHeight: 400 } }}
      >
        {notifications.length === 0 ? (
          <MenuItem disabled>Keine Benachrichtigungen</MenuItem>
        ) : (
          notifications.map((notification) => (
            <MenuItem key={notification.id} onClick={handleNotificationClose}>
              <Box>
                <Typography variant="subtitle2">{notification.title}</Typography>
                <Typography variant="body2" color="text.secondary" noWrap>
                  {notification.message}
                </Typography>
              </Box>
            </MenuItem>
          ))
        )}
      </Menu>

      {/* User Menu */}
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleMenuClose}
        PaperProps={{ sx: { width: 200 } }}
      >
        <MenuItem onClick={() => { handleMenuClose(); navigate('/settings'); }}>
          <ListItemIcon>
            <AccountCircle fontSize="small" />
          </ListItemIcon>
          Profil
        </MenuItem>
        <Divider />
        <MenuItem onClick={handleLogout}>
          <ListItemIcon>
            <Logout fontSize="small" />
          </ListItemIcon>
          Abmelden
        </MenuItem>
      </Menu>

      {/* Sidebar */}
      <Box
        component="nav"
        sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}
      >
        <Drawer
          variant={isMobile ? 'temporary' : 'permanent'}
          open={isMobile ? isSidebarOpen : true}
          onClose={() => setSidebarOpen(false)}
          ModalProps={{ keepMounted: true }}
          sx={{
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
              borderRight: '1px solid #E2E8F0',
            },
          }}
        >
          {drawer}
        </Drawer>
      </Box>

      {/* Main Content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: { md: `calc(100% - ${drawerWidth}px)` },
          mt: 8,
          bgcolor: '#F8FAFC',
          minHeight: '100vh',
        }}
      >
        <Outlet />
      </Box>
    </Box>
  );
}
