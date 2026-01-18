'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { player, getPlayerWebSocketUrl, type Question, type SubmitAnswerResponse } from '@/lib/api';
import { usePlayerWebSocket, type NextQuestionEvent, type EvaluationResultEvent, type RoomEndedEvent } from '@/hooks/useWebSocket';

type GameState = 'loading' | 'answering' | 'evaluated' | 'done' | 'waiting_for_ai';

export default function PlayerGame() {
    const params = useParams();
    const router = useRouter();
    const code = params.code as string;

    const [gameState, setGameState] = useState<GameState>('loading');
    const [currentQuestion, setCurrentQuestion] = useState<Question | null>(null);
    const [answer, setAnswer] = useState('');
    const [degreeValue, setDegreeValue] = useState(3);
    const [submitting, setSubmitting] = useState(false);
    const [lastAttemptId, setLastAttemptId] = useState<string | null>(null);
    const [result, setResult] = useState<SubmitAnswerResponse | null>(null);

    const KahootColors = [
        'bg-[#e21b3c] hover:bg-[#c61734]', // Red
        'bg-[#1368ce] hover:bg-[#115ab3]', // Blue
        'bg-[#d89e00] hover:bg-[#b08100]', // Yellow
        'bg-[#26890c] hover:bg-[#1f6f0a]', // Green
    ];

    const MCQControl = ({ options, onSelect, disabled }: { options: string[], onSelect: (index: number) => void, disabled: boolean }) => {
        return (
            <div className="grid grid-cols-2 gap-4 mt-8">
                {options.map((opt, i) => (
                    <button
                        key={i}
                        onClick={() => onSelect(i)}
                        disabled={disabled}
                        className={`${KahootColors[i % KahootColors.length]} text-white p-8 rounded-xl text-center font-bold text-xl shadow-lg transition-transform active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed min-h-[120px] flex items-center justify-center`}
                    >
                        {opt}
                    </button>
                ))}
            </div>
        );
    };
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
        console.log("Received evaluation result:", event);
        setTotalPoints(prev => prev + (event.pointsEarned || 0));

        // Map the event fields to a result object for consistency
        const fullResult: SubmitAnswerResponse = {
            status: 'EVALUATED',
            resolution: event.resolution,
            pointsEarned: event.pointsEarned,
            evalSummary: event.evalSummary,
            nextQuestion: event.nextQuestion || null,
            followUp: event.followUp || null,
        };

        setResult(fullResult);
        setGameState('evaluated');
        setSubmitting(false);

        // Logic for auto-advance or follow-up
        if (fullResult.resolution === 'SAT') {
            if (fullResult.nextQuestion) {
                setTimeout(() => {
                    setCurrentQuestion(fullResult.nextQuestion);
                    setAnswer('');
                    setDegreeValue(3);
                    setResult(null);
                    setAttemptCount(0);
                    setGameState('answering');
                }, 2500);
            } else if (fullResult.followUp) {
                setTimeout(() => {
                    setCurrentQuestion(fullResult.followUp);
                    setAnswer('');
                    setResult(null);
                    setGameState('answering');
                }, 2500);
            } else {
                setTimeout(() => {
                    setGameState('done');
                }, 2500);
            }
        } else if (fullResult.resolution === 'UNSAT') {
            // Stay
        } else if (!fullResult.nextQuestion && !fullResult.followUp) {
            setTimeout(() => {
                setGameState('done');
            }, 2500);
        }
    }, []);

    const { disconnect } = usePlayerWebSocket(wsUrl, {
        onNextQuestion: handleNextQuestion,
        onEvaluationResult: handleEvaluationResult,
        onAIThinking: () => {
            console.log("AI is thinking...");
            setGameState('waiting_for_ai');
        },
        onRoomStarted: () => {
            console.log("Room started! Loading question...");
            loadCurrentQuestion();
        },
        onRoomEnded: (event: RoomEndedEvent) => {
            console.log("Room ended!", event);
            // Disconnect WebSocket
            disconnect();
            // Redirect to home or show ended message
            setGameState('done');
            setTimeout(() => {
                alert('The room has ended. Thank you for participating!');
                router.push('/');
            }, 1000);
        },
    });

    useEffect(() => {
        const url = getPlayerWebSocketUrl(code);
        setWsUrl(url);

        // Load current question on mount
        loadCurrentQuestion();
    }, [code]);

    const loadCurrentQuestion = async () => {
        try {
            const data = await player.getCurrentQuestion(code);
            if (data.question) {
                setCurrentQuestion(data.question);
                setGameState('answering');
            }
            if (data.player) {
                setTotalPoints(data.player.score || 0);
            }
        } catch (err) {
            console.error('Failed to load question:', err);
        }
    };

    const generateClientAttemptId = () => {
        return crypto.randomUUID();
    };

    const PromptDisplay = ({ text }: { text: string }) => {
        // Remove markdown-style asterisks and render as a single header
        const cleanText = text.replace(/\*/g, '');
        return <h2 className="text-xl font-semibold mb-6 leading-tight">{cleanText}</h2>;
    };

    const handleSubmit = async () => {
        if (!currentQuestion) return;

        const hasAnswer = currentQuestion.type === 'DEGREE' || answer.trim().length > 0;
        if (!hasAnswer) return;

        setSubmitting(true);

        try {
            // This now returns immediately with "Submitted" status
            await player.submitAnswer(code, {
                questionKey: currentQuestion.key,
                textAnswer: currentQuestion.type === 'ESSAY' ? answer : undefined,
                degreeValue: currentQuestion.type === 'DEGREE' ? degreeValue : undefined,
                clientAttemptId: generateClientAttemptId(),
            });

            // Do NOT update result yet. Wait for WebSocket.
            // Show "Waiting for AI" state
            setGameState('waiting_for_ai');

        } catch (err) {
            console.error('Failed to submit:', err);
            alert('Failed to submit answer. Please try again.');
            setSubmitting(false); // Reset on error
        }
        // Do NOT setSubmitting(false) here, wait for WS or timeout
    };

    const handleSkip = async () => {
        if (!currentQuestion) return;

        try {
            const response = await player.skipQuestion(code, currentQuestion.key);

            if (response.done) {
                setGameState('done');
            } else if (response.nextQuestion) {
                setCurrentQuestion(response.nextQuestion);
                setAnswer('');
                setDegreeValue(3);
                setResult(null);
                setAttemptCount(0);
                setGameState('answering');
            }
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
                        {totalPoints || 0} points
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
                <div className="flex items-center gap-4">
                    <div className="text-right">
                        <div className="text-2xl font-bold text-gradient">{totalPoints || 0}</div>
                        <div className="text-xs text-[var(--foreground-muted)]">points</div>
                    </div>
                    <button
                        onClick={() => {
                            if (confirm('Are you sure you want to leave the quiz? Your progress is saved.')) {
                                localStorage.removeItem('player_token');
                                localStorage.removeItem('player_id');
                                localStorage.removeItem('room_code');
                                router.push('/');
                            }
                        }}
                        className="btn btn-ghost text-xs text-[var(--foreground-muted)]"
                    >
                        Leave
                    </button>
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

                                <PromptDisplay text={currentQuestion.prompt} />

                                {/* Essay Input */}
                                {currentQuestion.type === 'ESSAY' && (gameState === 'answering' || gameState === 'waiting_for_ai') && (
                                    <textarea
                                        className="input mb-4"
                                        placeholder="Type your answer here..."
                                        value={answer}
                                        onChange={(e) => setAnswer(e.target.value)}
                                        rows={6}
                                        disabled={submitting || gameState === 'waiting_for_ai'}
                                        autoFocus
                                    />
                                )}

                                {/* Waiting UI */}
                                {gameState === 'waiting_for_ai' && (
                                    <div className="flex items-center justify-center py-8 text-[var(--accent)] animate-pulse">
                                        <div className="spinner mr-3" />
                                        <span>AI is analyzing your answer...</span>
                                    </div>
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
                                            disabled={submitting || gameState !== 'answering'}
                                        />
                                    </div>
                                )}

                                {currentQuestion.type === 'MCQ' && currentQuestion.options && (
                                    <MCQControl
                                        options={currentQuestion.options}
                                        onSelect={(index) => {
                                            // Auto submit for MCQ
                                            const attemptId = Math.random().toString(36).substring(7);
                                            setLastAttemptId(attemptId);
                                            setSubmitting(true);
                                            player.submitAnswer(code, {
                                                questionKey: currentQuestion.key,
                                                optionIndex: index,
                                                clientAttemptId: attemptId,
                                            }).catch(err => {
                                                console.error('Submit MCQ failed:', err);
                                                setSubmitting(false);
                                            });
                                            setGameState('waiting_for_ai');
                                        }}
                                        disabled={submitting}
                                    />
                                )}

                                {/* Actions */}
                                {gameState === 'answering' && currentQuestion.type !== 'MCQ' && (
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

                                {/* Try Again State (Evaluated but UNSAT) */}
                                {gameState === 'evaluated' && result?.resolution === 'UNSAT' && (
                                    <div className="mt-6 flex gap-3">
                                        <button
                                            onClick={() => {
                                                setGameState('answering');
                                                setResult(null);
                                            }}
                                            className="btn btn-primary flex-1"
                                        >
                                            Try Again
                                        </button>

                                        {attemptCount >= allowSkipAfter && (
                                            <button
                                                onClick={handleSkip}
                                                className="btn btn-ghost"
                                            >
                                                Skip Question
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
