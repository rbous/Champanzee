'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface WebSocketMessage {
    type: string;
    [key: string]: unknown;
}

interface UseWebSocketOptions {
    onMessage?: (message: WebSocketMessage) => void;
    onConnect?: () => void;
    onDisconnect?: () => void;
    onError?: (error: Event) => void;
    reconnectAttempts?: number;
    reconnectInterval?: number;
}

export function useWebSocket(url: string | null, options: UseWebSocketOptions = {}) {
    const {
        onMessage,
        onConnect,
        onDisconnect,
        onError,
        reconnectAttempts = 5,
        reconnectInterval = 3000,
    } = options;

    const [status, setStatus] = useState<ConnectionStatus>('disconnected');
    const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

    const wsRef = useRef<WebSocket | null>(null);
    const reconnectCountRef = useRef(0);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const connect = useCallback(() => {
        if (!url) return;

        // Clean up existing connection
        if (wsRef.current) {
            wsRef.current.close();
        }

        setStatus('connecting');
        const ws = new WebSocket(url);
        wsRef.current = ws;

        ws.onopen = () => {
            setStatus('connected');
            reconnectCountRef.current = 0;
            onConnect?.();
        };

        ws.onclose = () => {
            setStatus('disconnected');
            onDisconnect?.();

            // Attempt reconnection
            if (reconnectCountRef.current < reconnectAttempts) {
                reconnectCountRef.current += 1;
                reconnectTimeoutRef.current = setTimeout(() => {
                    connect();
                }, reconnectInterval);
            }
        };

        ws.onerror = (event) => {
            setStatus('error');
            onError?.(event);
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data) as WebSocketMessage;
                setLastMessage(message);
                onMessage?.(message);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };
    }, [url, onMessage, onConnect, onDisconnect, onError, reconnectAttempts, reconnectInterval]);

    const disconnect = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
        }
        reconnectCountRef.current = reconnectAttempts; // Prevent reconnection
        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }
        setStatus('disconnected');
    }, [reconnectAttempts]);

    const send = useCallback((data: unknown) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify(data));
        }
    }, []);

    // Connect on mount, disconnect on unmount
    useEffect(() => {
        if (url) {
            connect();
        }

        return () => {
            disconnect();
        };
    }, [url, connect, disconnect]);

    return {
        status,
        lastMessage,
        send,
        connect,
        disconnect,
    };
}

// ============================================
// Host WebSocket Hook
// ============================================

export interface PlayerJoinedEvent {
    type: 'player_joined';
    playerId: string;
    nickname: string;
}

export interface PlayerLeftEvent {
    type: 'player_left';
    playerId: string;
}

export interface LeaderboardEntry {
    playerId: string;
    nickname: string;
    score: number;
    rank: number;
}

export interface LeaderboardUpdateEvent {
    type: 'leaderboard_update';
    leaderboard: LeaderboardEntry[];
}

export interface PlayerProgressEvent {
    type: 'player_progress_update';
    playerId: string;
    questionKey: string;
    status: string;
    resolution: 'SAT' | 'UNSAT' | null;
}

export type HostEvent =
    | PlayerJoinedEvent
    | PlayerLeftEvent
    | LeaderboardUpdateEvent
    | PlayerProgressEvent;

export function useHostWebSocket(
    url: string | null,
    handlers: {
        onPlayerJoined?: (event: PlayerJoinedEvent) => void;
        onPlayerLeft?: (event: PlayerLeftEvent) => void;
        onLeaderboardUpdate?: (event: LeaderboardUpdateEvent) => void;
        onPlayerProgress?: (event: PlayerProgressEvent) => void;
    } = {}
) {
    const handleMessage = useCallback((message: WebSocketMessage) => {
        switch (message.type) {
            case 'player_joined':
                handlers.onPlayerJoined?.(message as PlayerJoinedEvent);
                break;
            case 'player_left':
                handlers.onPlayerLeft?.(message as PlayerLeftEvent);
                break;
            case 'leaderboard_update':
                handlers.onLeaderboardUpdate?.(message as LeaderboardUpdateEvent);
                break;
            case 'player_progress_update':
                handlers.onPlayerProgress?.(message as PlayerProgressEvent);
                break;
        }
    }, [handlers]);

    return useWebSocket(url, { onMessage: handleMessage });
}

// ============================================
// Player WebSocket Hook
// ============================================

export interface NextQuestionEvent {
    type: 'next_question';
    question: {
        key: string;
        type: 'ESSAY' | 'DEGREE';
        prompt: string;
        pointsMax?: number;
        scaleMin?: number;
        scaleMax?: number;
    };
}

export interface EvaluationResultEvent {
    type: 'evaluation_result';
    resolution: 'SAT' | 'UNSAT';
    points: number;
    summary: string;
}

export interface ErrorEvent {
    type: 'error';
    message: string;
}

export type PlayerEvent = NextQuestionEvent | EvaluationResultEvent | ErrorEvent;

export function usePlayerWebSocket(
    url: string | null,
    handlers: {
        onNextQuestion?: (event: NextQuestionEvent) => void;
        onEvaluationResult?: (event: EvaluationResultEvent) => void;
        onError?: (event: ErrorEvent) => void;
    } = {}
) {
    const handleMessage = useCallback((message: WebSocketMessage) => {
        switch (message.type) {
            case 'next_question':
                handlers.onNextQuestion?.(message as NextQuestionEvent);
                break;
            case 'evaluation_result':
                handlers.onEvaluationResult?.(message as EvaluationResultEvent);
                break;
            case 'error':
                handlers.onError?.(message as ErrorEvent);
                break;
        }
    }, [handlers]);

    return useWebSocket(url, { onMessage: handleMessage });
}
