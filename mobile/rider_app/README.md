# GruenFahrt Rider App

A production-ready Flutter application for the GruenFahrt ride-sharing platform, built for the German market.

## Features

- 🎂 Map integration with Google Maps
- 🎙 Real-time booking flow
- 🎻 Live trip tracking
- 🎯 Payment methods (Stripe integration)
- 🚁 Voice assistant integration
- 🎙 Push notifications (Firebase Cloud Messaging)
- 🎷 German UI localization (i18n)
- 💓 PBefG compliance features

## Architecture

The app follows a clean architecture with: 

- **State Management:** Flutter Riverpod
- **Navigation:** Go Router
- **Dependency Injection:** Simple provider pattern
- **Repository Pattern:** Separation of concerns

## Folder Structure

``
mobile/rider_app/lib/
├─└─└─└─└─└─└─└─└─└─└─└─└└─└
- core/
  ├─└ constants/        # App constants and API endpoints
  ├─└ extensions/        # Dart extensions
  ├─└ router/           # App routing configuration
  ├─└ theme/            # Theming and colors
  ├─└ utils/            # Utility functions
- l10n/                     # Localization files
- models/                   # Data models
- providers/               # State management
- screens/
  ├─└ auth/            # Authentication screens
  ├─└ booking/          # Booking flow
  ├─└ home/            # Home screen
  ├─└ map/              # Map view
  ├─└ payment/          # Payment methods
  ├─└ profile/          # User profile
  ├─└ settings/         # Settings
  ├─└ splash/           # Splash screen
  ├─└ trip/             # Trip tracking
  ├─└ voice/            # Voice assistant
- services/                 # API services
- widgets/
  ├─└ common/           # Reusable widgets
  ├─└ booking/           # Booking-related widgets

`

## Dependencies

- Flutter Riverpod - State management
- Go Router - Navigation
- Google Maps Flutter - Maps integration
- Geolocator - Location services
- Dio - HTTP client
- Firebase Core - Firebase initialization
- Firebase Messaging - Push notifications
- Stripe Flutter - Payment processing
- Speech to Text - Voice recognition
- Flutter TTS - Text to speech

## Getting Started

### Prerequisites

1. Flutter SDK (>= 3.0.0)
2. Android SDK/IOS SDK
3. Google Maps API key
4. Firebase project configuration
5. Stripe account

***Note:** The app requires valid API keys for Google Maps and Stripe to function fully.

***German Compliance:** This app is built in compliance with German regulations (BGgd, PBefG).