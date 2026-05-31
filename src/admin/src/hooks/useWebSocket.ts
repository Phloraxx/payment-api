import { useEffect, useRef } from "react";

export function useWebSocket(path: string, onMessage: (message: unknown) => void) {
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    const protocol = location.protocol === "https:" ? "wss" : "ws";
    const socket = new WebSocket(`${protocol}://${location.host}${path}`);
    socket.onmessage = (event) => {
      try {
        onMessageRef.current(JSON.parse(event.data));
      } catch {
        onMessageRef.current(event.data);
      }
    };
    return () => socket.close();
  }, [path]);
}
