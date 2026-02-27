// GruenFahrt Admin Dashboard Types

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  role: 'admin' | 'super_admin';
  isActive: boolean;
  createdAt: string;
  lastLoginAt?: string;
}

export interface Driver {
  id: string;
  userId: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  status: 'pending' | 'approved' | 'rejected' | 'suspended';
  pScheinStatus: 'pending' | 'verified' | 'expired' | 'rejected';
  pScheinNumber?: string;
  pScheinExpiryDate?: string;
  licenseNumber: string;
  licenseExpiryDate: string;
  city: string;
  rating: number;
  totalTrips: number;
  earnings: number;
  isOnline: boolean;
  currentLocation?: GeoLocation;
  documents: DriverDocument[];
  vehicleIds: string[];
  createdAt: string;
  updatedAt: string;
}

export interface DriverDocument {
  id: string;
  type: 'license' | 'p_schein' | 'insurance' | 'vehicle_registration' | 'background_check' | 'photo';
  status: 'pending' | 'approved' | 'rejected';
  fileUrl: string;
  fileName: string;
  uploadedAt: string;
  reviewedAt?: string;
  reviewNotes?: string;
}

export interface Rider {
  id: string;
  userId: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  status: 'active' | 'suspended' | 'banned';
  rating: number;
  totalTrips: number;
  totalSpent: number;
  paymentMethods: PaymentMethod[];
  createdAt: string;
  lastRideAt?: string;
}

export interface PaymentMethod {
  id: string;
  type: 'card' | 'paypal' | 'sepa';
  last4?: string;
  isDefault: boolean;
}

export interface Vehicle {
  id: string;
  driverId: string;
  make: string;
  model: string;
  year: number;
  color: string;
  licensePlate: string;
  vin: string;
  type: 'standard' | 'xl' | 'premium' | 'electric';
  status: 'active' | 'inactive' | 'maintenance';
  insuranceExpiry: string;
  tuvExpiry: string;
  documents: VehicleDocument[];
  createdAt: string;
  updatedAt: string;
}

export interface VehicleDocument {
  id: string;
  type: 'registration' | 'insurance' | 'tuv';
  status: 'valid' | 'expired' | 'pending';
  fileUrl: string;
  expiryDate: string;
}

export interface Trip {
  id: string;
  riderId: string;
  riderName: string;
  driverId?: string;
  driverName?: string;
  vehicleId?: string;
  status: 'requested' | 'accepted' | 'in_progress' | 'completed' | 'cancelled';
  pickup: Location;
  dropoff: Location;
  route?: GeoLocation[];
  estimatedDistance: number;
  estimatedDuration: number;
  actualDistance?: number;
  actualDuration?: number;
  fare: Fare;
  paymentStatus: 'pending' | 'completed' | 'failed' | 'refunded';
  rating?: number;
  feedback?: string;
  cancellationReason?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
}

export interface Location {
  address: string;
  lat: number;
  lng: number;
}

export interface GeoLocation {
  lat: number;
  lng: number;
  timestamp: string;
}

export interface Fare {
  base: number;
  distance: number;
  time: number;
  surge?: number;
  discount?: number;
  total: number;
  currency: string;
}

export interface AnalyticsSummary {
  totalRevenue: number;
  totalTrips: number;
  activeDrivers: number;
  activeRiders: number;
  averageTripValue: number;
  completionRate: number;
  period: 'day' | 'week' | 'month' | 'year';
}

export interface RevenueData {
  date: string;
  revenue: number;
  trips: number;
  commission: number;
}

export interface ComplianceReport {
  id: string;
  type: 'driver_verification' | 'vehicle_inspection' | 'data_privacy' | 'financial_audit';
  status: 'open' | 'in_progress' | 'resolved';
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  assignedTo?: string;
  relatedEntityId?: string;
  relatedEntityType?: 'driver' | 'vehicle' | 'trip';
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
}

export interface PlatformSettings {
  baseFare: number;
  perKmRate: number;
  perMinuteRate: number;
  minimumFare: number;
  cancellationFee: number;
  commissionRate: number;
  surgePricingEnabled: boolean;
  maxSurgeMultiplier: number;
  driverPayoutSchedule: 'daily' | 'weekly' | 'biweekly' | 'monthly';
  supportEmail: string;
  supportPhone: string;
}

export interface Notification {
  id: string;
  type: 'info' | 'warning' | 'error' | 'success';
  title: string;
  message: string;
  isRead: boolean;
  createdAt: string;
}

export interface DashboardStats {
  todayRevenue: number;
  todayTrips: number;
  activeTrips: number;
  onlineDrivers: number;
  pendingVerifications: number;
  openSupportTickets: number;
}
