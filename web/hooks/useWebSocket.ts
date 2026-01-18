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

    // Store handlers in refs to avoid reconnection on handler change
    const onMessageRef = useRef(onMessage);
    const onConnectRef = useRef(onConnect);
    const onDisconnectRef = useRef(onDisconnect);
    const onErrorRef = useRef(onError);

    useEffect(() => { onMessageRef.current = onMessage; }, [onMessage]);
    useEffect(() => { onConnectRef.current = onConnect; }, [onConnect]);
    useEffect(() => { onDisconnectRef.current = onDisconnect; }, [onDisconnect]);
    useEffect(() => { onErrorRef.current = onError; }, [onError]);

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
            onConnectRef.current?.();
        };

        ws.onclose = () => {
            setStatus('disconnected');
            onDisconnectRef.current?.();

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
            onErrorRef.current?.(event);
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data) as WebSocketMessage;
                setLastMessage(message);
                onMessageRef.current?.(message);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };
    }, [url, reconnectAttempts, reconnectInterval]);

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
    | PlayerProgressEvent
    | RoomEndedEvent;

export function useHostWebSocket(
    url: string | null,
    handlers: {
        onPlayerJoined?: (event: PlayerJoinedEvent) => void;
        onPlayerLeft?: (event: PlayerLeftEvent) => void;
        onLeaderboardUpdate?: (event: LeaderboardUpdateEvent) => void;
        onPlayerProgress?: (event: PlayerProgressEvent) => void;
        onRoomEnded?: (event: RoomEndedEvent) => void;
    } = {}
) {
    const handleMessage = useCallback((message: WebSocketMessage) => {
        const payload = message.payload as Record<string, unknown> || {};
        const event = { ...payload, type: message.type };

        switch (message.type) {
            case 'player_joined':
                handlers.onPlayerJoined?.(event as unknown as PlayerJoinedEvent);
                break;
            case 'player_left':
                handlers.onPlayerLeft?.(event as unknown as PlayerLeftEvent);
                break;
            case 'leaderboard_update':
                handlers.onLeaderboardUpdate?.(event as unknown as LeaderboardUpdateEvent);
                break;
            case 'player_progress_update':
                handlers.onPlayerProgress?.(event as unknown as PlayerProgressEvent);
                break;
            case 'room_ended':
                handlers.onRoomEnded?.(event as unknown as RoomEndedEvent);
                break;
        }
    }, [handlers]);

    return useWebSocket(url, { onMessage: handleMessage });
}

// ============================================
// Player WebSocket Hook
// ============================================

import { type Question } from '@/lib/api';

export interface NextQuestionEvent {
    type: 'next_question';
    question: Question;
}

export interface EvaluationResultEvent {
    type: 'evaluation_result';
    resolution: 'SAT' | 'UNSAT';
    pointsEarned: number;
    evalSummary: string;
    nextQuestion?: Question | null;
    followUp?: Question | null;
}

export interface ErrorEvent {
    type: 'error';
    message: string;
}

export interface AIThinkingEvent {
    type: 'ai_thinking';
    questionKey: string;
}

export interface RoomStartedEvent {
    type: 'room_started';
    status: 'ACTIVE';
}

export interface RoomEndedEvent {
    type: 'room_ended';
    status: 'ENDED';
}

export type PlayerEvent = NextQuestionEvent | EvaluationResultEvent | AIThinkingEvent | ErrorEvent | RoomStartedEvent | RoomEndedEvent;

export function usePlayerWebSocket(
    url: string | null,
    handlers: {
        onNextQuestion?: (event: NextQuestionEvent) => void;
        onEvaluationResult?: (event: EvaluationResultEvent) => void;
        onAIThinking?: (event: AIThinkingEvent) => void;
        onError?: (event: ErrorEvent) => void;
        onRoomStarted?: (event: RoomStartedEvent) => void;
        onRoomEnded?: (event: RoomEndedEvent) => void;
    } = {}
) {
    const handleMessage = useCallback((message: WebSocketMessage) => {
        const payload = message.payload as Record<string, unknown> || {};
        const event = { ...payload, type: message.type };

        switch (message.type) {
            case 'next_question':
                handlers.onNextQuestion?.(event as unknown as NextQuestionEvent);
                break;
            case 'ai_thinking':
                handlers.onAIThinking?.(event as unknown as AIThinkingEvent);
                break;
            case 'evaluation_result':
                handlers.onEvaluationResult?.(event as unknown as EvaluationResultEvent);
                break;
            case 'error':
                handlers.onError?.(event as unknown as ErrorEvent);
                break;
            case 'room_started':
                handlers.onRoomStarted?.(event as unknown as RoomStartedEvent);
                break;
            case 'room_ended':
                handlers.onRoomEnded?.(event as unknown as RoomEndedEvent);
                break;
        }
    }, [handlers]);

    return useWebSocket(url, { onMessage: handleMessage });
}
