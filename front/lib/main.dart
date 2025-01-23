import 'package:flutter/material.dart';
import 'screens/chat_screen.dart';

void main() {
  runApp(const MemorAIApp());
}

class MemorAIApp extends StatelessWidget {
  const MemorAIApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark().copyWith(
        scaffoldBackgroundColor: Colors.black,
        textTheme: const TextTheme(
          bodyMedium: TextStyle(color: Colors.white),
        ),
      ),
      home: const ChatScreen(),
    );
  }
}
