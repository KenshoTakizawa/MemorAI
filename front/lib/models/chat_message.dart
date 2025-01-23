class ChatMessage {
  final String id;
  final String content;
  final String role;
  final DateTime timestamp;
  bool isLiked;
  bool isDisliked;

  ChatMessage({
    required this.content,
    required this.role,
    this.id = '', // デフォルト値を設定
    DateTime? timestamp, // オプショナルパラメータに変更
    this.isLiked = false,
    this.isDisliked = false,
  }) : timestamp = timestamp ?? DateTime.now(); // デフォルト値を設定

  factory ChatMessage.fromJson(Map<String, dynamic> json) {
    return ChatMessage(
      id: json['id'] ?? '', // null の場合に空文字を設定
      role: json['role'] ?? 'assistant',
      content: json['content'] ?? '不明な内容',
      timestamp: json['timestamp'] != null
          ? DateTime.parse(json['timestamp'])
          : DateTime.now(), // null の場合に現在の日時を設定
      isLiked: json['isLiked'] ?? false,
      isDisliked: json['isDisliked'] ?? false,
    );
  }
}
