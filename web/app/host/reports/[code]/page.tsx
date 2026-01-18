'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { reports, type RoomSnapshot, type AIReport } from '@/lib/api';

type Tab = 'snapshot' | 'ai';

export default function ReportsPage() {
    const params = useParams();
    const code = params.code as string;

    const [activeTab, setActiveTab] = useState<Tab>('snapshot');
    const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
    const [aiReport, setAiReport] = useState<AIReport | null>(null);
    const [loadingSnapshot, setLoadingSnapshot] = useState(true);
    const [loadingAI, setLoadingAI] = useState(false);
    const [pollingAI, setPollingAI] = useState(false);

    useEffect(() => {
        loadSnapshot();
    }, [code]);

    const loadSnapshot = async () => {
        try {
            const data = await reports.getSnapshot(code);
            setSnapshot(data);
        } catch (err) {
            console.error('Failed to load snapshot:', err);
        } finally {
            setLoadingSnapshot(false);
        }
    };

    const triggerAIReport = async () => {
        setLoadingAI(true);
        try {
            await reports.triggerAIReport(code);
            setPollingAI(true);
            pollAIReport();
        } catch (err) {
            console.error('Failed to trigger AI report:', err);
        } finally {
            setLoadingAI(false);
        }
    };

    const pollAIReport = async () => {
        try {
            const data = await reports.getAIReport(code);
            setAiReport(data);

            if (data.status === 'pending') {
                setTimeout(pollAIReport, 3000);
            } else {
                setPollingAI(false);
            }
        } catch (err) {
            console.error('Failed to get AI report:', err);
            setPollingAI(false);
        }
    };

    return (
        <div className="min-h-screen p-6">
            {/* Header */}
            <header className="flex items-center justify-between mb-8">
                <div className="flex items-center gap-4">
                    <a href="/host" className="btn btn-ghost">← Dashboard</a>
                    <div>
                        <h1 className="text-2xl font-bold">Room Report</h1>
                        <p className="text-[var(--foreground-muted)] text-sm">
                            Room: <span className="font-mono">{code}</span>
                        </p>
                    </div>
                </div>
            </header>

            {/* Tabs */}
            <div className="flex gap-2 mb-6">
                <button
                    onClick={() => setActiveTab('snapshot')}
                    className={`btn ${activeTab === 'snapshot' ? 'btn-primary' : 'btn-secondary'}`}
                >
                    📊 Snapshot
                </button>
                <button
                    onClick={() => {
                        setActiveTab('ai');
                        if (!aiReport) triggerAIReport();
                    }}
                    className={`btn ${activeTab === 'ai' ? 'btn-primary' : 'btn-secondary'}`}
                >
                    🤖 AI Insights
                </button>
            </div>

            {/* Snapshot Tab */}
            {activeTab === 'snapshot' && (
                <div className="space-y-6 animate-fade-in">
                    {loadingSnapshot ? (
                        <div className="card flex items-center justify-center py-12">
                            <div className="spinner" style={{ width: 40, height: 40 }} />
                        </div>
                    ) : snapshot ? (
                        <>
                            {/* Overview Cards */}
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                <div className="card text-center">
                                    <div className="text-3xl font-bold text-gradient mb-1">
                                        {Math.round((snapshot.completionRate || 0) * 100)}%
                                    </div>
                                    <div className="text-sm text-[var(--foreground-muted)]">Completion Rate</div>
                                </div>
                                <div className="card text-center">
                                    <div className="text-3xl font-bold mb-1">
                                        {snapshot.leaderboard?.length || 0}
                                    </div>
                                    <div className="text-sm text-[var(--foreground-muted)]">Total Players</div>
                                </div>
                                <div className="card text-center">
                                    <div className="text-3xl font-bold text-[var(--warning)] mb-1">
                                        {Math.round((snapshot.skipRate || 0) * 100)}%
                                    </div>
                                    <div className="text-sm text-[var(--foreground-muted)]">Skip Rate</div>
                                </div>
                            </div>

                            {/* Leaderboard */}
                            <div className="card">
                                <h2 className="text-lg font-semibold mb-4">Final Leaderboard</h2>
                                {snapshot.leaderboard?.length > 0 ? (
                                    <div className="space-y-2">
                                        {snapshot.leaderboard.slice(0, 10).map((entry, i) => (
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
                                ) : (
                                    <p className="text-[var(--foreground-muted)] text-center py-4">No players</p>
                                )}
                            </div>

                            {/* Per-Question Breakdown */}
                            <div className="card">
                                <h2 className="text-lg font-semibold mb-4">Question Breakdown</h2>
                                <div className="space-y-6">
                                    {snapshot.questionProfiles?.map((q) => (
                                        <div key={q.key} className="p-4 rounded-lg bg-[var(--background-elevated)]">
                                            <div className="flex items-center gap-2 mb-2">
                                                <span className="badge badge-neutral">{q.key}</span>
                                                <span className="badge badge-neutral">{q.type}</span>
                                            </div>
                                            <p className="text-sm mb-4">{q.prompt}</p>

                                            {/* Stats */}
                                            <div className="grid grid-cols-3 gap-4 mb-4">
                                                <div>
                                                    <div className="text-lg font-bold text-[var(--success)]">{q.satCount}</div>
                                                    <div className="text-xs text-[var(--foreground-muted)]">Satisfactory</div>
                                                </div>
                                                <div>
                                                    <div className="text-lg font-bold text-[var(--warning)]">{q.unsatCount}</div>
                                                    <div className="text-xs text-[var(--foreground-muted)]">Unsatisfactory</div>
                                                </div>
                                                <div>
                                                    <div className="text-lg font-bold text-[var(--error)]">{q.skipCount}</div>
                                                    <div className="text-xs text-[var(--foreground-muted)]">Skipped</div>
                                                </div>
                                            </div>

                                            {/* Rating histogram */}
                                            {q.ratingHist && q.ratingHist.length > 0 && (
                                                <div className="mb-4">
                                                    <div className="text-xs text-[var(--foreground-muted)] mb-2">Rating Distribution</div>
                                                    <div className="flex items-end gap-1 h-16">
                                                        {q.ratingHist.map((count, i) => {
                                                            const max = Math.max(...q.ratingHist!);
                                                            const height = max > 0 ? (count / max) * 100 : 0;
                                                            return (
                                                                <div key={i} className="flex-1 flex flex-col items-center">
                                                                    <div
                                                                        className="histogram-bar w-full"
                                                                        style={{ height: `${height}%`, minHeight: count > 0 ? 4 : 0 }}
                                                                    />
                                                                    <span className="text-xs mt-1 text-[var(--foreground-muted)]">{i + 1}</span>
                                                                </div>
                                                            );
                                                        })}
                                                    </div>
                                                    {q.mean && (
                                                        <div className="text-xs text-[var(--foreground-muted)] mt-2">
                                                            Mean: {q.mean.toFixed(1)} | Median: {q.median?.toFixed(1)}
                                                        </div>
                                                    )}
                                                </div>
                                            )}

                                            {/* Themes */}
                                            {q.topThemes?.length > 0 && (
                                                <div className="mb-3">
                                                    <div className="text-xs text-[var(--foreground-muted)] mb-2">Top Themes</div>
                                                    <div className="flex flex-wrap gap-2">
                                                        {q.topThemes.map((theme, i) => (
                                                            <span key={i} className="theme-tag">{theme}</span>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}

                                            {/* Misunderstandings */}
                                            {q.misunderstandings?.length > 0 && (
                                                <div>
                                                    <div className="text-xs text-[var(--foreground-muted)] mb-2">Common Misunderstandings</div>
                                                    <ul className="text-sm space-y-1">
                                                        {q.misunderstandings.map((m, i) => (
                                                            <li key={i} className="text-[var(--warning)]">• {m}</li>
                                                        ))}
                                                    </ul>
                                                </div>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </div>

                            {/* Room Summary */}
                            {snapshot.roomSummary && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">Room Summary</h2>

                                    {snapshot.roomSummary.topThemes?.length > 0 && (
                                        <div className="mb-4">
                                            <div className="text-sm text-[var(--foreground-muted)] mb-2">Top Themes</div>
                                            <div className="flex flex-wrap gap-2">
                                                {snapshot.roomSummary.topThemes.map((theme, i) => (
                                                    <span key={i} className="theme-tag">{theme}</span>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {snapshot.roomSummary.contrasts?.length > 0 && (
                                        <div className="mb-4">
                                            <div className="text-sm text-[var(--foreground-muted)] mb-2">Key Contrasts</div>
                                            <div className="space-y-2">
                                                {snapshot.roomSummary.contrasts.map((c, i) => (
                                                    <div key={i} className="p-3 rounded-lg bg-[var(--background-elevated)]">
                                                        <div className="font-medium mb-1">{c.axis}</div>
                                                        <div className="text-sm text-[var(--foreground-muted)]">
                                                            {c.sideA} vs {c.sideB}
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {snapshot.roomSummary.frictionPoints?.length > 0 && (
                                        <div>
                                            <div className="text-sm text-[var(--foreground-muted)] mb-2">Friction Points</div>
                                            <ul className="text-sm space-y-1">
                                                {snapshot.roomSummary.frictionPoints.map((f, i) => (
                                                    <li key={i} className="text-[var(--error)]">• {f}</li>
                                                ))}
                                            </ul>
                                        </div>
                                    )}
                                </div>
                            )}
                        </>
                    ) : (
                        <div className="card text-center py-12">
                            <p className="text-[var(--foreground-muted)]">No snapshot data available</p>
                        </div>
                    )}
                </div>
            )}

            {/* AI Report Tab */}
            {activeTab === 'ai' && (
                <div className="animate-fade-in">
                    {loadingAI || pollingAI ? (
                        <div className="card text-center py-12">
                            <div className="spinner mx-auto mb-4" style={{ width: 40, height: 40 }} />
                            <p className="text-[var(--foreground-muted)]">
                                {loadingAI ? 'Triggering AI analysis...' : 'Generating insights...'}
                            </p>
                            <p className="text-xs text-[var(--foreground-muted)] mt-2">This may take a moment</p>
                        </div>
                    ) : aiReport?.status === 'ready' ? (
                        <div className="space-y-6">
                            {/* Executive Summary */}
                            {aiReport.executiveSummary && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">📋 Executive Summary</h2>
                                    <ul className="space-y-2">
                                        {aiReport.executiveSummary.map((point, i) => (
                                            <li key={i} className="flex gap-2">
                                                <span className="text-[var(--accent)]">•</span>
                                                <span>{point}</span>
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {/* Key Themes */}
                            {aiReport.keyThemes && aiReport.keyThemes.length > 0 && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">🎯 Key Themes</h2>
                                    <div className="space-y-4">
                                        {aiReport.keyThemes.map((theme, i) => (
                                            <div key={i} className="p-4 rounded-lg bg-[var(--background-elevated)]">
                                                <div className="flex items-center gap-3 mb-2">
                                                    <span className="theme-tag">{theme.name}</span>
                                                    <span className="text-sm text-[var(--foreground-muted)]">
                                                        {theme.percentage}% of responses
                                                    </span>
                                                </div>
                                                <p className="text-sm mb-3">{theme.meaning}</p>
                                                {theme.evidence && (
                                                    <div className="text-xs text-[var(--foreground-muted)]">
                                                        Evidence: {theme.evidence.join('; ')}
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Contrasts */}
                            {aiReport.contrasts && aiReport.contrasts.length > 0 && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">⚖️ Key Contrasts</h2>
                                    <div className="space-y-3">
                                        {aiReport.contrasts.map((c, i) => (
                                            <div key={i} className="p-4 rounded-lg bg-[var(--background-elevated)]">
                                                <div className="font-medium mb-2">{c.axis}</div>
                                                <div className="grid grid-cols-2 gap-4 text-sm">
                                                    <div className="p-2 rounded bg-[var(--success-bg)]">{c.sideA}</div>
                                                    <div className="p-2 rounded bg-[var(--warning-bg)]">{c.sideB}</div>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Recommendations */}
                            {aiReport.recommendedNextQuestions && aiReport.recommendedNextQuestions.length > 0 && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">💡 Recommended Next Questions</h2>
                                    <ul className="space-y-2">
                                        {aiReport.recommendedNextQuestions.map((q, i) => (
                                            <li key={i} className="p-3 rounded-lg bg-[var(--background-elevated)]">
                                                {q}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {/* Recommended Edits */}
                            {aiReport.recommendedEdits && aiReport.recommendedEdits.length > 0 && (
                                <div className="card">
                                    <h2 className="text-lg font-semibold mb-4">✏️ Suggested Question Edits</h2>
                                    <div className="space-y-4">
                                        {aiReport.recommendedEdits.map((edit, i) => (
                                            <div key={i} className="p-4 rounded-lg bg-[var(--background-elevated)]">
                                                <div className="badge badge-neutral mb-2">{edit.questionKey}</div>
                                                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-2">
                                                    <div>
                                                        <div className="text-xs text-[var(--foreground-muted)] mb-1">Original</div>
                                                        <div className="text-sm p-2 rounded bg-[var(--error-bg)]">{edit.original}</div>
                                                    </div>
                                                    <div>
                                                        <div className="text-xs text-[var(--foreground-muted)] mb-1">Suggested</div>
                                                        <div className="text-sm p-2 rounded bg-[var(--success-bg)]">{edit.suggested}</div>
                                                    </div>
                                                </div>
                                                <div className="text-xs text-[var(--foreground-muted)]">
                                                    Reason: {edit.reason}
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    ) : aiReport?.status === 'error' ? (
                        <div className="card text-center py-12">
                            <div className="text-4xl mb-4">❌</div>
                            <p className="text-[var(--error)]">Failed to generate AI report</p>
                            <button onClick={triggerAIReport} className="btn btn-primary mt-4">
                                Retry
                            </button>
                        </div>
                    ) : (
                        <div className="card text-center py-12">
                            <div className="text-4xl mb-4">🤖</div>
                            <h3 className="text-lg font-semibold mb-2">AI Insights</h3>
                            <p className="text-[var(--foreground-muted)] mb-4">
                                Generate deep insights and recommendations
                            </p>
                            <button onClick={triggerAIReport} className="btn btn-primary">
                                Generate Report
                            </button>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
