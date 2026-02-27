import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../core/theme/app_theme.dart';
import '../providers/auth_provider.dart';
import '../models/driver_model.dart';

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  @override
  void initState() {
    super.initState();
    context.read<AuthBloc>().add(const LoadDriverProfile());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: BlocBuilder<AuthBloc, AuthState>(
        builder: (context, state) {
          if (state is AuthLoading) {
            return const Center(child: CircularProgressIndicator());
          }

          if (state is AuthAuthenticated && state.driver != null) {
            return _buildProfileContent(state.driver!);
          }

          return const Center(child: Text('Fehler beim Laden des Profils'));
        },
      ),
    );
  }

  Widget _buildProfileContent(Driver driver) {
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildProfileHeader(driver),
            const SizedBox(height: 24),
            _buildVerificationStatus(driver),
            const SizedBox(height: 24),
            _buildVehicleInfo(driver),
            const SizedBox(height: 24),
            _buildStatistics(driver),
            const SizedBox(height: 24),
            _buildActionButtons(),
          ],
        ),
      ),
    );
  }

  Widget _buildProfileHeader(Driver driver) {
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: AppTheme.primaryGreen,
        borderRadius: BorderRadius.circular(16),
        gradient: LinearGradient(
          colors: [
            AppTheme.primaryGreen,
            AppTheme.primaryGreen.withOpacity(0.8),
          ],
          begin: Alignment.centerLeft,
          end: Alignment.centerRight,
        ),
      ),
      child: Column(
        children: [
          Center(
            child: Container(
              width: 100,
              height: 100,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: Colors.white,
                border: Border.all(color: Colors.white, width: 4),
              ),
              child: ClipOVal(
                borderRadius: BorderRadius.circular(50),
                child: driver.profileImageUrl != null
                    ? Image.network(driver.profileImageUrl!)
                    : Center(
                        child: Text(
                          driver.firstName[0],
                          style: const TextStyle(
                            color: AppTheme.primaryGreen,
                            fontSize: 40,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Center(
            child: Text(
              '${driver.firstName} ${driver.lastName}',
              style: const TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          Center(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.2),
                borderRadius: BorderRadius.circular(20),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.star, color: Colors.amber, size: 18),
                  const SizedBox(width: 4),
                  Text(
                    '\\${driver.rating}',
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.wv600,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    '(${driver.totalTrips} Fahrten)',
                    style: TextStyle(
                      color: Colors.white.withOpacity(0.8),
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildVerificationStatus(Driver driver) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Verifiierungsstatus',
          style: const TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 16),
        _buildVerificationItem(
          icon: Icons.badge,
          title: 'Führerschein',
          status: driver.isVerified ? VerificationStatus.verified : VerificationStatus.pending,
        ),
        _buildVerificationItem(
          icon: Icons.credit_card,
          title: 'Fahrzeugnis',
          status: VerificationStatus.verified,
        ),
        _buildVerificationItem(
          icon: Icons.local_activity,
          title: 'Fahrtugaung',
          status: driver.isVerified ? VerificationStatus.verified : VerificationStatus.pending,
        ),
        _buildVerificationItem(
          icon: Icons.directions_car,
          title: 'Fahrzeug'ung',
          status: driver.vehicle.licensePlate != null && driver.vehicle.licensePlate!.isNotEmpty
              ? VerificationStatus.verified
              : VerificationStatus.pending,
        ),
      ],
    );
  }

  Widget _buildVerificationItem({
    required IconData icon,
    required String title,
    required VerificationStatus status,
  }) {
    final color = status == VerificationStatus.verified
        ? AppTheme.success
        : status == VerificationStatus.pending
            ? Colors.orange
            : AppTheme.error;

    final statusText = status == VerificationStatus.verified
        ? 'Verifiziert'
        : status == VerificationStatus.pending
            ? 'Ausstehend'
            : 'Fahlgeschlagen';

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.grey[50],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: color.withOpacity(0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(icon, color: color, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              title,
              style: const TextStyle(
                fontWeight: FontWeight.w500,
                fontSize: 16,
              ),
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: color.withOpacity(0.1),
              borderRadius: BorderRadius.circular(16),
            ),
            child: Text(
              statusText,
              style: TextStyle(
                color: color,
                fontWeight: FontWeight.w600,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildVehicleInfo(Driver driver) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Fahrzeug',
          style: const TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 16),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: Colors.grey[50],
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildInfoRow('Marke', driver.vehicle.make),
              const SizedBox(height: 8),
              _buildInfoRow('Modell', driver.vehicle.model),
              const SizedBox(height: 8),
              _buildInfoRow('Farbe', driver.vehicle.color),
              const SizedBox(height: 8),
              _buildInfoRow('Kennzeichen', driver.vehicle.licensePlate ?? 'Nicht hinzugügt'),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          label,
          style: TextStyle(
            color: Colors.grey[600],
            fontSize: 14,
          ),
        ),
        Text(
          value,
          style: const TextStyle(
            fontWeight: FontWeight.w500,
            fontSize: 16,
          ),
        ),
      ],
    );
  }

  Widget _buildStatistics(Driver driver) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Statistiken',
          style: const TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            _buildStatCard('${driver.totalTrips}', 'Fahrten gesamt'),
            const SizedBox(width: 16),
            _buildStatCard('${driver.rating.toStringAsFixed(1)}', 'Durchschnitt'),
            const SizedBox(width: 16),
            _buildStatCard('${driver.joinedDate.year}', 'Mitglied seit'),
          ],
        ),
      ],
    );
  }

  Widget _buildStatCard(String value, String label) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.grey[50],
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          children: [
            Text(
              value,
              style: const TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              label,
              style: TextStyle(
                color: Colors.grey[600],
                fontSize: 12,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButtons() {
    return Column(
      children: [
        ElevatedButton.icon(
          onPressed: () {
            // Todo Navigate to edit profile
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Profil bearbeiten')),
            );
          },
          icon: const Icon(Icons.edit),
          label: const Text('Profil bearbeiten'),
          style: ElevatedButton.styleFrom(
            backgroundColor: AppTheme.primaryGreen,
            foregroundColor: Colors.white,
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
          ),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          onPressed: () {
            context.read<AuthBloc>().add(const Logout());
          },
          icon: const Icon(Icons.logout),
          label: const Text('Abmelden'),
          style: OutlinedButton.styleFrom(
            sideBackgroundColor: Colors.white,
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
          ),
        ),
      ],
    );
  }
}

enum VerificationStatus { verified, pending, failed }
