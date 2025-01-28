import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/chat_message.dart';

class ChatService {
  static const String apiUrl = 'http://172.16.80.125:8080/chat';
  final String userId = "1"; 

  ChatService._internal(); 

  static Future<ChatService> create() async {
    return ChatService._internal(); 
  }

  Future<ChatMessage> sendMessage(String message) async {
    try {
      final response = await http.post(
        Uri.parse(apiUrl),
        headers: {
          'Content-Type': 'application/json',
        },
        body: jsonEncode({
          'message': message,
          'user_id': userId, 
        }),
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        return ChatMessage(
          id: data['id'], 
          content: data['reply'], 
          role: 'asistant',
          timestamp: DateTime.parse(data['timestamp']), 
        );
      } else {
        throw Exception('Failed to connect to API');
      }
    } catch (e) {
      throw Exception('Error: $e');
    }
  }

  Future<void> updateMessageFlag({
    required String userId,
    required String timestamp,
    bool? isLiked,
    bool? isDisliked,
  }) async {
    try {
      final body = json.encode({
        'userId': userId,
        'timestamp': timestamp,
        'isLiked': isLiked,
        'isDisliked': isDisliked,
      });

      // デバッグログ
      print("Update Message Flag Body: $body");

      final response = await http.post(
        Uri.parse('$apiUrl/update-flag'),
        headers: {'Content-Type': 'application/json'},
        body: body,
      );

      if (response.statusCode != 200) {
        throw Exception('Failed to update message flag');
      }
    } catch (e) {
      throw Exception('Error updating message flag: $e');
    }
  }

  Future<List<ChatMessage>> fetchConversations(String userId) async {
    final response = await http.get(
      Uri.parse('$apiUrl/conversations?userId=$userId'),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to fetch conversations');
    }

    final data = json.decode(response.body);
    final List<dynamic> conversations = data['conversations'];

    return conversations.map((json) => ChatMessage.fromJson(json)).toList();
  }

  Future<ChatMessage> getAITopic() async {
    try {
      final response = await http.get(
        Uri.parse('$apiUrl/research-ai?userId=$userId'),
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        return ChatMessage(
          id: data['id'],
          content: data['reply'],
          role: 'assistant',
          timestamp: DateTime.parse(data['timestamp']),
        );
      } else {
        throw Exception('Failed to fetch AI topic');
      }
    } catch (e) {
      throw Exception('Error: $e');
    }
  }
}
