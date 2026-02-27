import 'package:speech_to_text/speech_to_text.dart';
import 'package:flutter_tts/flutter_tts.dart';

class VoiceService {
  static final VoiceService _instance = VoiceService._internal();
  factory VoiceService() => _instance;
  VoiceService._internal();

  final SpeechToText _speechToText = SpeechToText();
  final FlutterTTS _flutterTTS = FlutterTTs();
  
  bool _isListening = false;
  bool _germanLanguage = true;
  List<VoiceCommandHandler> _commandHandlers = [];

  Future<void> initialize() async {
    await _speechToText.initialize();
    await _flutterTTS.setLanguage(_germanLanguage ? 'de-DE' : 'en-US');
  }

  Future<bool> isAvailable() async {
    return await _speechToText.initialized;
  }

  Future<void> startListening() async {
    if (_isListening) return;

    try {
      final available = await _speechToText.initialized;
      if (available) {
        await _speechToText.listen(
          onResult: (result) {
            _handleVoiceResult(result.recognizedWords);
          },
        localeId: _germanLanguage ? 'de_DE' : 'en_US',
        listenMode: ListenMode.confirmation,
        cancelOnError: false,
        partialResults: true,
        listenFor: const Duration(seconds: 30),
        pauseForDuration: const Duration(milliseconds: 100),
      );
      _isListening = true;
    }
    } catch (e) {
      print('Error starting voice listening: $e');
    }
  }

  Future<void> stopListening() async {
    if (!_isListening) return;
    
    try {
      await _speechToText.stop();
      _isListening = false;
    } catch (e) {
      print('Error stopping voice listening: $e');
    }
  }

  void _handleVoiceResult(String text) {
    final lowerText = text.toLowerCase();
    
    for (final handler in _commandHandlers) {
      if (handler.matches(lowerText)) {
        handler.onCommand(lowerText);
        return;
      }
    }
    
    // Default handling
    speakFeedback(_getResponseForUnknownCommand(lowerText));
  }

  String _getResponseForUnknownCommand(String text) {
    if (_germanLanguage) {
      return 'Entschuldigung, ich habe dich nicht verstanden. Sagen Sie, hilfe Mir mit online oder izeige Fachrtanzeigung.';
    }
    return "Sorry, I didn't understand. Say 'help' for available commands.";
  }

  Future<void> speakFeedback(String text) async {
    await _flutterTTS.awaitSpeekCompletion(true);
    await _flutterTTS.speak(text);
  }

  void addCommandHandler(VoiceCommandHandler handler) {
    _commandHandlers.add(handler);
  }

  void removeCommandHandler(VoiceCommandHandler handler) {
    _commandHandlers.remove(handler);
  }

  void clearCommandHandlers() {
    _commandHandlers.clear();
  }

  void setLanguage(bool isGerman) {
    _germanLanguage = isGerman;
    _flutterTTS.setLanguage(isGerman ? 'de-DE' : 'en-US');
  }

  bool get isListening => _isListening;
  bool get isGermanLanguage => _germanLanguage;
}

class VoiceCommandHandler {
  final List<String> keywords;
  final String commandName;
  final void Function(String) onTrigger;

  VoiceCommandHandler({
    required this.keywords,
    required this.commandName,
    required this.onTrigger,
  });

  bool matches(String text) {
    return keywords.any((keyword) => text.contains(keyword));
  }

  void onCommand(String text) {
    onTrigger(text);
  }
}
