import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import { useAuth } from '../services/AuthContext';

interface WebSocketContextType {
  isWsConnected: boolean;
  lastMessage: MessageEvent | null;
  disconnect: () => void;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export const useWebSocket = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

interface WebSocketProviderProps {
  children: React.ReactNode;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const { token } = useAuth();
  const [isWsConnected, setIsWsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const isManuallyClosing = useRef(false);

  const disconnect = useCallback(() => {
    if (ws.current) {
      console.log("Disconnecting WebSocket...");
      isManuallyClosing.current = true;
      ws.current.close();
      ws.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    if (!token || ws.current) {
      return;
    }

    isManuallyClosing.current = false; // Reset flag on new connection attempt
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/ws?token=${token}`;

    const socket = new WebSocket(wsUrl);
    ws.current = socket;

    socket.onopen = () => {
      console.log('WebSocket connected');
      setIsWsConnected(true);
    };

    socket.onmessage = (event) => {
      setLastMessage(event);
    };

    socket.onclose = () => {
      console.log('WebSocket disconnected');
      setIsWsConnected(false);
      ws.current = null;
      // Only reconnect if it wasn't a manual closure
      if (!isManuallyClosing.current) {
        setTimeout(() => connect(), 5000);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      // The onclose event will be fired automatically after an error
    };

  }, [token]);

  useEffect(() => {
    if (token) {
      connect();
    } else {
      disconnect();
    }

    // The cleanup function handles disconnection on logout or component unmount
    return () => {
      disconnect();
    };
  }, [token, connect, disconnect]);

  const value = {
    isWsConnected,
    lastMessage,
    disconnect,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};
