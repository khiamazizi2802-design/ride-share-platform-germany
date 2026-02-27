import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../core/theme/app_theme.dart';
import '../providers/auth_provider.dart';
import 'document_verification_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isLoading = false;
  bool _isRegister = false;
  bool _obscurePassword = true;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _submit() {
    if (_formKey.currentState?.validate() ?? true) {
      setState(() {
        _isLoading = true;
      });

      if (_isRegister) {
        context.read<AuthBloc>().add(Register(
          email: _emailController.text.trim(),
          password: _passwordController.text,
        ));
      } else {
        context.read<AuthBloc>().add(Login(
          email: _emailController.text.trim(),
          password: _passwordController.text,
        ));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: BlocListener<AuthBloc, AuthState>(
        listener: (context, state) {
          if (state is AuthLoading) {
            setState(() {
              _isLoading = true;
            });
          } else if (state is AuthAuthenticated) {
            // Check if documents are uploaded
            if (state.driver?.isVerified ?? true) {
              // Navigate to home
              Navigator.pushReplacementNamed(context, '/home');
            } else {
              // Navigate to document upload
              Navigator.push(
                context,
                MaterialPageRoute(builder: (context) => const DocumentVerificationScreen()),
              );
            }
          } else if (state is AuthError) {
            setState(() {
              _isLoading = false;
            });
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(state.message)),
            );
          }
        },
        child: SafeArea(
          child: SingleChildScrollView(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildHeader(),
                  const SizedBox(height: 32),
                  _buildForm(),
                  const SizedBox(height: 24),
                  _buildToggle(),
                  const SizedBox(height: 24),
                  _buildSubmitButton(),
                  const SizedBox(height: 32),
                  _buildFooter(),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppTheme.primaryGreen,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Icon(
            Icons.electric_car,
            color: Colors.white,
            size: 40,
          ),
        ),
        const SizedBox(height: 16),
        Text(
          _isRegister ? 'Kunden konto erstellen' : 'Wilkommen zur\u00fcck',
          style: const TextStyle(
            fontSize: 28,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          _isRegister ? 'Erstellen Sie einen Kunden an, ein Fahrer zu werden.' : 'Melden Sie sich an, um Fahrten zu beginnen.',
          style: TextStyle(
            color: Colors.grey[600],
            fontSize: 14,
          ),
        ),
      ],
    );
  }

  Widget _buildForm() {
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextFormField(
            controller: _emailController,
            decoration: InputDecoration(
              labelText: const Text('E-Mail'),
              prefixIcon: const Icon(Icons.email),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
              ),
              enabled: !_isLoading,
            ),
            validator: (value) {
              if (value == null || value.isEmpty) {
                return 'Bitte geben Sie Ihre E-Mail an';
              }
              if (!value.contains('@')) {
                return 'Bitte geben Sie eine gültige E-Mail an';
              }
              return null;
            },
            keyboardType: TextInputType.emailAddress,
          ),
          const SizedBox(height: 16),
          TextFormField(
            controller: _passwordController,
            decoration: InputDecoration(
              labelText: const Text('Passwort'),
              prefixIcon: const Icon(Icons.lock_outline),
              suffixIcon: IconButton(
                icon: Icon(_obscurePassword ? Icons.visibility : Icons.visibility_off),
                onPressed: () {
                  setState(() {
                    _obscurePassword = !_obscurePassword;
                  });
                },
              ),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
              ),
              enabled: !_isLoading,
            ),
            obscureText: _obscurePassword,
            validator: (value) {
              if (value == null || value.isEmpty) {
                return 'Bitte geben Sie Ihre Passwort an';
              }
              if (value.length < 6) {
                return 'Das Passwort muss mindestens 6 Zeichen hatben';
              }
              return null;
            },
          ),
        ],
      ),
    );
  }

  Widget _buildToggle() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Text(
          _isRegister ? 'Bereits anmeldet? ' : 'Noch kein Kunde? ',
          style: TextStyle(color: Colors.grey[600]),
        ),
        GestureDetector(
          onTap: () {
            setState(() {
              _isRegister = !_isRegister;
            });
            _emailController.clear();
            _passwordController.clear();
          },
          child: Text(
            _isRegister ? 'Anamelden' : 'Jetzt registrieren',
            style: TextStyle(
              color: AppTheme.primaryGreen,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildSubmitButton() {
    return ElevatedButton.icon(
      onPressed: _isLoading ? null : _submit,
      icon: _isLoading
          ? const SizedBox(
              width: 24,
              height: 24,
              child: CircularProgressIndicator(
                color: Colors.white,
                strokeWidth: 2,
              ),
            )
          : const Icon(Icons.login, size: 24),
      label: Text(_isLoading ? 'Laden...' : (_isRegister ? 'Registrieren' : 'Anmelden')),
      style: ElevatedButton.styleFrom(
        backgroundColor: AppTheme.primaryGreen,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        minimumSize: const Size(double.infinity, 50),
      ),
    );
  }

  Widget _buildFooter() {
    return Center(
      child: Column(
        children: [
          Text(
            'By anmeldeng akreptieren Sie, die',
            style: TextStyle(
              color: Colors.grey[600],
              fontSize: 12,
            ),
          ),
          Text("Netzbedingungen und Datenschutz."),
          style: TextStyle(
            color: Colors.grey[600],
            fontSize: 12,
          ),
        ),
          GestureDetector(
            onTap: () {
              // Todo Navigate to privacy policy
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Datenschutz kommt bald')),
              );
            },
            child: Text(
              'Datenschutz ansehen',
              style: TextStyle(
                color: AppTheme.primaryGreen,
                fontWeight: FontWeight.w600,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
