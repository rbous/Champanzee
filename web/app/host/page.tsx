'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { auth, surveys, rooms, type Survey } from '@/lib/api';

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
                <div className="spinner" style={{ width: 40, height: 40 }} />
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6">
            {/* Header */}
            <header className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-2xl font-bold">
                        <span className="text-gradient">Dashboard</span>
                    </h1>
                    <p className="text-[var(--foreground-muted)] text-sm mt-1">
                        Manage your surveys and rooms
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <a href="/host/surveys/new" className="btn btn-primary">
                        + New Survey
                    </a>
                    <button onClick={handleLogout} className="btn btn-ghost">
                        Logout
                    </button>
                </div>
            </header>

            {/* Surveys Grid */}
            {surveyList.length === 0 ? (
                <div className="card text-center py-12">
                    <div className="text-4xl mb-4">📋</div>
                    <h3 className="text-lg font-semibold mb-2">No surveys yet</h3>
                    <p className="text-[var(--foreground-muted)] mb-4">
                        Create your first AI-powered survey to get started
                    </p>
                    <a href="/host/surveys/new" className="btn btn-primary">
                        Create Survey
                    </a>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 stagger-children">
                    {surveyList.map((survey) => (
                        <div key={survey.id} className="card">
                            <h3 className="text-lg font-semibold mb-2">{survey.title}</h3>
                            <p className="text-[var(--foreground-muted)] text-sm mb-4 line-clamp-2">
                                {survey.intent}
                            </p>

                            <div className="flex items-center gap-2 mb-4">
                                <span className="badge badge-neutral">
                                    {survey.questions?.length || 0} questions
                                </span>
                                <span className="badge badge-neutral">
                                    Max {survey.settings?.maxFollowUps || 0} follow-ups
                                </span>
                            </div>

                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => handleCreateRoom(survey.id)}
                                    disabled={creatingRoom === survey.id}
                                    className="btn btn-primary flex-1"
                                >
                                    {creatingRoom === survey.id ? (
                                        <span className="flex items-center gap-2">
                                            <div className="spinner" style={{ width: 16, height: 16 }} />
                                            Creating...
                                        </span>
                                    ) : (
                                        'Start Room'
                                    )}
                                </button>
                                <a
                                    href={`/host/surveys/${survey.id}`}
                                    className="btn btn-secondary"
                                >
                                    Edit
                                </a>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
