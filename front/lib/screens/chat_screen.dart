import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import '../services/chat_service.dart';
import '../models/chat_message.dart';
import 'dart:developer';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  ChatScreenState createState() => ChatScreenState();
}

class ChatScreenState extends State<ChatScreen> {
  final List<ChatMessage> _messages = [];
  final TextEditingController _controller = TextEditingController();
  late final ChatService _chatService;
  bool _isLoading = false;
  final ScrollController _scrollController = ScrollController();
  bool _isInitialized = false;

  @override
  void initState() {
    super.initState();
    _initializeChatService();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      Future.delayed(const Duration(milliseconds: 100), () {
        if (_scrollController.hasClients) {
          _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
        }
      });
    });
  }

  Future<void> _initializeChatService() async {
    _chatService = await ChatService.create();
    await _loadPastConversations();
    setState(() {
      _isInitialized = true;
    });
  }

  Future<void> _loadPastConversations() async {
    try {
      final pastMessages = await _chatService.fetchConversations('1');
      setState(() {
        _messages.addAll(pastMessages);
      });

      // リスト追加後、次のフレーム（＝描画完了）になってからスクロールする
      WidgetsBinding.instance.addPostFrameCallback((_) async {
        _scrollToBottom();
        await Future.delayed(const Duration(milliseconds: 2000));
        _scrollToBottom();
      });
    } catch (e) {
      log("Failed to load past conversations: $e");
    }
  }

  void _sendMessage() async {
    if (!_isInitialized) {
      log('ChatService not initialized yet');
      return;
    }
    if (_controller.text.isNotEmpty) {
      final userMessage = ChatMessage(role: 'user', content: _controller.text);

      setState(() {
        _messages.add(userMessage);
        _isLoading = true;
      });

      WidgetsBinding.instance.addPostFrameCallback((_) async {
        _scrollToBottom();
      });

      _controller.clear();

      try {
        final botReply = await _chatService.sendMessage(userMessage.content);
        setState(() {
          _messages.add(botReply);
          _isLoading = false;
          _scrollToBottom(); // 新しいメッセージを追加後にスクロール
        });
      } catch (e) {
        setState(() {
          _messages.add(ChatMessage(
            role: 'bot',
            content: 'エラーが発生しました: $e',
          ));
          _isLoading = false;
          _scrollToBottom(); // エラーでもスクロール
        });
      }
    }
  }

  // void _scrollToBottom() {
  //   WidgetsBinding.instance.addPostFrameCallback((_) {
  //     if (_scrollController.hasClients) {
  //       _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
  //     }
  //   });
  // }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Colors.grey[900],
        title: const Row(
          children: [
            Icon(Icons.memory, color: Colors.blue),
            SizedBox(width: 8),
            Text('MemorAI', style: TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 5.0),
            child: ElevatedButton.icon(
              style: ElevatedButton.styleFrom(
                backgroundColor: Colors.blue,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8),
                ),
              ),
              icon: const Icon(Icons.article, size: 16),
              label: const Text('AIの話題ちょうだい', style: TextStyle(fontSize: 14)),
              onPressed: () async {
                try {
                  setState(() {
                    _isLoading = true;
                  });
                  final aiTopic = await _chatService.getAITopic();
                  setState(() {
                    _messages.add(aiTopic);
                    _isLoading = false;
                    _scrollToBottom();
                  });
                } catch (e) {
                  setState(() {
                    _messages.add(ChatMessage(
                      role: 'assistant',
                      content: 'エラーが発生しました: $e',
                    ));
                    _isLoading = false;
                    _scrollToBottom();
                  });
                }
              },
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              controller: _scrollController,
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(8, 8, 12, 8),
              itemCount: _messages.length + (_isLoading ? 1 : 0),
              itemBuilder: (context, index) {
                if (index == _messages.length && _isLoading) {
                  return const Align(
                    alignment: Alignment.centerLeft,
                    child: Padding(
                      padding: EdgeInsets.all(8.0),
                      child: CircularProgressIndicator(),
                    ),
                  );
                }
                final message = _messages[index];
                final isUser = message.role == 'user';
                return Align(
                  alignment:
                      isUser ? Alignment.centerRight : Alignment.centerLeft,
                  child: Column(
                    crossAxisAlignment: isUser
                        ? CrossAxisAlignment.end
                        : CrossAxisAlignment.start,
                    children: [
                      Container(
                        margin: const EdgeInsets.symmetric(vertical: 4),
                        padding: const EdgeInsets.all(12),
                        constraints: BoxConstraints(
                          maxWidth: MediaQuery.of(context).size.width * 0.8,
                        ),
                        decoration: BoxDecoration(
                          color: isUser ? Colors.blue[600] : Colors.grey[800],
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: isUser
                            ? Text(
                                message.content,
                                style: const TextStyle(color: Colors.white),
                              )
                            : MarkdownBody(
                                data: message.content,
                                styleSheet: MarkdownStyleSheet(
                                  p: const TextStyle(color: Colors.white),
                                  code: TextStyle(
                                    backgroundColor: Colors.grey[700],
                                    color: Colors.white,
                                    fontFamily: 'monospace',
                                  ),
                                  codeblockDecoration: BoxDecoration(
                                    color: Colors.grey[700],
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                  blockquote: TextStyle(
                                    color: Colors.grey[400],
                                    fontStyle: FontStyle.italic,
                                  ),
                                  a: const TextStyle(color: Colors.blue),
                                ),
                              ),
                      ),
                      if (!isUser) // ボットのメッセージにだけボタンを表示
                        Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: Icon(
                                Icons.thumb_up,
                                color:
                                    message.isLiked ? Colors.blue : Colors.grey,
                              ),
                              onPressed: () async {
                                setState(() {
                                  message.isLiked = !message.isLiked;
                                  if (message.isLiked) {
                                    message.isDisliked = false;
                                  }
                                });

                                try {
                                  await _chatService.updateMessageFlag(
                                    userId: '1',
                                    timestamp:
                                        message.timestamp.toIso8601String(),
                                    isLiked: message.isLiked,
                                  );
                                } catch (e) {
                                  print("Failed to update like flag: $e");
                                }
                              },
                            ),
                            IconButton(
                              icon: Icon(
                                Icons.thumb_down,
                                color: message.isDisliked
                                    ? Colors.red
                                    : Colors.grey,
                              ),
                              onPressed: () async {
                                setState(() {
                                  message.isDisliked = !message.isDisliked;
                                  if (message.isDisliked) {
                                    message.isLiked = false;
                                  }
                                });

                                try {
                                  await _chatService.updateMessageFlag(
                                    userId: '1',
                                    timestamp:
                                        message.timestamp.toIso8601String(),
                                    isDisliked: message.isDisliked,
                                  );
                                } catch (e) {
                                  print("Failed to update dislike flag: $e");
                                }
                              },
                            ),
                          ],
                        ),
                    ],
                  ),
                );
              },
            ),
          ),
          SafeArea(
            child: Container(
              padding: const EdgeInsets.only(
                  left: 16, right: 16, top: 8, bottom: 8), // 上部に余白を追加
              color: Colors.black,
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _controller,
                      style: const TextStyle(color: Colors.white, fontSize: 14),
                      decoration: InputDecoration(
                        hintText: 'メッセージを入力...',
                        hintStyle: TextStyle(color: Colors.grey[400]),
                        filled: true,
                        fillColor: Colors.grey[800],
                        contentPadding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 4),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: BorderSide.none,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: _sendMessage,
                    icon: const Icon(Icons.send, color: Colors.white),
                    color: Colors.blue,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }
}
