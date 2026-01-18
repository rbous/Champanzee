'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { rooms, getHostWebSocketUrl, type LeaderboardEntry } from '@/lib/api';
import { useHostWebSocket, type PlayerJoinedEvent, type PlayerLeftEvent, type LeaderboardUpdateEvent, type PlayerProgressEvent } from '@/hooks/useWebSocket';

interface Player {
    id: string;
    nickname: string;
    status: 'joined' | 'answering' | 'done';
    currentQuestion?: string;
}

export default function RoomDashboard() {
    const params = useParams();
    const router = useRouter();
    const code = params.code as string;

    const [roomStatus, setRoomStatus] = useState<'waiting' | 'active' | 'ended'>('waiting');
    const [players, setPlayers] = useState<Map<string, Player>>(new Map());
    const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
    const [wsUrl, setWsUrl] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    // WebSocket handlers
    const handlePlayerJoined = useCallback((event: PlayerJoinedEvent) => {
        setPlayers(prev => {
            const updated = new Map(prev);
            updated.set(event.playerId, {
                id: event.playerId,
                nickname: event.nickname,
                status: 'joined',
            });
            return updated;
        });
    }, []);

    const handlePlayerLeft = useCallback((event: PlayerLeftEvent) => {
        setPlayers(prev => {
            const updated = new Map(prev);
            updated.delete(event.playerId);
            return updated;
        });
    }, []);

    const handleLeaderboardUpdate = useCallback((event: LeaderboardUpdateEvent) => {
        setLeaderboard(event.leaderboard);
    }, []);

    const handlePlayerProgress = useCallback((event: PlayerProgressEvent) => {
        setPlayers(prev => {
            const updated = new Map(prev);
            const player = updated.get(event.playerId);
            if (player) {
                updated.set(event.playerId, {
                    ...player,
                    status: 'answering',
                    currentQuestion: event.questionKey,
                });
            }
            return updated;
        });
    }, []);

    const { status: wsStatus } = useHostWebSocket(wsUrl, {
        onPlayerJoined: handlePlayerJoined,
        onPlayerLeft: handlePlayerLeft,
        onLeaderboardUpdate: handleLeaderboardUpdate,
        onPlayerProgress: handlePlayerProgress,
    });

    useEffect(() => {
        // Set up WebSocket connection
        const url = getHostWebSocketUrl(code);
        setWsUrl(url);
    }, [code]);

    const handleStartRoom = async () => {
        setLoading(true);
        try {
            await rooms.start(code);
            setRoomStatus('active');
        } catch (err) {
            console.error('Failed to start room:', err);
            alert('Failed to start room');
        } finally {
            setLoading(false);
        }
    };

    const handleEndRoom = async () => {
        if (!confirm('Are you sure you want to end this room?')) return;

        setLoading(true);
        try {
            await rooms.end(code);
            setRoomStatus('ended');
            router.push(`/host/reports/${code}`);
        } catch (err) {
            console.error('Failed to end room:', err);
            alert('Failed to end room');
        } finally {
            setLoading(false);
        }
    };

    const copyJoinLink = () => {
        const link = `${window.location.origin}/?code=${code}`;
        navigator.clipboard.writeText(link);
    };

    const playerList = Array.from(players.values());

    return (
        <div className="min-h-screen p-6">
            {/* Header */}
            <header className="flex items-center justify-between mb-8">
                <div className="flex items-center gap-4">
                    <a href="/host" className="btn btn-ghost">← Back</a>
                    <div>
                        <h1 className="text-xl font-bold">Room Dashboard</h1>
                        <div className="flex items-center gap-2 mt-1">
                            <span className={`w-2 h-2 rounded-full ${wsStatus === 'connected' ? 'bg-[var(--success)]' :
                                    wsStatus === 'connecting' ? 'bg-[var(--warning)] animate-pulse' :
                                        'bg-[var(--error)]'
                                }`} />
                            <span className="text-sm text-[var(--foreground-muted)]">
                                {wsStatus === 'connected' ? 'Live' : wsStatus}
                            </span>
                        </div>
                    </div>
                </div>

                <div className="flex items-center gap-3">
                    {roomStatus === 'waiting' && (
                        <button
                            onClick={handleStartRoom}
                            disabled={loading || playerList.length === 0}
                            className="btn btn-success"
                        >
                            {loading ? 'Starting...' : '▶ Start Room'}
                        </button>
                    )}
                    {roomStatus === 'active' && (
                        <button
                            onClick={handleEndRoom}
                            disabled={loading}
                            className="btn btn-danger"
                        >
                            {loading ? 'Ending...' : '⏹ End Room'}
                        </button>
                    )}
                </div>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Room Code Display */}
                <div className="lg:col-span-2">
                    <div className="card text-center py-8 mb-6">
                        <p className="text-sm text-[var(--foreground-muted)] mb-2">Room Code</p>
                        <div className="room-code mb-4">{code}</div>
                        <div className="flex items-center justify-center gap-3">
                            <button onClick={copyJoinLink} className="btn btn-secondary">
                                📋 Copy Join Link
                            </button>
                            <span className="badge badge-neutral">
                                {playerList.length} player{playerList.length !== 1 ? 's' : ''} joined
                            </span>
                        </div>
                    </div>

                    {/* Leaderboard */}
                    <div className="card">
                        <h2 className="text-lg font-semibold mb-4">Leaderboard</h2>
                        {leaderboard.length === 0 ? (
                            <div className="text-center py-8 text-[var(--foreground-muted)]">
                                {roomStatus === 'waiting'
                                    ? 'Start the room to see scores'
                                    : 'Waiting for answers...'}
                            </div>
                        ) : (
                            <div className="space-y-2">
                                {leaderboard.map((entry, i) => (
                                    <div key={entry.playerId} className="leaderboard-item">
                                        <div className={`leaderboard-rank ${i === 0 ? 'gold' : i === 1 ? 'silver' : i === 2 ? 'bronze' : ''
                                            }`}>
                                            {entry.rank}
                                        </div>
                                        <div className="flex-1">
                                            <div className="font-medium">{entry.nickname}</div>
                                        </div>
                                        <div className="text-xl font-bold text-gradient">
                                            {entry.score}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>

                {/* Players Sidebar */}
                <div className="card h-fit">
                    <h2 className="text-lg font-semibold mb-4">Players</h2>
                    {playerList.length === 0 ? (
                        <div className="text-center py-8 text-[var(--foreground-muted)]">
                            <div className="text-3xl mb-2">👥</div>
                            <p>Waiting for players...</p>
                            <p className="text-xs mt-2">Share the room code to invite</p>
                        </div>
                    ) : (
                        <div className="space-y-2">
                            {playerList.map((player) => (
                                <div
                                    key={player.id}
                                    className="flex items-center gap-3 p-3 rounded-lg bg-[var(--background-elevated)]"
                                >
                                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-[var(--accent-start)] to-[var(--accent-end)] flex items-center justify-center text-sm font-bold">
                                        {player.nickname.charAt(0).toUpperCase()}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <div className="font-medium truncate">{player.nickname}</div>
                                        {player.currentQuestion && (
                                            <div className="text-xs text-[var(--foreground-muted)]">
                                                On {player.currentQuestion}
                                            </div>
                                        )}
                                    </div>
                                    <span className={`badge ${player.status === 'joined' ? 'badge-neutral' :
                                            player.status === 'answering' ? 'badge-warning' :
                                                'badge-success'
                                        }`}>
                                        {player.status}
                                    </span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
