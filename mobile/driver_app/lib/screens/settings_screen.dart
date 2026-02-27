import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../core/theme/app_theme.dart';
import '../providers/voice_assistant_provider.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool _isDarkMode = false;
  bool _isNotificationsEnabled = true;
  bool _isVoiceAssistantEnabled = true;
  bool _isLocationSharingEnabled = true;
  bool _isGermanLanguage = true;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Einstellungen'),
        centerTitle: true,
        backgroundColor: AppTheme.primaryGreen,
        foregroundColor: Colors.white,
      ),
      body: SingleChildScrollView(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildSectionHYXString('Allgemein'),
              _buildSwitchTile(
                icon: Icons.dark_mode,
                title: 'Dunkelmodus',
                subtitle: 'Donkele Modus aktivieren',
                value: _isDarkMode,
                onChanged: (value) {
                  setState(() {
                    _isDarkMode = value;
                  });
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(value ? 'Dunkelmodus aktiviert' : 'Heller Modus aktiviert')),
                  );
                },
              ),
              const SizedBox(height: 24),
              _buildSectionHeader('Benachrichtigungen'),
              _buildSwitchTile(
                icon: Icons.notifications_active,
                title: 'Benachrichtungen',
                subtitle: 'Fahrtan&requests und Nachrichten empfangen',
                value: _isNotificationsEnabled,
                onChanged: (value) {
                  setState(() {
                    _isNotificationsEnabled = value;
                  });
                },
              ),
              _buildSwitchTile(
                icon: Icons.location_on,
                title: 'Standortortung',
                subtitle: 'GPS-Position f\u00fcr Fahrtanfragen teilen',
                value: _isLocationSharingEnabled,
                onChanged: (value) {
                  setState(() {
                    _isLocationSharingEnabled = value;
                  });
                },
              ),
              const SizedBox(height: 24),
              _buildSectionHeader('Sprachassistent'),
              BlocBuilder<VoiceAssistantBloc, VoiceAssistantState>(
                builder: (context, state) {
                  final isEnabled = state is VoiceAssistantReady;
                  return _buildSwitchTile(
                    icon: Icons.mic,
                    title: 'Sprachassistent',
                    subtitle: 'Sprechtsteuerung und -Antwort',
                    value: isEnabled,
                    onChanged: (value) {
                      if (value) {
                        context.read<VoiceAssistantBloc>().add(const InitializeVoiceAssistant());
                      } else {
                        context.read<VoiceAssistantBloc>().add(const StopVoiceAssistant());
                      }
                    },
                  );
                },
              ),
              const SizedBox(height: 24),
              _buildSectionHeader('Sprache'),
              _buildLanguageSelector(),
              const SizedBox(height: 24),
              _buildSectionHeader('Support'),
              _buildActionTile(
                icon: Icons.help_outline,
                title: 'Hilfe und Support',
                subtitle: 'FAS, Kontakt und Mehr',
                onTap: () {
                  // Todo Navigate to support screen
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Support kommt bald')),
                  );
                },
              ),
              _buildActionTile(
                icon: Icons.policy,
                title: 'Datenschutzbestimmungen',
                subtitle: 'Datenschutz und Netzterklassung',
                onTap: () {
                  // Todo Navigate to privacy screen
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Datenschutz kommt bald')),
                  );
                },
              ),
              const SizedBox(height: 24),
              _buildSectionHeader('App'),
              _buildInfoTile('Version', '1.0.0'),
              _buildInfoTile('Build', '2025.0227.1'),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        title,
        style: TextStyle(
          color: AppTheme.primaryGreen,
          fontSize: 14,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Widget _buildSwitchTile({
    required IconData icon,
    required String title,
    required String subtitle,
    required bool value,
    required ValueChangedCallback<bool> onChanged,
  }) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: ListTile(
        leading: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: AppTheme.primaryGreen.withOpacity(0.1),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(icon, color: AppTheme.primaryGreen, size: 24),
        ),
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.w500)),
        subtitle: Text(subtitle, style: TextStyle(color: Colors.grey[600], fontSize: 12)),
        trailing: Switch(
          value: value,
          onChanged: onChanged,
          activeColor: AppTheme.primaryGreen,
        ),
      ),
    );
  }

  Widget _buildActionTile({
    required IconData icon,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: ListTile(
        onTap: onTap,
        leading: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: AppTheme.primaryGreen.withOpacity(0.1),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(icon, color: AppTheme.primaryGreen, size: 24),
        ),
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.w500)),
        subtitle: Text(subtitle, style: TextStyle(color: Colors.grey[600], fontSize: 12)),
        trailing: const Icon(Icons.chevron_right, color: Colors.grey400]),
      ),
    );
  }

  Widget _buildLanguageSelector() {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Row(
        children: [
          Icon(Icons.language, color: AppTheme.primaryGreen, size: 24),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Sprache',
                  style: TextStyle(fontWeight: FontWeight.w500),
                ),
                Text(
                  'Deutsch / German',
                  style: TextStyle(color: Colors.grey[600], fontSize: 12),
                ),
              ],
            ),
          ),
          SegmentedButton<String>(
            selected: {if (_isGermanLanguage) 'de' else 'en'},
            onSelectionChanged: (Set<String> selection) {
              setState(() {
                _isGermanLanguage = selection.contains('de');
              });
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(_isGermanLanguage ? 'Sprache auf Deutsch gestellt': 'Language set to English')),
              );
            },
            segments: const <ButtonSegment<String>>[
              ButtonSegment<String>(
                value: 'de',
                label: const Text('DE'),
              ),
              ButtonSegment<String>(
                value: 'en',
                label: const Text('EN'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildInfoTile(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: TextStyle(color: Colors.grey[600]))),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }
}
