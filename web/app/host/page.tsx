'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { auth, surveys, rooms, type Survey } from '@/lib/api';
import LobbyBackground from '@/components/LobbyBackground';

export default function HostDashboard() {
    const router = useRouter();
    const [surveyList, setSurveyList] = useState<Survey[]>([]);
    const [loading, setLoading] = useState(true);
    const [creatingRoom, setCreatingRoom] = useState<string | null>(null);

    useEffect(() => {
        if (!auth.isLoggedIn()) {
            router.push('/host/login');
            return;
        }

        loadSurveys();
    }, [router]);

    const loadSurveys = async () => {
        try {
            const data = await surveys.list();
            setSurveyList(data || []);
        } catch (err) {
            console.error('Failed to load surveys:', err);
        } finally {
            setLoading(false);
        }
    };

    const handleCreateRoom = async (surveyId: string) => {
        setCreatingRoom(surveyId);
        try {
            const { roomCode } = await rooms.create(surveyId);
            router.push(`/host/rooms/${roomCode}`);
        } catch (err) {
            console.error('Failed to create room:', err);
            alert('Failed to create room');
        } finally {
            setCreatingRoom(null);
        }
    };

    const handleLogout = () => {
        auth.logout();
        router.push('/host/login');
    };

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center">
                    <div className="spinner mx-auto mb-4" style={{ width: 40, height: 40 }} />
                    <p className="font-bold text-[var(--text-muted)] mb-4">Loading the magic...</p>
                    <button
                        onClick={handleLogout}
                        className="btn btn-ghost text-xs"
                    >
                        Logout & Exit
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6 relative">
            <LobbyBackground />

            <div className="relative z-10 max-w-6xl mx-auto">
                {/* Header */}
                <header className="flex items-center justify-between mb-8">
                    <div className="text-center md:text-left">
                        <h1 className="text-4xl font-black mb-2">
                            <span className="text-party-gradient drop-shadow-sm">🎉 Dashboard</span>
                        </h1>
                        <p className="text-lg font-bold text-[var(--text-muted)]">
                            Manage your surveys and start some fun!
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <a href="/host/surveys/new" className="btn btn-green hover:scale-105">
                            + New Survey
                        </a>
                        <button onClick={handleLogout} className="btn btn-ghost hover:rotate-2">
                            👋 Logout
                        </button>
                    </div>
                </header>

                {/* Surveys Grid */}
                {surveyList.length === 0 ? (
                    <div className="card-party text-center py-16 animate-slide-up">
                        <div className="text-6xl mb-6 animate-bounce-slow">📋</div>
                        <h3 className="text-2xl font-black mb-4">No surveys yet!</h3>
                        <p className="text-lg font-bold text-[var(--text-muted)] mb-8">
                            Create your first AI-powered survey to get started
                        </p>
                        <a href="/host/surveys/new" className="btn btn-primary text-xl py-4 px-8 hover:scale-105">
                            🚀 Create Survey
                        </a>
                    </div>
                ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                        {surveyList.map((survey, index) => (
                            <div
                                key={survey.id}
                                className="card-party animate-pop-in"
                                style={{ animationDelay: `${index * 0.1}s` }}
                            >
                                <div className="absolute -top-3 -right-3 text-2xl rotate-12">📝</div>

                                <h3 className="text-xl font-black mb-3 text-purple">
                                    {survey.title}
                                </h3>
                                <p className="text-[var(--text-muted)] font-bold mb-4 line-clamp-2">
                                    {survey.intent}
                                </p>

                                <div className="flex items-center gap-2 mb-6">
                                    <span className="badge-party" style={{ borderColor: 'var(--color-blue)' }}>
                                        {survey.questions?.length || 0} questions
                                    </span>
                                    <span className="badge-party" style={{ borderColor: 'var(--color-yellow)' }}>
                                        Max {survey.settings?.maxFollowUps || 0} follow-ups
                                    </span>
                                </div>

                                <div className="flex items-center gap-3">
                                    <button
                                        onClick={() => handleCreateRoom(survey.id)}
                                        disabled={creatingRoom === survey.id}
                                        className="btn btn-green flex-1 hover:scale-105"
                                    >
                                        {creatingRoom === survey.id ? (
                                            <span className="flex items-center gap-2">
                                                <div className="spinner border-white" style={{ width: 16, height: 16 }} />
                                                Creating...
                                            </span>
                                        ) : (
                                            '🎮 Start Room'
                                        )}
                                    </button>
                                    <a
                                        href={`/host/surveys/${survey.id}`}
                                        className="btn btn-secondary hover:rotate-2"
                                    >
                                        ✏️ Edit
                                    </a>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
