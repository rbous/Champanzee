'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { player, getPlayerWebSocketUrl, type Question, type SubmitAnswerResponse } from '@/lib/api';
import { usePlayerWebSocket, type NextQuestionEvent, type EvaluationResultEvent } from '@/hooks/useWebSocket';

type GameState = 'loading' | 'answering' | 'evaluated' | 'done';

export default function PlayerGame() {
    const params = useParams();
    const router = useRouter();
    const code = params.code as string;

    const [gameState, setGameState] = useState<GameState>('loading');
    const [currentQuestion, setCurrentQuestion] = useState<Question | null>(null);
    const [answer, setAnswer] = useState('');
    const [degreeValue, setDegreeValue] = useState(3);
    const [submitting, setSubmitting] = useState(false);
    const [result, setResult] = useState<SubmitAnswerResponse | null>(null);
    const [totalPoints, setTotalPoints] = useState(0);
    const [attemptCount, setAttemptCount] = useState(0);
    const [wsUrl, setWsUrl] = useState<string | null>(null);
    const [allowSkipAfter, setAllowSkipAfter] = useState(1);

    useEffect(() => {
        const storedSkip = localStorage.getItem('room_allow_skip_after');
        if (storedSkip) {
            setAllowSkipAfter(parseInt(storedSkip, 10));
        }
    }, []);

    // WebSocket handlers
    const handleNextQuestion = useCallback((event: NextQuestionEvent) => {
        setCurrentQuestion(event.question);
        setAnswer('');
        setDegreeValue(3);
        setResult(null);
        setAttemptCount(0);
        setGameState('answering');
    }, []);

    const handleEvaluationResult = useCallback((event: EvaluationResultEvent) => {
        setTotalPoints(prev => prev + event.points);
    }, []);

    usePlayerWebSocket(wsUrl, {
        onNextQuestion: handleNextQuestion,
        onEvaluationResult: handleEvaluationResult,
    });

    useEffect(() => {
        const url = getPlayerWebSocketUrl(code);
        setWsUrl(url);

        // Load current question on mount
        loadCurrentQuestion();
    }, [code]);

    const loadCurrentQuestion = async () => {
        try {
            const question = await player.getCurrentQuestion(code);
            if (question) {
                setCurrentQuestion(question);
                setGameState('answering');
            }
        } catch (err) {
            console.error('Failed to load question:', err);
        }
    };

    const generateClientAttemptId = () => {
        return crypto.randomUUID();
    };

    const handleSubmit = async () => {
        if (!currentQuestion) return;

        const hasAnswer = currentQuestion.type === 'DEGREE' || answer.trim().length > 0;
        if (!hasAnswer) return;

        setSubmitting(true);

        try {
            const response = await player.submitAnswer(code, {
                questionKey: currentQuestion.key,
                textAnswer: currentQuestion.type === 'ESSAY' ? answer : undefined,
                degreeValue: currentQuestion.type === 'DEGREE' ? degreeValue : undefined,
                clientAttemptId: generateClientAttemptId(),
            });

            setResult(response);
            setTotalPoints(prev => prev + response.pointsEarned);
            setAttemptCount(prev => prev + 1);
            setGameState('evaluated');

            // If there's a next question or follow-up, prepare for it
            if (response.nextQuestion) {
                setTimeout(() => {
                    setCurrentQuestion(response.nextQuestion);
                    setAnswer('');
                    setDegreeValue(3);
                    setResult(null);
                    setAttemptCount(0);
                    setGameState('answering');
                }, 2500);
            } else if (response.followUp) {
                setTimeout(() => {
                    setCurrentQuestion(response.followUp);
                    setAnswer('');
                    setResult(null);
                    setGameState('answering');
                }, 2500);
            } else if (!response.nextQuestion && !response.followUp) {
                setTimeout(() => {
                    setGameState('done');
                }, 2500);
            }
        } catch (err) {
            console.error('Failed to submit:', err);
            alert('Failed to submit answer. Please try again.');
        } finally {
            setSubmitting(false);
        }
    };

    const handleSkip = async () => {
        if (!currentQuestion) return;

        try {
            await player.skipQuestion(code, currentQuestion.key);
            // The WebSocket will push the next question
            setAnswer('');
            setDegreeValue(3);
            setResult(null);
        } catch (err) {
            console.error('Failed to skip:', err);
        }
    };

    // Loading state
    if (gameState === 'loading') {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center">
                    <div className="spinner mx-auto mb-4" style={{ width: 40, height: 40 }} />
                    <p className="text-[var(--foreground-muted)]">Loading question...</p>
                </div>
            </div>
        );
    }

    // Done state
    if (gameState === 'done') {
        return (
            <div className="min-h-screen flex items-center justify-center p-6">
                <div className="card text-center max-w-md animate-slide-up">
                    <div className="text-5xl mb-4">🎉</div>
                    <h1 className="text-2xl font-bold mb-2">You&apos;re Done!</h1>
                    <p className="text-[var(--foreground-muted)] mb-6">
                        Thanks for completing the survey
                    </p>
                    <div className="text-4xl font-bold text-gradient mb-6">
                        {totalPoints} points
                    </div>
                    <button
                        onClick={() => router.push('/')}
                        className="btn btn-secondary"
                    >
                        Back to Home
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6 flex flex-col">
            {/* Header */}
            <header className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-4">
                    <span className="badge badge-neutral">Room: {code}</span>
                </div>
                <div className="text-right">
                    <div className="text-2xl font-bold text-gradient">{totalPoints}</div>
                    <div className="text-xs text-[var(--foreground-muted)]">points</div>
                </div>
            </header>

            {/* Question Card */}
            <div className="flex-1 flex items-center justify-center">
                <div className="w-full max-w-2xl">
                    {/* Result display */}
                    {gameState === 'evaluated' && result && (
                        <div className={`card mb-4 animate-slide-up ${result.resolution === 'SAT'
                            ? 'border-[var(--success)]'
                            : 'border-[var(--warning)]'
                            }`} style={{ borderWidth: 2 }}>
                            <div className="flex items-center gap-3 mb-3">
                                <span className={`badge ${result.resolution === 'SAT' ? 'badge-success' : 'badge-warning'
                                    }`}>
                                    {result.resolution === 'SAT' ? '✓ Satisfactory' : '⚡ Needs More'}
                                </span>
                                <span className="text-xl font-bold text-gradient">
                                    +{result.pointsEarned} pts
                                </span>
                            </div>
                            <p className="text-[var(--foreground-muted)]">{result.evalSummary}</p>
                            {result.followUp && (
                                <p className="text-sm mt-2 text-[var(--accent)]">
                                    Follow-up question coming up...
                                </p>
                            )}
                            {result.nextQuestion && (
                                <p className="text-sm mt-2 text-[var(--success)]">
                                    Next question loading...
                                </p>
                            )}
                        </div>
                    )}

                    {/* Question */}
                    <div className="card animate-fade-in">
                        {currentQuestion && (
                            <>
                                <div className="flex items-center gap-2 mb-4">
                                    <span className="badge badge-neutral">{currentQuestion.key}</span>
                                    <span className="badge badge-neutral">
                                        {currentQuestion.type === 'ESSAY' ? '📝 Essay' : '📊 Rating'}
                                    </span>
                                    {currentQuestion.pointsMax && (
                                        <span className="text-sm text-[var(--foreground-muted)]">
                                            Up to {currentQuestion.pointsMax} pts
                                        </span>
                                    )}
                                </div>

                                <h2 className="text-xl font-semibold mb-6">{currentQuestion.prompt}</h2>

                                {/* Essay Input */}
                                {currentQuestion.type === 'ESSAY' && gameState === 'answering' && (
                                    <textarea
                                        className="input mb-4"
                                        placeholder="Type your answer here..."
                                        value={answer}
                                        onChange={(e) => setAnswer(e.target.value)}
                                        rows={6}
                                        disabled={submitting}
                                        autoFocus
                                    />
                                )}

                                {/* Degree Slider */}
                                {currentQuestion.type === 'DEGREE' && gameState === 'answering' && (
                                    <div className="mb-6">
                                        <div className="flex justify-between text-sm text-[var(--foreground-muted)] mb-2">
                                            <span>{currentQuestion.scaleMin || 1}</span>
                                            <span className="text-2xl font-bold text-gradient">{degreeValue}</span>
                                            <span>{currentQuestion.scaleMax || 5}</span>
                                        </div>
                                        <input
                                            type="range"
                                            className="slider"
                                            min={currentQuestion.scaleMin || 1}
                                            max={currentQuestion.scaleMax || 5}
                                            value={degreeValue}
                                            onChange={(e) => setDegreeValue(parseInt(e.target.value))}
                                            disabled={submitting}
                                        />
                                    </div>
                                )}

                                {/* Actions */}
                                {gameState === 'answering' && (
                                    <div className="flex items-center gap-3">
                                        <button
                                            onClick={handleSubmit}
                                            disabled={submitting || (currentQuestion.type === 'ESSAY' && !answer.trim())}
                                            className="btn btn-primary flex-1 py-4"
                                        >
                                            {submitting ? (
                                                <span className="flex items-center gap-2">
                                                    <div className="spinner" style={{ width: 20, height: 20 }} />
                                                    Submitting...
                                                </span>
                                            ) : (
                                                'Submit Answer'
                                            )}
                                        </button>

                                        {attemptCount >= allowSkipAfter && (
                                            <button
                                                onClick={handleSkip}
                                                className="btn btn-ghost"
                                            >
                                                Skip →
                                            </button>
                                        )}
                                    </div>
                                )}
                            </>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
