// Import React and ReactDOM from CDN
const { useState, useRef, useEffect } = React;
const { createRoot } = ReactDOM;

const ChatUI = () => {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [ws, setWs] = useState(null);
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef(null);
  const chatContainerRef = useRef(null);
  const [currentStreamingMessage, setCurrentStreamingMessage] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);

  // Auto-scroll effect
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [messages, currentStreamingMessage]);

  useEffect(() => {
    const websocket = new WebSocket('ws://localhost:8080/ws');

    websocket.onopen = () => {
      console.log('Connected to WebSocket');
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'ai' && message.isStream) {
        // Start streaming and accumulate the message
        setIsStreaming(true);
        setCurrentStreamingMessage(prev => prev + message.content);
      } else if (message.type === 'ai' && !message.isStream) {
        // Final message - add the complete streaming message to messages array
        if (currentStreamingMessage) {
          setMessages(prev => [...prev, {
            type: 'ai',
            content: currentStreamingMessage,
            sender: 'ai'
          }]);
          // Clear streaming message after adding to permanent messages
          setCurrentStreamingMessage('');
        }
        setIsStreaming(false);
      } else {
        // Handle other message types (system messages, etc)
        setMessages(prev => [...prev, message]);
      }
    };

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    websocket.onclose = () => {
      console.log('Disconnected from WebSocket');
    };

    setWs(websocket);

    return () => {
      websocket.close();
    };
  }, []);

  const handleSendMessage = () => {
    if (!inputMessage.trim() || !ws) return;

    const message = {
      type: 'user',
      content: inputMessage,
      sender: 'user'
    };

    // Store the message content before clearing
    const messageToSend = inputMessage.trim();

    // Add message to UI first
    setMessages(prev => [...prev, message]);

    // Send message through WebSocket
    try {
      ws.send(JSON.stringify(message));
      // Only clear input after successful send
      setInputMessage('');
      setCurrentStreamingMessage(''); // Clear any previous streaming message
    } catch (error) {
      console.error('Failed to send message:', error);
      // Optionally show error message to user
      setMessages(prev => [...prev, {
        type: 'error',
        content: 'Failed to send message. Please try again.',
        sender: 'system'
      }]);
    }
  };

  const handleFileUpload = async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    setIsUploading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await fetch('/api/upload', {
        method: 'POST',
        body: formData
      });

      if (!response.ok) {
        throw new Error('Upload failed');
      }

      const result = await response.json();
      setMessages(prev => [...prev, {
        type: 'system',
        content: `File "${result.file}" uploaded successfully! You can now ask me questions about it.`,
        sender: 'ai'
      }]);
    } catch (error) {
      console.error('File upload failed', error);
      setMessages(prev => [...prev, {
        type: 'error',
        content: "Sorry, the file upload failed. Please try again.",
        sender: 'system'
      }]);
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-200 to-pink-200 flex items-center justify-center p-4">
      <div className="w-full max-w-2xl bg-white rounded-xl shadow-xl overflow-hidden">
        {/* Header */}
        <div className="bg-gradient-to-r from-blue-500 to-blue-600 p-4">
          <h1 className="text-white text-xl font-bold">Financial Planner AI</h1>
        </div>

        {/* Chat Messages Container */}
        <div className="h-[500px] overflow-y-auto p-4 space-y-4 hide-scrollbar bg-gray-50" ref={chatContainerRef}>
          {messages.map((message, index) => (
            <div key={index} className={`flex ${message.sender === 'user' ? 'justify-end' : 'justify-start'} message-appear`}>
              <div className={`max-w-[80%] p-3 rounded-2xl ${
                message.sender === 'user' 
                  ? 'bg-blue-500 text-white' 
                  : message.type === 'error'
                    ? 'bg-red-100 text-red-700'
                    : message.type === 'system'
                      ? 'bg-gray-100 text-gray-700'
                      : 'bg-white text-gray-700 shadow-md'
              }`}>
                {message.content}
              </div>
            </div>
          ))}
          {currentStreamingMessage && (
            <div className="flex justify-start message-appear">
              <div className="max-w-[80%] p-3 rounded-2xl bg-white text-gray-700 shadow-md">
                {currentStreamingMessage}
              </div>
            </div>
          )}
        </div>

        {/* Input Area */}
        <div className="p-4 border-t border-gray-200 bg-white">
          <div className="flex items-center space-x-2">
            <input 
              type="text"
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              placeholder="Type your message..."
              className="flex-grow p-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
            />
            
            <button 
              onClick={handleSendMessage}
              className="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors"
              disabled={!inputMessage.trim() || isStreaming}
            >
              Send
            </button>

            <input 
              type="file" 
              ref={fileInputRef}
              onChange={handleFileUpload}
              className="hidden"
              accept=".pdf,.txt,.docx,.csv"
            />
            
            <button 
              onClick={() => fileInputRef.current.click()}
              className="bg-gray-500 text-white px-4 py-2 rounded-lg hover:bg-gray-600 transition-colors"
              disabled={isUploading}
            >
              {isUploading ? 'Uploading...' : 'Upload'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

// Create root and render the app
const root = createRoot(document.getElementById('root'));
root.render(<ChatUI />); 