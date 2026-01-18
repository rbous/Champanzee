'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { player, getPlayerWebSocketUrl, type Question, type SubmitAnswerResponse } from '@/lib/api';
import { usePlayerWebSocket, type NextQuestionEvent, type EvaluationResultEvent, type RoomEndedEvent } from '@/hooks/useWebSocket';
import LobbyBackground from '@/components/LobbyBackground';
import GameBackground from '@/components/GameBackground';

type GameState = 'loading' | 'answering' | 'evaluated' | 'done' | 'waiting_for_ai' | 'waiting_for_start';

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

    const PartyThemes = [
        { bg: 'bg-[#e21b3c]', hover: 'hover:bg-[#f32d4e]', icon: '🔺' },
        { bg: 'bg-[#1368ce]', hover: 'hover:bg-[#257cf0]', icon: '🔷' },
        { bg: 'bg-[#d89e00]', hover: 'hover:bg-[#ffbd0a]', icon: '🟡' },
        { bg: 'bg-[#26890c]', hover: 'hover:bg-[#34a311]', icon: '🟩' },
    ];

    const MCQControl = ({ options, onSelect, disabled }: { options: string[], onSelect: (index: number) => void, disabled: boolean }) => {
        return (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-8">
                {options.map((opt, i) => {
                    const theme = PartyThemes[i % PartyThemes.length];
                    return (
                        <button
                            key={i}
                            onClick={() => onSelect(i)}
                            disabled={disabled}
                            className={`
                                ${theme.bg} ${theme.hover}
                                text-white p-6 rounded-2xl text-left font-black text-xl
                                border-b-8 border-r-8 border-black/30
                                shadow-[4px_4px_0px_#000]
                                active:border-b-2 active:border-r-2 active:shadow-none active:translate-y-1 active:translate-x-1
                                transition-all duration-100
                                disabled:opacity-50 disabled:cursor-not-allowed
                                min-h-[100px] flex items-center gap-4
                            `}
                        >
                            <span className="bg-white/20 w-12 h-12 rounded-xl flex items-center justify-center text-2xl flex-shrink-0 shadow-inner">
                                {theme.icon}
                            </span>
                            <span className="flex-1 drop-shadow-md">
                                {opt}
                            </span>
                        </button>
                    );
                })}
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
            // Increment attempt count for skip logic
            setAttemptCount(prev => prev + 1);
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
            } else {
                setGameState('waiting_for_start');
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
        return (
            <div className="relative">
                <div className="absolute -left-4 -top-4 text-4xl opacity-20 -rotate-12 select-none">💬</div>
                <h2 className="text-2xl sm:text-3xl font-black mb-8 leading-tight tracking-tight text-[var(--text-dark)] drop-shadow-sm">
                    {cleanText}
                </h2>
            </div>
        );
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
                    <p className="text-[var(--foreground-muted)]">Loading...</p>
                </div>
            </div>
        );
    }

    // Waiting for room to start
    if (gameState === 'waiting_for_start') {
        return (
            <div className="min-h-screen flex items-center justify-center relative">
                <LobbyBackground />
                <div className="relative z-10 text-center card-party max-w-md animate-bounce-slow">
                    <div className="text-6xl mb-6">⏳</div>
                    <h1 className="text-3xl font-black mb-4">Waiting for Party...</h1>
                    <p className="text-xl font-bold text-[var(--text-muted)]">
                        The host will start the game soon. Get ready!
                    </p>
                </div>
            </div>
        );
    }

    // Done state
    if (gameState === 'done') {
        return (
            <div className="min-h-screen flex items-center justify-center p-6 relative">
                <LobbyBackground />
                <div className="card-party text-center max-w-md animate-slide-up">
                    <div className="text-6xl mb-6 animate-spin-slow">🎉</div>
                    <h1 className="text-4xl font-black mb-4">You're Done!</h1>
                    <p className="text-xl font-bold text-[var(--text-muted)] mb-8">
                        Thanks for playing!
                    </p>
                    <div className="text-5xl font-black text-party-gradient mb-8 drop-shadow-sm">
                        {totalPoints || 0} pts
                    </div>
                    <button
                        onClick={() => router.push('/')}
                        className="btn btn-secondary w-full"
                    >
                        Back to Home
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6 flex flex-col relative overflow-hidden">
            <GameBackground />
            {/* Header */}
            <header className="relative z-10 flex items-center justify-between mb-6">
                <div className="flex items-center gap-4">
                    <span className="badge-party bg-white shadow-sm">Room: {code}</span>
                </div>
                <div className="flex items-center gap-4">
                    <div className="text-right bg-white px-4 py-2 rounded-xl border-2 border-black shadow-[4px_4px_0px_#000]">
                        <div className="text-2xl font-black text-[var(--color-purple)]">{totalPoints || 0}</div>
                        <div className="text-xs font-bold text-[var(--text-muted)] uppercase">points</div>
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
                        className="btn btn-ghost text-xs font-bold"
                    >
                        Leave
                    </button>
                </div>
            </header>

            {/* Question Card */}
            <div className="flex-1 flex items-center justify-center relative z-10">
                <div className="w-full max-w-2xl">
                    {/* Result display */}
                    {gameState === 'evaluated' && result && (
                        <div className={`card-party mb-8 animate-slide-up ${result.resolution === 'SAT'
                            ? 'border-[var(--color-green)]'
                            : 'border-[var(--color-yellow)]'
                            }`} style={{ borderWidth: '4px' }}>
                            <div className="flex items-center gap-3 mb-3">
                                <span className="badge-party text-lg" style={{
                                    borderColor: result.resolution === 'SAT' ? 'var(--color-green)' : 'var(--color-yellow)'
                                }}>
                                    {result.resolution === 'SAT' ? '✓ NAILED IT!' : '⚡ KEEP GOING!'}
                                </span>
                                <span className="text-2xl font-black text-[var(--color-purple)]">
                                    +{result.pointsEarned} pts
                                </span>
                            </div>
                            <p className="text-lg font-bold text-[var(--text-dark)]">{result.evalSummary}</p>
                            {result.followUp && (
                                <p className="text-sm font-bold mt-4 text-[var(--color-blue)] animate-pulse">
                                    Follow-up question incoming...
                                </p>
                            )}
                            {result.nextQuestion && (
                                <p className="text-sm font-bold mt-4 text-[var(--color-green)] animate-pulse">
                                    Next question loading...
                                </p>
                            )}
                        </div>
                    )}

                    {/* Question */}
                    <div className="card-party animate-pop-in">
                        {currentQuestion && (
                            <>
                                <div className="flex items-center gap-2 mb-6">
                                    <span className="badge-party" style={{ borderColor: 'var(--border-color)' }}>#{currentQuestion.key}</span>
                                    <span className="badge-party" style={{ borderColor: 'var(--color-blue)' }}>
                                        {currentQuestion.type === 'ESSAY' ? '📝 Essay' :
                                            currentQuestion.type === 'MCQ' ? '🔘 Choice' : '📊 Rating'}
                                    </span>
                                    {currentQuestion.pointsMax && (
                                        <span className="text-sm font-bold text-[var(--text-muted)] ml-auto">
                                            Up to {currentQuestion.pointsMax} pts
                                        </span>
                                    )}
                                </div>

                                <div className="text-3xl font-black mb-8 leading-tight">
                                    <PromptDisplay text={currentQuestion.prompt} />
                                </div>

                                {/* Essay Input */}
                                {currentQuestion.type === 'ESSAY' && (gameState === 'answering' || gameState === 'waiting_for_ai') && (
                                    <textarea
                                        className="input-party mb-6 text-lg min-h-[200px]"
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
                                    <div className="flex flex-col items-center justify-center py-12 text-[var(--color-purple)]">
                                        <div className="relative mb-6">
                                            <div className="text-6xl animate-bounce">🧠</div>
                                            <div className="absolute -right-2 -top-2 text-2xl animate-pulse">✨</div>
                                        </div>
                                        <div className="flex items-center gap-3">
                                            <div className="spinner" />
                                            <span className="font-black text-2xl tracking-tighter italic">AI IS COOKING...</span>
                                        </div>
                                        <p className="text-sm font-bold mt-4 text-[var(--text-muted)] uppercase tracking-widest">Analyzing your brilliance</p>
                                    </div>
                                )}

                                {/* Degree Slider */}
                                {currentQuestion.type === 'DEGREE' && gameState === 'answering' && (
                                    <div className="mb-8 p-6 bg-[var(--bg-cream)] rounded-xl border-2 border-[var(--border-color)]">
                                        <div className="flex justify-between text-lg font-bold text-[var(--text-muted)] mb-4">
                                            <span>{currentQuestion.scaleMin || 1}</span>
                                            <span className="text-4xl font-black text-[var(--color-purple)]">{degreeValue}</span>
                                            <span>{currentQuestion.scaleMax || 5}</span>
                                        </div>
                                        <input
                                            type="range"
                                            className="w-full h-4 bg-[var(--border-color)] rounded-full appearance-none cursor-pointer"
                                            style={{
                                                background: `linear-gradient(to right, var(--color-pink) 0%, var(--color-purple) ${(degreeValue - (currentQuestion.scaleMin || 1)) / ((currentQuestion.scaleMax || 5) - (currentQuestion.scaleMin || 1)) * 100}%, var(--bg-cream) ${(degreeValue - (currentQuestion.scaleMin || 1)) / ((currentQuestion.scaleMax || 5) - (currentQuestion.scaleMin || 1)) * 100}%, var(--bg-cream) 100%)`
                                            }}
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
                                            className="btn btn-primary flex-1 py-4 text-xl hover:scale-105"
                                        >
                                            {submitting ? (
                                                <span className="flex items-center gap-2">
                                                    <div className="spinner border-white" style={{ width: 24, height: 24 }} />
                                                    Sending...
                                                </span>
                                            ) : (
                                                '🚀 Submit Answer'
                                            )}
                                        </button>
                                    </div>
                                )}

                                {/* Try Again State (Evaluated but UNSAT) */}
                                {gameState === 'evaluated' && result?.resolution === 'UNSAT' && (
                                    <div className="mt-8 flex gap-4">
                                        <button
                                            onClick={() => {
                                                setGameState('answering');
                                                setResult(null);
                                            }}
                                            className="btn btn-primary flex-1 hover:scale-105"
                                        >
                                            🔄 Try Again
                                        </button>

                                        <button
                                            onClick={handleSkip}
                                            className="btn btn-secondary border-dashed"
                                        >
                                            Skip Question
                                        </button>
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
