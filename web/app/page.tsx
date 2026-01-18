'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { rooms } from '@/lib/api';

export default function Home() {
  const router = useRouter();
  const [roomCode, setRoomCode] = useState('');
  const [nickname, setNickname] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleJoin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!roomCode.trim() || !nickname.trim()) {
      setError('Please enter both room code and nickname');
      return;
    }

    setLoading(true);
    setError('');

    try {
      await rooms.join(roomCode.toUpperCase(), nickname);
      router.push(`/play/${roomCode.toUpperCase()}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to join room');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-4">
      {/* Hero Section */}
      <div className="text-center mb-12 animate-fade-in">
        <h1 className="text-5xl font-bold mb-4">
          <span className="text-gradient">2026 Champs</span>
        </h1>
        <p className="text-lg text-[var(--foreground-muted)] max-w-md">
          AI-powered surveys with real-time feedback and deep insights
        </p>
      </div>

      {/* Join Room Card */}
      <div className="card w-full max-w-md animate-slide-up">
        <h2 className="text-xl font-semibold mb-6 text-center">Join a Room</h2>

        <form onSubmit={handleJoin} className="space-y-4">
          <div>
            <label className="input-label">Room Code</label>
            <input
              type="text"
              className="input text-center text-2xl font-mono tracking-widest uppercase"
              placeholder="ABC123"
              value={roomCode}
              onChange={(e) => setRoomCode(e.target.value.toUpperCase())}
              maxLength={8}
              autoComplete="off"
            />
          </div>

          <div>
            <label className="input-label">Your Nickname</label>
            <input
              type="text"
              className="input"
              placeholder="Enter your name"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              maxLength={20}
              autoComplete="off"
            />
          </div>

          {error && (
            <div className="text-[var(--error)] text-sm text-center animate-fade-in">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="btn btn-primary w-full text-lg py-4"
          >
            {loading ? (
              <span className="flex items-center gap-2">
                <div className="spinner" style={{ width: 20, height: 20 }} />
                Joining...
              </span>
            ) : (
              'Join Room'
            )}
          </button>
        </form>
      </div>

      {/* Host Link */}
      <div className="mt-8 animate-fade-in" style={{ animationDelay: '0.3s' }}>
        <a
          href="/host/login"
          className="btn btn-ghost text-sm"
        >
          Host a Survey →
        </a>
      </div>

      {/* Decorative elements */}
      <div className="fixed bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[var(--accent)] to-transparent opacity-30" />
    </div>
  );
}
