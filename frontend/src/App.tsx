import { useState, useRef, useEffect } from 'react';
import './App.css';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  emotion?: string;
  suggestions?: string[];
}

interface ChatResponse {
  response: string;
  emotion: string;
  suggestions: string[];
  sessionId: string;
}

function formatResponse(text: string): string {
  return text.replace(
    /\[book::(.+?)::(.+?)\]/g,
    '<span class="book-link">$1</span>'
  );
}

function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = async (text: string) => {
    if (!text.trim() || loading) return;

    const userMessage: Message = { role: 'user', content: text };
    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setLoading(true);

    try {
      const body: Record<string, string> = { message: text };
      if (sessionId) body.sessionId = sessionId;

      const res = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      const data: ChatResponse = await res.json();

      if (data.sessionId) setSessionId(data.sessionId);

      const assistantMessage: Message = {
        role: 'assistant',
        content: data.response,
        emotion: data.emotion,
        suggestions: data.suggestions,
      };
      setMessages((prev) => [...prev, assistantMessage]);
    } catch {
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: 'Sorry, something went wrong. Please try again.' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const lastAssistant = [...messages].reverse().find((m) => m.role === 'assistant');
  const suggestions = lastAssistant?.suggestions ?? [];

  return (
    <div className="app">
      <header className="header">
        <h1>TB: Talking Bookshelf</h1>
        <p className="subtitle">A bookshelf that talks about its owner's reading experiences</p>
      </header>

      <div className="chat-container">
        <div className="messages">
          {messages.length === 0 && (
            <div className="empty-state">
              Ask me about the books on my shelf!
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`message ${msg.role}`}>
              {msg.role === 'assistant' && <span className="avatar">&#128218;</span>}
              <div
                className="bubble"
                dangerouslySetInnerHTML={{
                  __html: msg.role === 'assistant' ? formatResponse(msg.content) : msg.content,
                }}
              />
            </div>
          ))}
          {loading && (
            <div className="message assistant">
              <span className="avatar">&#128218;</span>
              <div className="bubble loading-dots">
                <span /><span /><span />
              </div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {suggestions.length > 0 && !loading && (
          <div className="suggestions">
            {suggestions.map((s, i) => (
              <button key={i} className="chip" onClick={() => sendMessage(s)}>
                {s}
              </button>
            ))}
          </div>
        )}

        <form
          className="input-area"
          onSubmit={(e) => {
            e.preventDefault();
            sendMessage(input);
          }}
        >
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask about books..."
            maxLength={250}
            disabled={loading}
          />
          <button type="submit" disabled={!input.trim() || loading}>
            Send
          </button>
        </form>
      </div>
    </div>
  );
}

export default App;
