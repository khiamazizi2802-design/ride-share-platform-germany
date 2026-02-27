export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  role: 'rider' | 'driver' | 'admin';
  createdAt: string;
  updatedAt: string;
}

export interface Rider {
  id: string;
  userId: string;
  rating: number;
  totalTrips: number;
  preferredPaymentMethod?: string;
}

export interface Driver {
  id: string;
  userId: string;
  status: 'pending' | 'approved' | 'rejected' | 'suspended';
  rating: number;
  totalTrips: number;
  earnings: number;
  isAvailable: boolean;
  licenseNumber: string;
  licenseVerified: boolean;
  vehicleInfo: Vehicle;
  documents: DriverDocument[];
}

export interface Vehicle {
  make: string;
  model: string;
  year: number;
  color: string;
  licensePlate: string;
  seats: number;
}

export interface DriverDocument {
  id: string;
  type: 'license' | 'insurance' | 'registration' | 'background_check' | 'vehicle_photo' | 'selfie';
  status: 'pending' | 'approved' | 'rejected';
  uploadedAt: string;
  url?: string;
}

export interface Trip {
  id: string;
  riderId: string;
  driverId?: string;
  status: 'requested' | 'accepted' | 'in_progress' | 'completed' | 'cancelled';
  pickup: Location;
  destination: Location;
  fare: number;
  distance: number;
  duration: number;
  createdAt: string;
  completedAt?: string;
}

export interface Location {
  address: string;
  latitude: number;
  longitude: number;
}

export interface PaymentMethod {
  id: string;
  type: 'card' | 'paypal' | 'sepa';
  last4?: string;
  brand?: string;
  isDefault: boolean;
}

export interface EarningsSummary {
  today: number;
  thisWeek: number;
  thisMonth: number;
  total: number;
  tripsCount: number;
}

export interface DocumentUpload {
  file: File;
  type: DriverDocument['type'];
}