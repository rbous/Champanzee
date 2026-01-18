'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { auth } from '@/lib/api';

export default function HostLogin() {
    const router = useRouter();
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!username.trim() || !password.trim()) {
            setError('Please enter username and password');
            return;
        }

        setLoading(true);
        setError('');

        try {
            await auth.login(username, password);
            router.push('/host');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Invalid credentials');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex flex-col items-center justify-center px-4">
            {/* Back Link */}
            <div className="fixed top-6 left-6">
                <a href="/" className="btn btn-ghost text-sm">
                    ← Back
                </a>
            </div>

            {/* Login Card */}
            <div className="card w-full max-w-md animate-slide-up">
                <div className="text-center mb-8">
                    <h1 className="text-2xl font-bold mb-2">Host Login</h1>
                    <p className="text-[var(--foreground-muted)] text-sm">
                        Sign in to create and manage surveys
                    </p>
                </div>

                <form onSubmit={handleLogin} className="space-y-4">
                    <div>
                        <label className="input-label">Username</label>
                        <input
                            type="text"
                            className="input"
                            placeholder="admin"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            autoComplete="username"
                        />
                    </div>

                    <div>
                        <label className="input-label">Password</label>
                        <input
                            type="password"
                            className="input"
                            placeholder="••••••••"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            autoComplete="current-password"
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
                        className="btn btn-primary w-full"
                    >
                        {loading ? (
                            <span className="flex items-center gap-2">
                                <div className="spinner" style={{ width: 20, height: 20 }} />
                                Signing in...
                            </span>
                        ) : (
                            'Sign In'
                        )}
                    </button>
                </form>
            </div>
        </div>
    );
}
