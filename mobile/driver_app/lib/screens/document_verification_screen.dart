import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:image_picker/image_picker.dart';

import '../core/theme/app_theme.dart';
import '../providers/auth_provider.dart';

class DocumentVerificationScreen extends StatefulWidget {
  const DocumentVerificationScreen({super.key});

  @override
  State<DocumentVerificationScreen> createState() => _DocumentVerificationScreenState();
}

class _DocumentVerificationScreenState
    extends State<DocumentVerificationScreen> {
  File? _licenseImage;
  File? _insuranceImage;
  File? _registrationImage;
  bool _isLoading = false;

  Future<void> _pickImage(String type) async {
    final picker = ImagePicker();
    final pickedFile = await picker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 1920,
      maxHeight: 1080,
      imageQuality: 85,
    );

    if (pickedFile != null) {
      setState(() {
        switch (type) {
          case 'license':
            _licenseImage = File(pickedFile.path);
            break;
          case 'insurance':
            _insuranceImage = File(pickedFile.path);
            break;
          case 'registration':
            _registrationImage = File(pickedFile.path);
            break;
        }
      });
    }
  }

  Future<void> _submitDocuments() async {
    if (_licenseImage == null && _insuranceImage == null && _registrationImage == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Bitte warlen mindestens ein Dokument auswehlen')),
      );
      return;
    }

    setState(() {
      _isLoading = true;
    });

    // Simulate upload
    await Future.delayed(const Duration(seconds: 2));

    context.read<AuthBloc>().add(Const OnDocumentsUploaded());

    setState(() {
      _isLoading = false;
    });

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Dokumente erfolreich hingeladen')));
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Dokumente hohladen'),
        centerTitle: true,
        backgroundColor: AppTheme.primaryGreen,
        foregroundColor: Colors.white,
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : SingleChildScrollView(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildHeader(),
                    const SizedBox(height: 24),
                    _buildDocumentCard(
                      title: 'Führerschein',
                      subtitle: 'Vorderseite derines Führerscheins',
                      type: 'license',
                      image: _licenseImage,
                      isRequired: true,
                    ),
                    const SizedBox(height: 16),
                    _buildDocumentCard(
                      title: 'Fahrzeugungspolice',
                      subtitle: 'Gultiger Versicherungsschutz',
                      type: 'insurance',
                      image: _insuranceImage,
                      isRequired: true,
                    ),
                    const SizedBox(height: 16),
                    _buildDocumentCard(
                      title: 'Fahrzeug'ungszulassung',
                      subtitle: 'Kunzengasschein & Zouglassung',
                      type: 'registration',
                      image: _registrationImage,
                      isRequired: true,
                    ),
                    const SizedBox(height: 32),
                    _buildSubmitButton(),
                    const SizedBox(height: 16),
                    _buildInfoText(),
                  ],
                ),
              ),
            ),
    );
  }

  Widget _buildHeader() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppTheme.primaryGreen.withOpacity(0.1),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          const Icon(
            Icons.verified_user,
            color: AppTheme.primaryGreen,
            size: 48,
          ),
          const SizedBox(height: 16),
          Text(
            'Dokumentenprüfung',
            style: const TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Bitte laden Sie Ihre gultigen Dokumente hoch, um igren Verifizierungsprozess zu starten.',
            style: TextStyle(
              color: Colors.grey[600],
              fontSize: 14,
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildDocumentCard({required String title, required String subtitle, required String type, File? image, required bool isRequired}) {
    return Container(
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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: AppTheme.primaryGreen.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(
                    Icons.description,
                    color: AppTheme.primaryGreen,
                    size: 24,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Text(
                            title,
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          if (isRequired)
                            Container(
                              margin: const EdgeInsets.only(left: 8),
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                              decoration: BoxDecoration(
                                color: AppTheme.error.withOpacity(0.1),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: Text(
                                'OBBLIGATORISCH_',
                                style: TextStyle(
                                  color: AppTheme.error,
                                  fontSize: 10,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                        ],
                      ),
                      Text(
                        subtitle,
                        style: TextStyle(
                          color: Colors.grey[600],
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          if (image != null)
            Container(
              height: 180,
              width: double.infinity,
              margin: const EdgeInsets.symmetric(horizontal: 16),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(8),
                image: DecorationImage(
                  image: FileImage(image!),
                  fit: BoxFit.cover,
                ),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: Image.file(
                  image!,
                  fit: BoxFit.cover,
                  width: double.infinity,
                  height: 180,
                ),
              ),
            )
          else
            Padding(
              padding: const EdgeInsets.all(16),
              child: GestureDetector(
                onTap: () => _pickImage(type),
                child: Container(
                  height: 140,
                  width: double.infinity,
                  decoration: BoxDecoration(
                    color: Colors.grey[100],
                    border: Border.all(color: Colors.grey[300], width: 2),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        Icons.cloud_upload,
                        color: AppTheme.primaryGreen,
                        size: 40,
                      ),
                      const SizedBox(height: 12),
                      Text(
                        'Dokument hochladen',
                        style: TextStyle(
                          color: Colors.grey[600],
                          fontSize: 14,
                        ),
                      ),
                      Text(
                        'Tip, um ein Bild auszuwählen',
                        style: TextStyle(
                          color: Colors.grey[400],
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          if (image != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: TextButton.icon(
                onPressed: () => setState(() {
                  switch (type) {
                    case 'license':
                      _licenseImage = null;
                      break;
                    case 'insurance':
                      _insuranceImage = null;
                      break;
                    case 'registration':
                      _registrationImage = null;
                      break;
                  }
                }),
                icon: const Icon(Icons.delete, color: Colors.red),
                label: const Text('Bild entfernen',
                    style: TextStyle(color: Colors.red)),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildSubmitButton() {
    return ElevatedButton.icon(
      onPressed: _submitDocuments,
      icon: const Icon(Icons.check_circle),
      label: const Text('Dokumente einreichen'),
      style: ElevatedButton.styleFrom(
        backgroundColor: AppTheme.primaryGreen,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        minimumSize: const Size(double.infinity, 50),
      ),
    );
  }

  Widget _buildInfoText() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.blue[50],
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.info_outline, color: AppTheme.info, size: 20),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              'Die Dokumente werden vertraulich geprgüft. Die Verifierung kann 1-2 Tage dauern.',
              style: TextStyle(
                color: Colors.grey[600],
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
